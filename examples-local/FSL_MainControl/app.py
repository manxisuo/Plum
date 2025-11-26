#!/usr/bin/env python3
import copy
import logging
import os
import threading
import time
import uuid
from typing import Any, Dict, List, Optional

import requests
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import FileResponse, JSONResponse
from fastapi.staticfiles import StaticFiles

logging.basicConfig(level=logging.INFO, format="[%(levelname)s] %(message)s")
logger = logging.getLogger("FSL_MainControl")

DEFAULTS = {
    "ting_count": 4,
    "tings": [
        {
            "id": "usv-1",
            "name": "USV 1",
            "position": {"lat": 30.6820, "lon": 122.5080},
            "speed_mps": 40.0,
            "sonar_range_m": 300.0,
            "suspect_prob": 0.45,
            "confirm_prob": 0.55,
        },
        {
            "id": "usv-2",
            "name": "USV 2",
            "position": {"lat": 30.6815, "lon": 122.5100},
            "speed_mps": 40.0,
            "sonar_range_m": 300.0,
            "suspect_prob": 0.4,
            "confirm_prob": 0.6,
        },
        {
            "id": "usv-3",
            "name": "USV 3",
            "position": {"lat": 30.6825, "lon": 122.5120},
            "speed_mps": 40.0,
            "sonar_range_m": 300.0,
            "suspect_prob": 0.35,
            "confirm_prob": 0.65,
        },
        {
            "id": "usv-4",
            "name": "USV 4",
            "position": {"lat": 30.6810, "lon": 122.5090},
            "speed_mps": 40.0,
            "sonar_range_m": 300.0,
            "suspect_prob": 0.5,
            "confirm_prob": 0.5,
        },
    ],
    "task_area": {
        "top_left": {"lat": 30.6775, "lon": 122.4950},
        "bottom_right": {"lat": 30.6520, "lon": 122.5250},
    },
}

STAGE_DISPLAY = {
    "created": "任务",
    "plan": "作业规划",
    "扫雷": "扫雷",
    "查证": "查证",
    "灭雷": "灭雷",
    "评估": "评估",
}


def _display_stage(stage: str) -> str:
    if not stage:
        return stage
    base = stage.split("_", 1)[0]
    return STAGE_DISPLAY.get(base, STAGE_DISPLAY.get(stage, stage))


def _normalize_ting_count(tings: List[Dict[str, Any]], requested: int) -> int:
    if requested <= 0:
        requested = len(tings)
    if requested <= 0:
        raise ValueError("至少需要一个艇")
    if len(tings) < requested:
        raise ValueError("提供的艇数量不足")
    return requested


class TaskState:
    def __init__(
        self,
        config: Dict[str, Any],
        plan: Dict[str, Any],
        random_seed: int,
        workflow_id: Optional[str],
        stages: Dict[str, bool],
    ):
        self.task_id = config["task_id"]
        self.config = config
        self.plan = plan
        self.random_seed = random_seed
        self.workflow_id = workflow_id
        self.workflow_stages = stages or {}

        self.stage = "sweep_pending"
        self.created_at = time.time()
        self.updated_at = self.created_at

        self.tings = config["tings"]
        self.work_zones = plan["work_zones"]

        self.suspect_mines: List[Dict[str, Any]] = []
        self.confirmed_mines: List[Dict[str, Any]] = []
        self.cleared_mines: List[Dict[str, Any]] = []
        self.destroyed_mines: List[Dict[str, Any]] = []
        self.evaluated_mines: List[Dict[str, Any]] = []
        self.tracks: List[Dict[str, Any]] = []
        
        # 服务调用日志
        self.service_calls: List[Dict[str, Any]] = []

        self.timeline: List[Dict[str, Any]] = [
            {"stage": _display_stage("created"), "timestamp": time.time(), "message": "任务已创建"}
        ]
    def has_stage(self, stage: str) -> bool:
        if stage == "扫雷":
            return True
        return self.workflow_stages.get(stage, False)

    def append_tracks(self, items: List[Dict[str, Any]]):
        if not items:
            return
        self.tracks.extend(items)

    def record_event(self, stage: str, message: str):
        self.timeline.append(
            {
                "stage": _display_stage(stage),
                "timestamp": time.time(),
                "message": message,
            }
        )
        self.updated_at = time.time()
    
    def record_service_call(
        self,
        service_name: str,
        endpoint: str,
        method: str,
        request_data: Optional[Dict[str, Any]] = None,
        response_data: Optional[Dict[str, Any]] = None,
        status_code: Optional[int] = None,
        error: Optional[str] = None,
        duration_ms: Optional[float] = None,
        endpoint_info: Optional[Dict[str, Any]] = None,
    ):
        """记录服务调用"""
        call_info = {
            "service_name": service_name,
            "endpoint": endpoint,
            "method": method,
            "timestamp": time.time(),
            "request": request_data,
            "response": response_data,
            "status_code": status_code,
            "error": error,
            "duration_ms": duration_ms,
            "endpoint_info": endpoint_info,  # 包含 ip, port, nodeId, instanceId 等信息
        }
        self.service_calls.append(call_info)
        self.updated_at = time.time()


# _normalize_stage_name 函数已移除，直接使用中文阶段名称

def _fetch_workflow_stages(workflow_id: Optional[str]) -> Dict[str, bool]:
    stages = {"扫雷": True, "查证": False, "灭雷": False, "评估": False}
    if not workflow_id:
        return stages
    controller = get_controller_base()
    try:
        resp = requests.get(f"{controller}/v1/dag/workflows/{workflow_id}", timeout=5)
        if resp.status_code != 200:
            logger.warning("获取工作流详情失败(%s): %s", workflow_id, resp.text)
            return stages
        data = resp.json()
        nodes = data.get("nodes") or data.get("Nodes") or []
        task_defs = data.get("taskDefinitions") or data.get("TaskDefinitions") or []
        task_def_map: Dict[str, Dict[str, Any]] = {}
        for item in task_defs:
            def_id = (
                item.get("defId")
                or item.get("DefID")
                or item.get("definitionId")
                or item.get("DefinitionID")
            )
            if def_id:
                task_def_map[str(def_id)] = item

        for node in nodes:
            candidates: List[str] = []
            labels = node.get("labels") or node.get("Labels") or {}
            if isinstance(labels, dict):
                for key in ("workflow.stage", "stage"):
                    value = labels.get(key)
                    if value:
                        candidates.append(value)

            for key in ("name", "Name", "taskName", "TaskName", "title", "Title"):
                value = node.get(key)
                if value:
                    candidates.append(value)

            task_def_id = node.get("taskDefId") or node.get("TaskDefID") or node.get("task_def_id")
            task_def = None
            if task_def_id:
                task_def = task_def_map.get(str(task_def_id))

            if task_def:
                td_labels = task_def.get("labels") or task_def.get("Labels") or {}
                if isinstance(td_labels, dict):
                    for key in ("workflow.stage", "stage"):
                        value = td_labels.get(key)
                        if value:
                            candidates.append(value)
                for key in ("name", "Name", "taskName", "TaskName"):
                    value = task_def.get(key)
                    if value:
                        candidates.append(value)

            stage = ""
            for cand in candidates:
                # 直接识别中文阶段名称
                cand_str = str(cand).strip()
                if cand_str in ("扫雷", "查证", "调查", "灭雷", "摧毁", "评估", "评价"):
                    # 标准化为中文名称
                    if cand_str in ("调查", "查证"):
                        stage = "查证"
                    elif cand_str in ("摧毁", "灭雷"):
                        stage = "灭雷"
                    elif cand_str in ("评价", "评估"):
                        stage = "评估"
                    else:
                        stage = cand_str
                    break

            if stage and stage in stages:
                stages[stage] = True
    except Exception as exc:
        logger.warning("解析工作流阶段失败(%s): %s", workflow_id, exc)
    return stages


class TaskManager:
    def __init__(self):
        self.tasks: Dict[str, TaskState] = {}
        self.lock = threading.Lock()

    def _get_task_locked(self, task_id: str) -> TaskState:
        task = self.tasks.get(task_id)
        if not task:
            raise HTTPException(status_code=404, detail="任务不存在")
        return task

    def create_task(
        self, 
        payload: Dict[str, Any], 
        plan_service_url: str,
        plan_endpoint_info: Optional[Dict[str, Any]] = None,
    ) -> TaskState:
        ting_count = _normalize_ting_count(payload["tings"], payload.get("ting_count", 0))
        plan_payload = {
            "ting_count": ting_count,
            "task_area": payload["task_area"],
        }

        # 记录服务调用开始
        call_start = time.time()
        logger.info("请求 FSL_Plan：%s", plan_payload)
        
        try:
            resp = requests.post(f"{plan_service_url}/planArea", json=plan_payload, timeout=5)
            call_duration = (time.time() - call_start) * 1000  # 转换为毫秒
            
            if resp.status_code != 200:
                # 记录失败的调用
                task_id = str(uuid.uuid4())
                state = TaskState(
                    config={**payload, "task_id": task_id, "ting_count": ting_count},
                    plan={"work_zones": []},
                    random_seed=0,
                    workflow_id=payload.get("workflow_id"),
                    stages={},
                )
                state.record_service_call(
                    service_name="FSL_Plan",
                    endpoint="/planArea",
                    method="POST",
                    request_data=plan_payload,
                    response_data={"error": resp.text},
                    status_code=resp.status_code,
                    duration_ms=call_duration,
                    endpoint_info=plan_endpoint_info,
                )
                raise HTTPException(status_code=502, detail=f"planArea 调用失败: {resp.text}")

            plan = resp.json()
            random_seed = int(time.time() * 1000) & 0xFFFFFFFF

            task_id = str(uuid.uuid4())
            workflow_id = payload.get("workflow_id")
            workflow_stages = _fetch_workflow_stages(workflow_id)
            config = {**payload, "task_id": task_id, "ting_count": ting_count}

            state = TaskState(
                config=config,
                plan=plan,
                random_seed=random_seed,
                workflow_id=workflow_id,
                stages=workflow_stages,
            )
            
            # 记录成功的服务调用
            state.record_service_call(
                service_name="FSL_Plan",
                endpoint="/planArea",
                method="POST",
                request_data=plan_payload,
                response_data=plan,
                status_code=resp.status_code,
                duration_ms=call_duration,
                endpoint_info=plan_endpoint_info,
            )
            state.record_event("plan", "完成作业区划分")
        except requests.exceptions.ConnectionError as e:
            # 连接错误：服务不可用，返回友好的错误信息
            call_duration = (time.time() - call_start) * 1000
            error_msg = f"无法连接到 FSL_Plan 服务 ({plan_service_url})，请确保服务已启动并注册"
            logger.error(f"{error_msg}: {e}")
            raise HTTPException(
                status_code=503,
                detail=error_msg
            )
        except Exception as e:
            # 记录其他异常
            call_duration = (time.time() - call_start) * 1000
            task_id = str(uuid.uuid4())
            state = TaskState(
                config={**payload, "task_id": task_id, "ting_count": ting_count},
                plan={"work_zones": []},
                random_seed=0,
                workflow_id=payload.get("workflow_id"),
                stages={},
            )
            state.record_service_call(
                service_name="FSL_Plan",
                endpoint="/planArea",
                method="POST",
                request_data=plan_payload,
                error=str(e),
                duration_ms=call_duration,
                endpoint_info=plan_endpoint_info,
            )
            logger.error(f"FSL_Plan 调用异常: {e}")
            raise HTTPException(
                status_code=502,
                detail=f"FSL_Plan 服务调用失败: {str(e)}"
            )

        with self.lock:
            self.tasks[task_id] = state

        return state

    def get_task(self, task_id: str) -> TaskState:
        with self.lock:
            return self._get_task_locked(task_id)

    def begin_stage(self, task_id: str, stage: str):
        with self.lock:
            task = self._get_task_locked(task_id)
            task.stage = f"{stage}_running"
            task.record_event(stage, f"{_display_stage(stage)} 阶段开始")

    def update_progress(self, task_id: str, stage: str, payload: Dict[str, Any]):
        with self.lock:
            task = self._get_task_locked(task_id)
            if "tings" in payload:
                task.tings = payload["tings"]
            if "tracks" in payload:
                task.append_tracks(payload["tracks"])
            if "suspect_mines" in payload:
                task.suspect_mines = payload["suspect_mines"]
            if "confirmed_mines" in payload:
                task.confirmed_mines = payload["confirmed_mines"]
            if "cleared_mines" in payload:
                task.cleared_mines = payload["cleared_mines"]
            if "destroyed_mines" in payload:
                task.destroyed_mines = payload["destroyed_mines"]
            if "evaluated_mines" in payload:
                task.evaluated_mines = payload["evaluated_mines"]
                if task.destroyed_mines:
                    eval_map = {item.get("id"): item for item in task.evaluated_mines if item.get("id")}
                    for mine in task.destroyed_mines:
                        mid = mine.get("id")
                        if mid and mid in eval_map:
                            score = eval_map[mid].get("evaluation_score")
                            if score is not None:
                                mine["evaluation_score"] = score
            task.stage = f"{stage}_running"
            task.updated_at = time.time()

    def finish_stage(self, task_id: str, stage: str, result: Dict[str, Any]):
        with self.lock:
            task = self._get_task_locked(task_id)
            if stage == "扫雷":
                task.suspect_mines = copy.deepcopy(result.get("suspect_mines", []))
                task.confirmed_mines = copy.deepcopy(result.get("confirmed_mines", []))
                task.append_tracks(result.get("tracks", []))
                task.tings = result.get("tings", task.tings)

                has_investigate = task.has_stage("查证")
                has_destroy = task.has_stage("灭雷")

                suspects_available = bool(task.suspect_mines)
                confirmed_available = bool(task.confirmed_mines)
                stage_name = _display_stage(stage)

                # 优先检查是否有疑似水雷需要查证
                if suspects_available and has_investigate:
                    task.stage = "investigate_pending"
                    task.record_event(stage, f"{stage_name} 完成")
                # 如果配置了查证阶段，即使只有确认水雷，也应该进入查证（让查证阶段决定后续流程）
                elif has_investigate:
                    task.stage = "investigate_pending"
                    task.record_event(stage, f"{stage_name} 完成")
                # 如果没有查证阶段，但有确认水雷，检查是否有灭雷阶段
                elif confirmed_available:
                    if has_destroy:
                        task.stage = "destroy_pending"
                        task.record_event(stage, f"{stage_name} 完成")
                    else:
                        task.stage = "completed"
                        task.record_event(stage, f"{stage_name} 完成")
                        # 任务完成，调用统计服务
                        _call_statistics_service(task)
                # 如果只有疑似水雷但没有查证阶段，但有灭雷阶段，则提升为确认并进入灭雷
                elif suspects_available and not has_investigate and has_destroy:
                    promoted = []
                    for mine in task.suspect_mines:
                        promoted_mine = copy.deepcopy(mine)
                        promoted_mine["status"] = "confirmed"
                        promoted.append(promoted_mine)
                    task.confirmed_mines.extend(promoted)
                    task.suspect_mines = []
                    task.stage = "destroy_pending"
                    task.record_event(stage, f"{stage_name} 完成")
                # 如果只有疑似水雷但没有后续阶段
                elif suspects_available and not has_investigate and not has_destroy:
                    task.stage = "completed"
                    task.record_event(stage, f"{stage_name} 完成")
                    # 任务完成，调用统计服务
                    _call_statistics_service(task)
                # 没有发现任何水雷
                else:
                    task.stage = "completed"
                    task.record_event(stage, f"{stage_name} 完成")
                    # 任务完成，调用统计服务
                    _call_statistics_service(task)

            elif stage == "查证":
                task.confirmed_mines = result.get("confirmed_mines", [])
                task.cleared_mines = result.get("cleared_mines", [])
                task.append_tracks(result.get("tracks", []))
                task.tings = result.get("tings", task.tings)

                stage_name = _display_stage(stage)

                # 如果配置了灭雷阶段，无论是否有确认水雷，都应该进入灭雷（让灭雷阶段决定后续流程）
                if task.has_stage("灭雷"):
                    task.stage = "destroy_pending"
                    task.record_event(stage, f"{stage_name} 完成")
                # 如果没有配置灭雷阶段，任务结束
                else:
                    task.stage = "completed"
                    task.record_event(stage, f"{stage_name} 完成")
                    # 任务完成，调用统计服务
                    _call_statistics_service(task)

            elif stage == "灭雷":
                task.destroyed_mines = result.get("destroyed_mines", [])
                task.append_tracks(result.get("tracks", []))
                task.tings = result.get("tings", task.tings)
                stage_name = _display_stage(stage)
                if task.destroyed_mines and task.has_stage("评估"):
                    task.evaluated_mines = []
                    task.stage = "evaluate_pending"
                    task.record_event(stage, f"{stage_name} 完成")
                else:
                    task.stage = "completed"
                    task.record_event(stage, f"{stage_name} 完成")
                    # 任务完成，调用统计服务
                    _call_statistics_service(task)

            elif stage == "评估":
                task.evaluated_mines = result.get("evaluated_mines", [])
                destroyed = result.get("destroyed_mines", [])
                if destroyed:
                    task.destroyed_mines = destroyed
                else:
                    eval_map = {item.get("id"): item.get("evaluation_score") for item in task.evaluated_mines}
                    for mine in task.destroyed_mines:
                        score = eval_map.get(mine.get("id"))
                        if score is not None:
                            mine["evaluation_score"] = score
                task.append_tracks(result.get("tracks", []))
                task.tings = result.get("tings", task.tings)
                task.stage = "completed"
                task.record_event(stage, f"{_display_stage(stage)} 完成")
                # 任务完成，调用统计服务
                _call_statistics_service(task)

            else:
                raise HTTPException(status_code=400, detail=f"未知阶段: {stage}")

    def fail_stage(self, task_id: str, stage: str, message: str):
        with self.lock:
            task = self._get_task_locked(task_id)
            task.stage = f"{stage}_failed"
            task.record_event(stage, f"{_display_stage(stage)} 阶段失败: {message or '未知错误'}")


task_manager = TaskManager()

app = FastAPI(title="FSL Main Control")
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
STATIC_DIR = os.path.join(BASE_DIR, "static")
app.mount("/static", StaticFiles(directory=STATIC_DIR), name="static")


def _discover_service(service_name: str, default_url: str) -> tuple[str, Optional[Dict[str, Any]]]:
    """使用服务发现 API 获取服务地址（lazy 模式）
    
    注意：lazy 模式的缓存由 Controller 端实现，客户端直接调用 API 即可。
    Controller 会返回缓存的端点（如果仍然可用），否则返回新的端点并更新缓存。
    
    返回: (service_url, endpoint_info)
    - service_url: 服务地址（如 "http://127.0.0.1:4100"）
    - endpoint_info: 端点详细信息（包含 ip, port, nodeId, instanceId 等），如果服务发现失败则为 None
    """
    controller_url = get_controller_base()
    try:
        response = requests.get(
            f"{controller_url}/v1/discovery/one",
            params={"service": service_name, "strategy": "lazy"},
            timeout=3,
        )
        if response.status_code == 200:
            endpoint = response.json()
            if isinstance(endpoint, dict):
                ip = endpoint.get("ip", "localhost")
                port = endpoint.get("port", 0)
                protocol = endpoint.get("protocol", "http")
                
                if port:
                    service_url = f"{protocol}://{ip}:{port}"
                    endpoint_info = {
                        "ip": ip,
                        "port": port,
                        "protocol": protocol,
                        "nodeId": endpoint.get("nodeId", ""),
                        "instanceId": endpoint.get("instanceId", ""),
                        "serviceName": endpoint.get("serviceName", service_name),
                        "healthy": endpoint.get("healthy", False),
                    }
                    logger.info(
                        f"发现服务 {service_name}: {service_url} "
                        f"(节点: {endpoint_info['nodeId']}, "
                        f"实例: {endpoint_info['instanceId']})"
                    )
                    return service_url, endpoint_info
                else:
                    logger.warning(f"服务 {service_name} 返回的端口无效: {endpoint}")
            else:
                logger.warning(f"服务 {service_name} 返回未知格式: {endpoint}")
        elif response.status_code == 404:
            logger.warning(f"服务 {service_name} 当前未发现可用端点 (404)，使用默认地址: {default_url}")
        else:
            logger.warning(
                f"服务发现接口返回 {response.status_code} ({service_name}): {response.text}，"
                f"使用默认地址: {default_url}"
            )
    except Exception as e:
        logger.warning(f"无法发现服务 {service_name}: {e}，使用默认地址: {default_url}")
    
    # 如果服务发现失败，使用默认地址
    return default_url, None


def get_plan_service_url() -> tuple[str, Optional[Dict[str, Any]]]:
    """获取 FSL_Plan 服务地址（通过服务发现）
    
    返回: (service_url, endpoint_info)
    """
    return _discover_service("planArea", "http://127.0.0.1:4100")


def get_statistics_service_url() -> tuple[str, Optional[Dict[str, Any]]]:
    """获取 FSL_Statistics 服务地址（通过服务发现）
    
    返回: (service_url, endpoint_info)
    """
    return _discover_service("analyzeTask", "http://127.0.0.1:4102")


def _call_statistics_service(task: TaskState):
    """在任务完成时自动调用统计服务"""
    try:
        statistics_url, statistics_endpoint_info = get_statistics_service_url()
        call_start = time.time()
        stats_payload = {
            "task_id": task.task_id,
            "stage": task.stage,
            "tings": task.tings,
            "suspect_mines": task.suspect_mines,
            "confirmed_mines": task.confirmed_mines,
            "cleared_mines": task.cleared_mines,
            "destroyed_mines": task.destroyed_mines,
            "evaluated_mines": task.evaluated_mines,
            "tracks": task.tracks,
            "timeline": task.timeline,
            "created_at": task.created_at,
            "updated_at": task.updated_at,
        }
        logger.info("任务完成，自动调用 FSL_Statistics：task_id=%s", task.task_id)
        resp = requests.post(f"{statistics_url}/analyze", json=stats_payload, timeout=5)
        call_duration = (time.time() - call_start) * 1000
        
        if resp.status_code == 200:
            stats_result = resp.json()
            # 记录服务调用
            task.record_service_call(
                service_name="FSL_Statistics",
                endpoint="/analyze",
                method="POST",
                request_data=stats_payload,
                response_data=stats_result,
                status_code=resp.status_code,
                duration_ms=call_duration,
                endpoint_info=statistics_endpoint_info,
            )
            logger.info("统计服务调用成功：task_id=%s", task.task_id)
        else:
            # 记录失败的调用
            task.record_service_call(
                service_name="FSL_Statistics",
                endpoint="/analyze",
                method="POST",
                request_data=stats_payload,
                response_data={"error": resp.text},
                status_code=resp.status_code,
                duration_ms=call_duration,
                endpoint_info=statistics_endpoint_info,
            )
            logger.warning("统计服务调用失败：task_id=%s, status=%d", task.task_id, resp.status_code)
    except Exception as e:
        logger.warning("统计服务调用异常：task_id=%s, error=%s", task.task_id, e)
        # 不抛出异常，避免影响任务完成流程


def get_controller_base() -> str:
    return os.environ.get("CONTROLLER_BASE", "http://127.0.0.1:8080")


def compose_stage_payload(task: TaskState, stage: str) -> Dict[str, Any]:
    payload: Dict[str, Any] = {
        "task_id": task.task_id,
        "stage": stage,
        "random_seed": task.random_seed,
        "tings": copy.deepcopy(task.tings),
        "suspect_mines": copy.deepcopy(task.suspect_mines),
        "confirmed_mines": copy.deepcopy(task.confirmed_mines),
        "destroyed_mines": copy.deepcopy(task.destroyed_mines),
        "evaluated_mines": copy.deepcopy(task.evaluated_mines),
    }
    if stage == "扫雷":
        payload["work_zones"] = copy.deepcopy(task.work_zones)
        payload["plan"] = copy.deepcopy(task.plan)
    elif stage not in {"查证", "灭雷", "评估"}:
        raise HTTPException(status_code=400, detail=f"未知阶段: {stage}")
    return payload


@app.get("/api/workflows")
def api_list_workflows():
    controller = get_controller_base()
    try:
        resp = requests.get(f"{controller}/v1/dag/workflows", timeout=5)
    except Exception as exc:
        logger.error("获取工作流失败: %s", exc)
        raise HTTPException(status_code=502, detail=f"请求 Controller 失败: {exc}")
    if resp.status_code != 200:
        raise HTTPException(status_code=resp.status_code, detail=resp.text)
    return resp.json()


@app.get("/api/workflows/{workflow_id}/runs")
def api_list_workflow_runs(workflow_id: str):
    controller = get_controller_base()
    try:
        resp = requests.get(f"{controller}/v1/dag/workflows/{workflow_id}/runs", timeout=5)
    except Exception as exc:
        logger.error("获取工作流运行失败: %s", exc)
        raise HTTPException(status_code=502, detail=f"请求 Controller 失败: {exc}")
    if resp.status_code != 200:
        raise HTTPException(status_code=resp.status_code, detail=resp.text)
    return resp.json()


@app.get("/api/workflows/{workflow_id}/runs/{run_id}/status")
def api_workflow_run_status(workflow_id: str, run_id: str):
    controller = get_controller_base()
    try:
        resp = requests.get(f"{controller}/v1/dag/runs/{run_id}/status", timeout=5)
    except Exception as exc:
        logger.error("获取工作流运行状态失败: %s", exc)
        raise HTTPException(status_code=502, detail=f"请求 Controller 失败: {exc}")
    if resp.status_code != 200:
        raise HTTPException(status_code=resp.status_code, detail=resp.text)
    return resp.json()


@app.post("/api/workflows/{workflow_id}/run")
def api_run_workflow(workflow_id: str, payload: Optional[Dict[str, Any]] = None):
    controller = get_controller_base()
    body = payload or {}
    try:
        resp = requests.post(
            f"{controller}/v1/dag/workflows/{workflow_id}/run",
            json=body,
            timeout=5,
        )
    except Exception as exc:
        logger.error("触发工作流运行失败: %s", exc)
        raise HTTPException(status_code=502, detail=f"请求 Controller 失败: {exc}")
    if resp.status_code != 200:
        raise HTTPException(status_code=resp.status_code, detail=resp.text)
    return resp.json()


@app.get("/")
def index():
    return FileResponse(os.path.join(STATIC_DIR, "index.html"))


@app.get("/healthz")
def healthz():
    """健康检查端点"""
    return {"status": "ok"}


@app.get("/api/config/defaults")
def api_defaults():
    return JSONResponse(DEFAULTS)


@app.post("/api/task/start")
def api_task_start(payload: Dict[str, Any]):
    required = ["tings", "task_area"]
    for field in required:
        if field not in payload:
            raise HTTPException(status_code=400, detail=f"缺少字段: {field}")

    # 创建任务（调用 FSL_Plan）
    plan_service_url, plan_endpoint_info = get_plan_service_url()
    task = task_manager.create_task(payload, plan_service_url=plan_service_url, plan_endpoint_info=plan_endpoint_info)
    
    response = {
        "task_id": task.task_id,
        "stage": task.stage,
        "扫雷_payload": compose_stage_payload(task, "扫雷"),
    }
    return JSONResponse(response)


@app.get("/api/task/{task_id}/stage/{stage}/input")
def api_stage_input(task_id: str, stage: str):
    task = task_manager.get_task(task_id)
    allowed = {"扫雷", "查证", "灭雷", "评估"}
    if stage not in allowed:
        raise HTTPException(status_code=400, detail=f"不支持的阶段: {stage}")
    if stage == "查证" and not task.suspect_mines:
        raise HTTPException(status_code=409, detail="当前没有疑似水雷")
    if stage == "灭雷" and not task.confirmed_mines:
        raise HTTPException(status_code=409, detail="当前没有确认水雷")
    if stage == "评估" and not task.destroyed_mines:
        raise HTTPException(status_code=409, detail="当前没有已销毁水雷")
    return JSONResponse(compose_stage_payload(task, stage))


@app.post("/api/task/{task_id}/stage/{stage}/begin")
def api_stage_begin(task_id: str, stage: str):
    task_manager.begin_stage(task_id, stage)
    return JSONResponse({"status": "ok"})


@app.post("/api/task/{task_id}/stage/{stage}/progress")
def api_stage_progress(task_id: str, stage: str, payload: Dict[str, Any]):
    task_manager.update_progress(task_id, stage, payload)
    return JSONResponse({"status": "ok"})


@app.post("/api/task/{task_id}/stage/{stage}/result")
def api_stage_result(task_id: str, stage: str, payload: Dict[str, Any]):
    if payload.get("status") == "error":
        message = payload.get("message", "阶段执行失败")
        logger.error("阶段 %s 执行失败: %s", stage, message)
        task_manager.fail_stage(task_id, stage, message)
        return JSONResponse({"status": "error", "message": message})

    task_manager.finish_stage(task_id, stage, payload)
    task = task_manager.get_task(task_id)

    response: Dict[str, Any] = {
        "task_id": task_id,
        "stage": task.stage,
    }
    if task.stage == "investigate_pending":
        response["next_payload"] = compose_stage_payload(task, "查证")
    elif task.stage == "destroy_pending":
        response["next_payload"] = compose_stage_payload(task, "灭雷")
    elif task.stage == "evaluate_pending":
        response["next_payload"] = compose_stage_payload(task, "评估")
    return JSONResponse(response)


@app.get("/api/status")
def api_status(task_id: Optional[str] = None):
    if not task_id:
        return JSONResponse({"tasks": list(task_manager.tasks.keys())})
    task = task_manager.get_task(task_id)
    response = {
        "task_id": task.task_id,
        "stage": task.stage,
        "config": task.config,
        "plan": task.plan,
        "tings": task.tings,
        "suspect_mines": task.suspect_mines,
        "confirmed_mines": task.confirmed_mines,
        "cleared_mines": task.cleared_mines,
        "destroyed_mines": task.destroyed_mines,
        "evaluated_mines": task.evaluated_mines,
        "tracks": task.tracks,
        "timeline": task.timeline,
        "service_calls": task.service_calls,
        "created_at": task.created_at,
        "updated_at": task.updated_at,
    }
    return JSONResponse(response)


@app.get("/api/task/{task_id}/service-calls")
def api_service_calls(task_id: str):
    """获取任务的服务调用日志"""
    task = task_manager.get_task(task_id)
    return JSONResponse({
        "task_id": task_id,
        "service_calls": task.service_calls,
        "total_calls": len(task.service_calls),
    })


@app.get("/api/task/{task_id}/statistics")
def api_task_statistics(task_id: str):
    """获取任务统计数据"""
    task = task_manager.get_task(task_id)
    
    # 调用统计服务
    statistics_url = get_statistics_service_url()
    try:
        call_start = time.time()
        stats_payload = {
            "task_id": task.task_id,
            "stage": task.stage,
            "tings": task.tings,
            "suspect_mines": task.suspect_mines,
            "confirmed_mines": task.confirmed_mines,
            "cleared_mines": task.cleared_mines,
            "destroyed_mines": task.destroyed_mines,
            "evaluated_mines": task.evaluated_mines,
            "tracks": task.tracks,
            "timeline": task.timeline,
            "created_at": task.created_at,
            "updated_at": task.updated_at,
        }
        logger.info("请求 FSL_Statistics：task_id=%s", task.task_id)
        resp = requests.post(f"{statistics_url}/analyze", json=stats_payload, timeout=5)
        call_duration = (time.time() - call_start) * 1000
        
        if resp.status_code == 200:
            stats_result = resp.json()
            # 记录服务调用
            task.record_service_call(
                service_name="FSL_Statistics",
                endpoint="/analyze",
                method="POST",
                request_data=stats_payload,
                response_data=stats_result,
                status_code=resp.status_code,
                duration_ms=call_duration,
            )
            return JSONResponse(stats_result)
        else:
            # 记录失败的调用
            task.record_service_call(
                service_name="FSL_Statistics",
                endpoint="/analyze",
                method="POST",
                request_data=stats_payload,
                response_data={"error": resp.text},
                status_code=resp.status_code,
                duration_ms=call_duration,
            )
            raise HTTPException(status_code=502, detail=f"统计服务调用失败: {resp.text}")
    except requests.exceptions.RequestException as e:
        logger.warning("统计服务不可用: %s", e)
        raise HTTPException(status_code=502, detail=f"统计服务不可用: {e}")
    except HTTPException:
        raise
    except Exception as e:
        logger.error("统计服务调用异常: %s", e)
        raise HTTPException(status_code=500, detail=f"统计服务调用异常: {e}")


if __name__ == "__main__":
    import uvicorn

    host = os.environ.get("MAINCONTROL_HOST", "0.0.0.0")
    port = int(os.environ.get("MAINCONTROL_PORT", "4000"))
    logger.info("MainControl 服务启动：%s:%s", host, port)
    uvicorn.run(app, host=host, port=port)


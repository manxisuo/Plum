#!/usr/bin/env python3
"""
简易流程编排脚本：
  1. 调用 MainControl /api/task/start 启动任务，得到 sweep payload
  2. 依次调用 FSL_Sweep → FSL_Investigate → FSL_Destroy Worker
  3. 将阶段结果通过 /api/task/{task_id}/stage/{stage}/result 回传

需要确保：
  - FSL_MainControl 运行在 http://127.0.0.1:4000
  - FSL_Plan/FSL_Sweep/FSL_Investigate/FSL_Destroy 均已启动并可访问
"""
import json
import os
import sys
import time
from typing import Dict

import requests

MAINCONTROL_BASE = os.environ.get("MAINCONTROL_BASE", "http://127.0.0.1:4000")

WORKER_ENDPOINTS = {
    "sweep": os.environ.get("FSL_SWEEP_ENDPOINT", "http://127.0.0.1:7001/task"),
    "investigate": os.environ.get("FSL_INVESTIGATE_ENDPOINT", "http://127.0.0.1:7002/task"),
    "destroy": os.environ.get("FSL_DESTROY_ENDPOINT", "http://127.0.0.1:7003/task"),
}


def post_json(url: str, payload: Dict) -> Dict:
    resp = requests.post(url, json=payload, timeout=10)
    if resp.status_code != 200:
        raise RuntimeError(f"调用 {url} 失败：{resp.status_code} {resp.text}")
    return resp.json()


def invoke_worker(stage: str, payload: Dict) -> Dict:
    endpoint = WORKER_ENDPOINTS.get(stage)
    if not endpoint:
        raise RuntimeError(f"未配置 {stage} worker endpoint")
    print(f"[{stage}] 调用 {endpoint}")
    data = {
        "taskName": f"fsl.{stage}",
        "payload": json.dumps(payload),
    }
    return post_json(endpoint, data)


def main():
    print("=== FSL 编排流程 ===")
    defaults = requests.get(f"{MAINCONTROL_BASE}/api/config/defaults", timeout=5).json()

    print("1) 启动任务")
    start_resp = post_json(f"{MAINCONTROL_BASE}/api/task/start", defaults)
    task_id = start_resp["task_id"]
    print(f"任务 ID: {task_id}")

    stage_sequence = ["sweep", "investigate", "destroy"]
    next_payload = start_resp.get("sweep_payload")

    for stage in stage_sequence:
        if not next_payload:
            print(f"[{stage}] 无待执行 payload，跳过")
            continue

        print(f"\n[{stage}] 阶段开始")
        post_json(f"{MAINCONTROL_BASE}/api/task/{task_id}/stage/{stage}/begin", {})

        worker_resp = invoke_worker(stage, next_payload)

        if worker_resp.get("status") == "error":
            raise RuntimeError(f"{stage} worker 执行失败: {worker_resp.get('message')}")

        print(f"[{stage}] 回传阶段结果至 MainControl")
        report_resp = post_json(
            f"{MAINCONTROL_BASE}/api/task/{task_id}/stage/{stage}/result",
            worker_resp,
        )

        next_payload = report_resp.get("next_payload")
        time.sleep(1)

        if report_resp.get("stage") == "completed":
            print("任务流程已完成")
            break

    status = requests.get(f"{MAINCONTROL_BASE}/api/status", params={"task_id": task_id}, timeout=5).json()
    print("\n=== 任务最终状态 ===")
    print(json.dumps(status, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"流程执行异常: {exc}", file=sys.stderr)
        sys.exit(1)


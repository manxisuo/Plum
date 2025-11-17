# FSL_MainControl 任务执行流程详解

本文档详细说明 FSL_MainControl 执行一个任务的整个过程中，前端、后端、Plum Controller、Plum Worker 之间的数据流和 API 调用关系。

## 架构组件

- **前端（Web UI）**：`examples-local/FSL_MainControl/static/app.js` + `index.html`
- **MainControl 后端**：`examples-local/FSL_MainControl/app.py` (FastAPI)
- **Plum Controller**：`controller/` (Go，DAG 执行器 + gRPC 服务器)
- **Plum Worker**：`examples-local/FSL_Sweep/main.cpp` 等（C++，通过 gRPC 连接 Controller）
- **FSL_Plan 服务**：独立的 HTTP 服务（作业区规划）

## 完整流程

### 阶段 1：任务启动（前端 → MainControl 后端 → FSL_Plan）

```
1. 用户在前端点击"开始任务"
   ↓
2. 前端调用：POST /api/task/start
   Body: { tings, task_area, workflow_id, ... }
   ↓
3. MainControl 后端：
   - 调用 FSL_Plan 服务：POST {plan_service_url}/planArea
   - 创建 TaskState 对象（内存中）
   - 返回：{ task_id, stage, 扫雷_payload }
   ↓
4. 前端收到响应，保存 task_id
   ↓
5. 前端调用：POST /api/workflows/{workflowId}/run
   Body: {
     taskPayload: {
       taskId: <task_id>,
       sweepPayload: <扫雷_payload>,
       stageControlBase: window.location.origin  // MainControl 地址
     }
   }
```

### 阶段 2：工作流调度（Controller DAG 执行器）

```
6. Controller 接收工作流运行请求
   ↓
7. DAG 执行器开始执行工作流
   ↓
8. 对于每个阶段节点（如"扫雷"）：
   a) 调用 MainControl：GET /api/task/{task_id}/stage/{stage}/input
      - 获取阶段输入 payload
      - MainControl 返回：{ task_id, stage, tings, work_zones, ... }
   ↓
   b) 调用 MainControl：POST /api/task/{task_id}/stage/{stage}/begin
      - 通知阶段开始
      - MainControl 更新任务状态：stage = "{stage}_running"
   ↓
   c) Controller 创建任务（Task）：
      - Executor: "embedded"（gRPC Worker）
      - PayloadJSON: <从 MainControl 获取的 payload>
      - State: "Pending"
   ↓
   d) Controller 通过 gRPC 推送任务给 Worker
```

### 阶段 3：Worker 执行任务（Worker ↔ MainControl）

```
9. Worker 通过 gRPC 连接到 Controller
   - Worker 发送注册信息：WorkerRegister
   - Controller 保存 Worker 连接
   ↓
10. Controller 通过 gRPC 流推送任务：
    TaskRequest {
      task_id: <controller_task_id>,
      name: "扫雷",
      payload: <JSON 字符串>
    }
    ↓
11. Worker 接收任务，开始执行：
    - 解析 payload
    - 创建 StageProgressSender（读取 MAIN_CONTROL_BASE 环境变量）
    - 进入执行循环
    ↓
12. Worker 执行过程中（每 150ms，每 4 个轨迹点）：
    调用 MainControl：POST /api/task/{task_id}/stage/{stage}/progress
    Body: {
      tings: [...],
      tracks: [...],
      suspect_mines: [...],
      confirmed_mines: [...]
    }
    ↓
    MainControl 更新任务状态：
    - task.stage = "{stage}_running"
    - task.tracks.append(...)
    - task.suspect_mines = ...
    - task.updated_at = time.time()
    ↓
13. Worker 执行完成：
    通过 gRPC 返回结果给 Controller：
    TaskResponse {
      task_id: <controller_task_id>,
      result: <JSON 字符串>,
      error: ""
    }
```

### 阶段 4：阶段完成通知（Controller → MainControl）

```
14. Controller 接收 Worker 结果
    ↓
15. Controller 调用 MainControl：POST /api/task/{task_id}/stage/{stage}/result
    Body: {
      status: "success",
      tings: [...],
      suspect_mines: [...],
      confirmed_mines: [...],
      tracks: [...]
    }
    ↓
16. MainControl 处理阶段结果：
    - task_manager.finish_stage(task_id, stage, result)
    - 更新任务状态：
      * task.suspect_mines = result.suspect_mines
      * task.confirmed_mines = result.confirmed_mines
      * task.tracks.append(result.tracks)
    - 决定下一个阶段：
      * 如果有疑似水雷 → stage = "investigate_pending"
      * 如果有确认水雷 → stage = "destroy_pending"
      * 否则 → stage = "completed"
    ↓
17. MainControl 返回响应：
    {
      task_id: <task_id>,
      stage: <next_stage>,
      next_payload: <下一阶段的 payload>（如果有）
    }
```

### 阶段 5：前端状态轮询（前端 ↔ MainControl）

```
18. 前端启动轮询（每秒一次）：
    setInterval(() => {
      GET /api/status?task_id={task_id}
    }, 1000)
    ↓
19. MainControl 返回任务状态：
    {
      task_id: <task_id>,
      stage: <current_stage>,
      tings: [...],
      suspect_mines: [...],
      confirmed_mines: [...],
      destroyed_mines: [...],
      tracks: [...],
      timeline: [...],
      updated_at: <timestamp>
    }
    ↓
20. 前端更新 UI：
    - 渲染地图（tracks, mines）
    - 更新信息面板（阶段、数量）
    - 更新时间线
```

### 阶段 6：后续阶段（循环执行）

后续阶段（查证、灭雷、评估）的执行流程与扫雷阶段相同：

```
21. Controller DAG 执行器检测到下一个阶段节点就绪
    ↓
22. 重复步骤 8-17：
    - 获取阶段输入
    - 通知阶段开始
    - 创建任务并推送给 Worker
    - Worker 执行并发送进度更新
    - Worker 返回结果
    - Controller 通知阶段完成
    ↓
23. 直到所有阶段完成，任务状态变为 "completed"
```

## 关键 API 端点

### MainControl 后端 API

| 端点 | 方法 | 调用者 | 用途 |
|------|------|--------|------|
| `/api/task/start` | POST | 前端 | 创建任务，调用 FSL_Plan |
| `/api/task/{task_id}/stage/{stage}/input` | GET | Controller | 获取阶段输入 payload |
| `/api/task/{task_id}/stage/{stage}/begin` | POST | Controller | 通知阶段开始 |
| `/api/task/{task_id}/stage/{stage}/progress` | POST | Worker | 更新阶段进度 |
| `/api/task/{task_id}/stage/{stage}/result` | POST | Controller | 通知阶段完成 |
| `/api/status` | GET | 前端 | 获取任务状态（轮询） |
| `/api/workflows` | GET | 前端 | 获取工作流列表 |
| `/api/workflows/{workflowId}/run` | POST | 前端 | 触发工作流运行 |
| `/api/workflows/{workflowId}/runs` | GET | 前端 | 获取工作流运行列表 |
| `/api/workflows/{workflowId}/runs/{runId}/status` | GET | 前端 | 获取工作流运行状态 |

### Controller API

| 端点 | 方法 | 调用者 | 用途 |
|------|------|--------|------|
| `/v1/dag/workflows/{workflowId}/run` | POST | MainControl 前端 | 触发工作流运行 |
| `/v1/dag/runs/{runId}/status` | GET | MainControl 前端 | 获取工作流运行状态 |
| gRPC `TaskStream` | 双向流 | Worker | Worker 注册、任务推送、结果返回 |

## 数据流图

```
┌─────────────┐
│   前端 UI   │
└──────┬──────┘
       │ 1. POST /api/task/start
       │ 2. POST /api/workflows/{id}/run
       │ 3. GET /api/status (轮询)
       ↓
┌──────────────────┐
│ MainControl 后端 │
└──────┬───────────┘
       │
       ├─→ 4. POST {plan_service}/planArea (FSL_Plan)
       │
       │ 5. GET /api/task/{id}/stage/{stage}/input ←──┐
       │ 6. POST /api/task/{id}/stage/{stage}/begin ←─┤
       │ 7. POST /api/task/{id}/stage/{stage}/result ←┤
       │                                              │
       │ 8. POST /api/task/{id}/stage/{stage}/progress ←──┐
       │                                                  │
       ↓                                                  │
┌──────────────────┐                                     │
│ Plum Controller  │                                     │
│  (DAG 执行器)    │                                     │
└──────┬───────────┘                                     │
       │                                                 │
       ├─→ gRPC TaskStream (推送任务) ──────────────────┘
       │
       ↓
┌──────────────────┐
│  Plum Worker     │
│ (FSL_Sweep 等)   │
└──────────────────┘
```

## 关键环境变量

### Worker 环境变量（由 Agent 注入）

- `MAIN_CONTROL_BASE`：MainControl 服务地址（用于发送进度更新）
- `CONTROLLER_GRPC_ADDR`：Controller gRPC 地址（用于连接 Controller）
- `WORKER_NODE_ID`：Worker 所在节点 ID
- `PLUM_INSTANCE_ID`：实例 ID

### MainControl 环境变量

- `CONTROLLER_BASE`：Controller HTTP API 地址（用于调用工作流 API）
- `MAINCONTROL_HOST` / `MAINCONTROL_PORT`：MainControl 服务监听地址

## 状态转换

```
任务状态（MainControl）：
created → sweep_pending → sweep_running → investigate_pending → 
investigate_running → destroy_pending → destroy_running → 
evaluate_pending → evaluate_running → completed

节点状态（Controller DAG）：
Pending → Ready → Running → Succeeded/Failed

任务状态（Controller Task）：
Pending → Running → Succeeded/Failed
```

## 注意事项

1. **进度更新的实时性**：
   - Worker 在执行过程中通过 HTTP POST 发送进度更新
   - 前端通过轮询（每秒）获取最新状态
   - 如果 Worker 无法连接到 MainControl（网络问题、环境变量未设置），进度更新会失败，但任务会继续执行

2. **阶段完成通知**：
   - 阶段完成时，Controller 会调用 MainControl 的 `/result` 接口
   - 即使进度更新失败，阶段完成通知也会正常执行（因为 Controller 和 MainControl 之间的连接是可靠的）

3. **服务发现**：
   - MainControl 通过 Controller 的服务发现 API 获取 FSL_Plan 地址
   - Worker 通过 Agent 注入的 `MAIN_CONTROL_BASE` 环境变量获取 MainControl 地址
   - Agent 通过 Controller 的服务发现 API 获取 MainControl 地址并注入到 Worker 容器

4. **网络模式**：
   - 在 bridge 网络模式下，Agent 会根据网络模式调整 MainControl 地址（类似 CONTROLLER_BASE 的处理）
   - 确保 Worker 容器可以访问 MainControl 服务


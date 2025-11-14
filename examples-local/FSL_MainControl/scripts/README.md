# FSL MainControl 联调脚本

本目录包含一个轻量的 Python 脚本 (`orchestrate_task.py`)，用于在没有正式编排 YAML 的情况下，快速串联 FSL 系列服务，验证端到端流程。

## 准备

确保以下服务已启动：

- `FSL_Plan`（默认 `http://127.0.0.1:4100`）
- `FSL_Sweep`（可以通过 `FSL_SWEEP_ENDPOINT` 指定，默认 `http://127.0.0.1:7001/task`）
- `FSL_Investigate`（`FSL_INVESTIGATE_ENDPOINT`，默认 `http://127.0.0.1:7002/task`）
- `FSL_Destroy`（`FSL_DESTROY_ENDPOINT`，默认 `http://127.0.0.1:7003/task`）
- `FSL_MainControl`（默认 `http://127.0.0.1:4000`）

> Worker 端点使用 `task_service.proto`，示例脚本通过 HTTP POST 直接发送 `taskName` 与 `payload` 字段。

## 运行

```bash
cd examples-local/FSL_MainControl/scripts
python3 orchestrate_task.py
```

脚本流程：

1. 调用 `POST /api/task/start` 启动任务；
2. 依次请求三个 Worker 的任务接口；
3. 将阶段结果通过 `POST /api/task/{id}/stage/{stage}/result` 回传；
4. 最后输出任务完整状态。

## 环境变量

- `MAINCONTROL_BASE`：MainControl 服务地址，默认为 `http://127.0.0.1:4000`
- `FSL_SWEEP_ENDPOINT`
- `FSL_INVESTIGATE_ENDPOINT`
- `FSL_DESTROY_ENDPOINT`

## 离线地图瓦片准备

同目录下提供了 `download_tiles.py` 用于在联网环境下载指定区域的 OSM 瓦片（默认围绕 30.664554, 122.510268），执行后会写入 `../static/tiles/`。拷贝该目录到离线环境即可让 Web UI 在无网络时使用本地地图。

## 调试建议

- 可以在 Worker 日志中观察任务输入、输出；
- MainControl UI 会每秒刷新状态，可对照脚本输出验证流程；
- 若某阶段没有待处理的目标，脚本会自动跳过或提前结束。


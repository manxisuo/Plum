# FSL 系列任务编排原型设计

> 目标：快速实现一个可运行的“扫雷-查证-灭雷”三阶段编排演示，突出 Plum 编排能力和地图态势可视化。

## 1. 总体架构

- `FSL_MainControl`（Python + FastAPI + 前端页面）
  - 收集参数、发起编排、聚合阶段结果、提供态势数据接口。
  - 使用 WebSocket 或轮询（本方案采用 1s 轮询 REST）向前端推送态势数据。
  - 通过 Plum gRPC Worker 机制依次调用 `FSL_Sweep` → `FSL_Investigate` → `FSL_Destroy`。
- `FSL_Plan`（HTTP 服务，端口 4100）
  - 接口：`POST /planArea`。
  - 输入：任务矩形 + Ting 数量。
  - 输出：每个 Ting 的作业矩形（按经度方向平均分割，超过 4 条 Ting 时循环分配）。
- `FSL_Sweep` / `FSL_Investigate` / `FSL_Destroy`
  - 均实现为 gRPC Worker，接口兼容 `examples/worker-demo`。
  - 负责模拟各阶段行为，输出阶段结果给下一阶段。
- 数据流
  1. MainControl 收集参数，调用 `FSL_Plan` 获取作业区。
  2. gRPC 链路依次执行阶段，阶段输出 JSON 直接进入下一阶段。
  3. MainControl 将阶段结果写入内存状态（必要时写入 Plum KV 供其他组件读取）。
  4. Web UI 每秒从 MainControl 拉取 `/api/status` 获取态势信息。

## 2. 共用数据模型

所有经纬度均使用 WGS84、小数表示，距离单位为米，速度单位为米/秒，时间戳使用 ISO8601 字符串。

```json
{
  "Ting": {
    "id": "string",
    "name": "string",
    "position": {"lat": 0.0, "lon": 0.0},
    "speed_mps": 8.0,
    "sonar_range_m": 80.0,
    "recognition_probabilities": {
      "suspect": 0.4,
      "confirmed": 0.6
    }
  },
  "Rectangle": {
    "top_left": {"lat": 0.0, "lon": 0.0},
    "bottom_right": {"lat": 0.0, "lon": 0.0}
  },
  "MineTarget": {
    "id": "string",
    "position": {"lat": 0.0, "lon": 0.0},
    "status": "suspect|confirmed|destroyed",
    "source_ting": "string",
    "detected_at": "2025-11-11T00:00:00Z"
  },
  "TrackPoint": {
    "timestamp": "2025-11-11T00:00:00Z",
    "lat": 0.0,
    "lon": 0.0,
    "phase": "sweep|investigate|destroy"
  }
}
```

## 3. 阶段接口定义

### 3.1 gRPC Worker proto

`proto/fsl_task.proto`（后续实现时生成 Python/C++ 代码）：

```proto
syntax = "proto3";

package fsl;

message LatLon {
  double lat = 1;
  double lon = 2;
}

message Rectangle {
  LatLon top_left = 1;
  LatLon bottom_right = 2;
}

message Ting {
  string id = 1;
  string name = 2;
  LatLon position = 3;
  double speed_mps = 4;
  double sonar_range_m = 5;
  double suspect_prob = 6;   // 识别为疑似的概率
  double confirm_prob = 7;   // 识别为确认的概率
}

message Mine {
  string id = 1;
  LatLon position = 2;
  string status = 3; // suspect|confirmed|destroyed|cleared
  string assigned_ting = 4;
}

message TrackPoint {
  string ting_id = 1;
  string phase = 2;
  string timestamp = 3;
  LatLon position = 4;
}

message SweepRequest {
  repeated Ting tings = 1;
  repeated Rectangle work_zones = 2; // 顺序与 tings 对应
  int32 random_seed = 3;
}

message SweepResponse {
  repeated Mine mines = 1;
  repeated TrackPoint tracks = 2;
}

message InvestigateRequest {
  repeated Ting tings = 1;
  repeated Mine suspect_mines = 2;
}

message InvestigateResponse {
  repeated Mine confirmed_mines = 1;
  repeated TrackPoint tracks = 2;
}

message DestroyRequest {
  repeated Ting tings = 1;
  repeated Mine confirmed_mines = 2;
}

message DestroyResponse {
  repeated Mine destroyed_mines = 1;
  repeated TrackPoint tracks = 2;
}
```

> 简化处理：概率控制在 Worker 内部使用，接口只传最终结果。

### 3.2 阶段执行流程

1. **Sweep**：每个作业区提前随机生成 2-3 个雷。艇以垂直航线扫描（北→南），当距离 < sonar_range 时，根据概率判断疑似/确认，保证至少一个疑似。
2. **Investigate**：对疑似目标按最近艇分配任务；艇移动至目标点后重新判定（若仍疑似则保持原状态）。
3. **Destroy**：对确认目标按最近艇分配；艇到达后即标记为 destroyed。

轨迹生成：所有阶段使用统一时间步长（例如 0.5s）线性插值。

## 4. MainControl 接口

- `POST /api/task/start`
  - 请求：任务参数 + Ting 列表。
  - 响应：任务 ID。
- `GET /api/status`
  - 查询参数：`task_id`。
  - 响应：阶段状态、所有艇位置/航迹、目标状态。
- `GET /api/config/defaults`
  - 返回默认参数用于前端表单。

Web 前端基于 Vite + React，地图采用 MapLibre GL。前端每 1s 请求 `/api/status`。

## 5. 开发步骤与调试建议

1. 建立 `examples-local/FSL_*` 工程目录，配置 `meta.ini`。
2. 实现 `FSL_Plan` HTTP 服务（Qt+C++，默认端口 4100）。
3. 实现 `FSL_Sweep` / `FSL_Investigate` / `FSL_Destroy` Worker，并提供 CLI 或单元测试验证。
4. 实现 `FSL_MainControl`（FastAPI + Leaflet 页面）。
5. **编排联调**：依次调用各阶段 Worker，将 JSON 结果回填到 MainControl。
   - 手动测试：通过 REST API 获取阶段输入、POST 阶段结果；
   - 自动编排：使用 Plum Workflow（YAML）串联任务。
6. UI 细化：在任务流程稳定后，完善地图图层、阶段提示与日志展示。

> 快速联调脚本：`examples-local/FSL_MainControl/scripts/orchestrate_task.py`。启动所有服务后运行，可自动调用三个阶段并回传结果。

### 5.1 构建说明（qmake）

```bash
# 第一次需要先构建 C++ SDK
make sdk_cpp          # 或传入 QMAKE=qt6-qmake

# 分别构建 FSL 组件（生成 exe 放在各自 bin/ 目录）
make examples_FSL_Plan
make examples_FSL_Sweep
make examples_FSL_Investigate
make examples_FSL_Destroy
# 或一次构建全部
make examples_FSL_All
```

默认使用 `qmake`，如环境安装的是 Qt6 可执行 `QMAKE=qt6-qmake make examples_FSL_All`。

### 5.2 离线地图瓦片准备

FSL_MainControl 默认尝试从 `static/tiles/{z}/{x}/{y}.png` 读取离线瓦片，若不存在则回退到在线 OSM。

1. 在有网络的环境中执行：

    ```bash
    cd examples-local/FSL_MainControl/scripts
    python3 download_tiles.py --min-zoom 11 --max-zoom 15
    ```

    可通过 `--lat --lon --radius` 调整范围（默认围绕 30.673916,122.973926）。

2. 将生成的 `static/tiles/` 目录整体拷贝到离线部署环境，保持路径结构不变。

3. 离线运行时无需额外配置；地图会优先加载本地瓦片，缺失时呈现透明占位图并自动回退到在线图层（若可联网）。

## 6. MainControl UI 提示

- 左侧面板输入任务参数（可加载默认配置）；
- 顶部按钮可启动任务，启动后界面每秒从 `/api/status` 获取态势；
- 地图图例区分任务区域、作业区、艇、不同状态的水雷以及航迹；
- 右侧面板实时显示当前阶段、疑似/确认/已销毁水雷数量，以及阶段时间线与日志。

## 6. 约束说明

- 忽略异常处理、设备故障、中断等复杂场景。
- 随机数可使用固定种子保证可重复性。
- 初始版本可将状态存于 MainControl 内存，后续视需要接入 KV Store。



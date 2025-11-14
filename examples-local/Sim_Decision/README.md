# SimDecision - 决策系统

SimDecision 是一个决策软件，通过 Web UI 界面调用 SimRoutePlan、SimNaviControl 和 SimSonar 三个服务，完成完整的决策流程。

## 功能特性

- 🎯 **完整决策流程**：依次执行航路规划 → 航控启动 → 声纳探测
- 🌐 **Web UI 界面**：直观展示服务调用过程和结果
- 📊 **实时状态监控**：监控三个服务的在线状态
- 🔄 **自动流程控制**：自动处理服务间的依赖关系
- 📈 **结果可视化**：清晰展示每个步骤的执行结果

## 技术栈

- **后端**: Python 3 + Flask
- **前端**: HTML + CSS + JavaScript
- **HTTP 客户端**: requests

## 安装依赖

```bash
cd examples-local/SimDecision
pip3 install -r requirements.txt
```

## 运行方法

### 方法1：直接运行

```bash
cd examples-local/SimDecision
python3 app.py
```

### 方法2：使用启动脚本

```bash
cd examples-local/SimDecision/bin
./start.sh
```

### 方法3：使用 Flask 命令

```bash
cd examples-local/SimDecision
export FLASK_APP=app.py
flask run --host=0.0.0.0 --port=3000
```

## 配置

可以通过环境变量配置服务地址：

```bash
export SIM_ROUTE_PLAN_URL=http://localhost:3100
export SIM_NAVI_CONTROL_URL=http://localhost:3200
export SIM_SONAR_URL=http://localhost:3300
export PORT=3000
export DEBUG=False
```

## 访问地址

启动后，在浏览器中访问：

```
http://localhost:3000
```

## 使用流程

1. **启动三个服务**：
   - SimRoutePlan (端口 3100)
   - SimNaviControl (端口 3200)
   - SimSonar (端口 3300)

2. **启动 SimDecision**：
   ```bash
   python3 app.py
   ```

3. **打开浏览器**访问 `http://localhost:3000`

4. **输入参数**：
   - 起点经纬度
   - 终点经纬度

5. **点击"执行决策流程"**，系统将：
   - 步骤1：调用 SimRoutePlan 进行航路规划
   - 步骤2：调用 SimNaviControl 启动航控
   - 步骤3：调用 SimSonar 进行声纳探测

6. **查看结果**：界面会实时显示每个步骤的执行状态和结果

## 界面说明

### 服务状态
- 显示三个服务的在线状态（绿色=在线，红色=离线）
- 每5秒自动刷新

### 输入参数
- 起点和终点的经纬度输入
- 支持小数输入

### 执行流程
- 实时显示三个步骤的执行状态
- 每个步骤显示：等待中 → 执行中 → 成功/失败
- 显示详细的执行结果（JSON 格式）

### 结果摘要
- 航路规划结果（航点数量）
- 航控启动结果（状态信息）
- 声纳探测结果（目标列表，包含类型、位置、置信度、距离等）
- 总耗时

## API 接口

### POST /api/execute

执行决策流程

**请求体**:
```json
{
  "point1": {
    "longitude": 116.0,
    "latitude": 39.0
  },
  "point2": {
    "longitude": 116.1,
    "latitude": 39.1
  },
  "obstacle": {
    "polygon": []
  }
}
```

**响应**:
```json
{
  "success": true,
  "workflow": {
    "step": 3,
    "total_steps": 3,
    "status": "success",
    "steps": [...],
    "total_time": 8.5
  },
  "results": {
    "route_plan": {...},
    "navi_control": {...},
    "sonar": {...}
  }
}
```

### GET /api/status

检查服务状态

**响应**:
```json
{
  "route_plan": true,
  "navi_control": true,
  "sonar": true
}
```

## 注意事项

1. **服务依赖**：确保三个服务（SimRoutePlan、SimNaviControl、SimSonar）都已启动
2. **端口占用**：默认端口 3000，可通过 `PORT` 环境变量修改
3. **超时设置**：由于 SimSonar 需要 3-5 秒，请求超时设置为 60 秒
4. **错误处理**：如果某个步骤失败，后续步骤不会执行
5. **浏览器兼容**：建议使用现代浏览器（Chrome、Firefox、Edge 等）

## 故障排查

### 服务无法连接

- 检查三个服务是否已启动
- 检查服务地址和端口是否正确
- 查看浏览器控制台的错误信息

### 执行失败

- 检查输入参数格式是否正确
- 查看服务日志了解详细错误信息
- 确认网络连接正常


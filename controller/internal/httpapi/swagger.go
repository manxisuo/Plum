package httpapi

import (
	"encoding/json"
	"net/http"
)

func handleSwaggerUI(w http.ResponseWriter, r *http.Request) {
	const html = `<!DOCTYPE html><html><head><meta charset="utf-8"/><title>Plum API - Swagger UI</title>
    <link rel="stylesheet" href="/swagger/swagger-ui.css" />
    </head><body>
    <div id="swagger"></div>
    <script src="/swagger/swagger-ui-bundle.js"></script>
    <script>window.ui = SwaggerUIBundle({ url: '/swagger/openapi.json', dom_id: '#swagger' });</script>
    </body></html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

func handleSwaggerCSS(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "controller/static/swagger-ui.css")
}

func handleSwaggerJS(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "controller/static/swagger-ui-bundle.js")
}

func handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	doc := generateOpenAPIDoc()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

func generateOpenAPIDoc() map[string]any {
	type OA = map[string]any

	doc := OA{
		"openapi": "3.0.0",
		"info": OA{
			"title":       "Plum Controller API",
			"version":     "0.1.0",
			"description": "Plum分布式任务调度系统API文档",
		},
		"servers": []OA{
			{"url": "http://localhost:8080", "description": "本地开发服务器"},
		},
		"paths": OA{
			"/healthz": OA{
				"get": OA{
					"summary":   "健康检查",
					"responses": OA{"200": OA{"description": "服务正常"}},
				},
			},
			"/v1/stream": OA{
				"get": OA{
					"summary":   "SSE事件流",
					"responses": OA{"200": OA{"description": "事件流"}},
				},
			},
			"/v1/nodes": OA{
				"get": OA{
					"summary":   "获取所有节点",
					"responses": OA{"200": OA{"description": "节点列表"}},
				},
			},
			"/v1/nodes/heartbeat": OA{
				"post": OA{
					"summary":   "节点心跳",
					"responses": OA{"200": OA{"description": "心跳确认"}},
				},
			},
			"/v1/nodes/{id}": OA{
				"get": OA{
					"summary":   "获取指定节点",
					"responses": OA{"200": OA{"description": "节点信息"}},
				},
				"delete": OA{
					"summary":   "删除节点",
					"responses": OA{"204": OA{"description": "删除成功"}},
				},
			},
			"/v1/apps": OA{
				"get": OA{
					"summary":   "获取所有应用",
					"responses": OA{"200": OA{"description": "应用列表"}},
				},
			},
			"/v1/apps/upload": OA{
				"post": OA{
					"summary":   "上传应用包",
					"responses": OA{"200": OA{"description": "上传成功"}},
				},
			},
			"/v1/apps/{id}": OA{
				"delete": OA{
					"summary":   "删除应用",
					"responses": OA{"204": OA{"description": "删除成功"}},
				},
			},
			"/v1/assignments": OA{
				"get": OA{
					"summary": "获取节点分配",
					"parameters": []OA{
						{"name": "nodeId", "in": "query", "required": true, "schema": OA{"type": "string"}, "description": "节点ID"},
						{"name": "limit", "in": "query", "required": false, "schema": OA{"type": "integer"}, "description": "限制数量"},
					},
					"responses": OA{"200": OA{"description": "分配列表"}},
				},
			},
			"/v1/assignments/{id}": OA{
				"patch": OA{
					"summary":   "更新分配状态",
					"responses": OA{"204": OA{"description": "更新成功"}},
				},
				"delete": OA{
					"summary":   "删除分配",
					"responses": OA{"204": OA{"description": "删除成功"}},
				},
			},
			"/v1/instances/status": OA{
				"post": OA{
					"summary":   "更新实例状态",
					"responses": OA{"204": OA{"description": "更新成功"}},
				},
			},
			"/v1/services/register": OA{
				"post": OA{
					"summary":   "注册服务端点",
					"responses": OA{"204": OA{"description": "注册成功"}},
				},
			},
			"/v1/services/heartbeat": OA{
				"post": OA{
					"summary":   "服务心跳",
					"responses": OA{"204": OA{"description": "心跳成功"}},
				},
			},
			"/v1/services": OA{
				"delete": OA{
					"summary": "删除服务端点",
					"parameters": []OA{
						{"name": "instanceId", "in": "query", "required": true, "schema": OA{"type": "string"}, "description": "实例ID"},
					},
					"responses": OA{"204": OA{"description": "删除成功"}},
				},
			},
			"/v1/services/list": OA{
				"get": OA{
					"summary":   "获取服务列表",
					"responses": OA{"200": OA{"description": "服务列表"}},
				},
			},
			"/v1/discovery": OA{
				"get": OA{
					"summary": "服务发现",
					"parameters": []OA{
						{"name": "service", "in": "query", "required": true, "schema": OA{"type": "string"}, "description": "服务名"},
						{"name": "version", "in": "query", "required": false, "schema": OA{"type": "string"}, "description": "版本"},
						{"name": "protocol", "in": "query", "required": false, "schema": OA{"type": "string"}, "description": "协议"},
					},
					"responses": OA{"200": OA{"description": "服务端点列表"}},
				},
			},
			"/v1/discovery/random": OA{
				"get": OA{
					"summary": "随机服务发现",
					"parameters": []OA{
						{"name": "service", "in": "query", "required": true, "schema": OA{"type": "string"}, "description": "服务名"},
						{"name": "version", "in": "query", "required": false, "schema": OA{"type": "string"}, "description": "版本"},
						{"name": "protocol", "in": "query", "required": false, "schema": OA{"type": "string"}, "description": "协议"},
					},
					"responses": OA{
						"200": OA{"description": "随机选择的端点"},
						"404": OA{"description": "未找到端点"},
					},
				},
			},
			"/v1/workers/register": OA{
				"post": OA{
					"summary":   "注册嵌入式工作器",
					"responses": OA{"204": OA{"description": "注册成功"}},
				},
			},
			"/v1/workers/heartbeat": OA{
				"post": OA{
					"summary":   "工作器心跳",
					"responses": OA{"204": OA{"description": "心跳成功"}},
				},
			},
			"/v1/workers": OA{
				"get": OA{
					"summary":   "获取工作器列表",
					"responses": OA{"200": OA{"description": "工作器列表"}},
				},
			},
			"/v1/resources/register": OA{
				"post": OA{
					"summary":   "注册资源",
					"responses": OA{"204": OA{"description": "注册成功"}},
				},
			},
			"/v1/resources/heartbeat": OA{
				"post": OA{
					"summary":   "资源心跳",
					"responses": OA{"204": OA{"description": "心跳成功"}},
				},
			},
			"/v1/resources": OA{
				"get": OA{
					"summary":   "获取资源列表",
					"responses": OA{"200": OA{"description": "资源列表"}},
				},
			},
			"/v1/resources/{id}": OA{
				"get": OA{
					"summary":   "获取指定资源",
					"responses": OA{"200": OA{"description": "资源信息"}},
				},
			},
			"/v1/resources/state": OA{
				"post": OA{
					"summary":   "提交资源状态",
					"responses": OA{"204": OA{"description": "提交成功"}},
				},
			},
			"/v1/resources/states": OA{
				"get": OA{
					"summary":   "获取资源状态列表",
					"responses": OA{"200": OA{"description": "状态列表"}},
				},
			},
			"/v1/resources/operation": OA{
				"post": OA{
					"summary":   "资源操作",
					"responses": OA{"200": OA{"description": "操作成功"}},
				},
			},
			"/v1/embedded-workers/register": OA{
				"post": OA{
					"summary":   "注册嵌入式工作器",
					"responses": OA{"204": OA{"description": "注册成功"}},
				},
			},
			"/v1/embedded-workers/heartbeat": OA{
				"post": OA{
					"summary":   "嵌入式工作器心跳",
					"responses": OA{"204": OA{"description": "心跳成功"}},
				},
			},
			"/v1/embedded-workers": OA{
				"get": OA{
					"summary":   "获取嵌入式工作器列表",
					"responses": OA{"200": OA{"description": "工作器列表"}},
				},
			},
			"/v1/embedded-workers/{id}": OA{
				"get": OA{
					"summary":   "获取指定嵌入式工作器",
					"responses": OA{"200": OA{"description": "工作器信息"}},
				},
				"delete": OA{
					"summary":   "删除嵌入式工作器",
					"responses": OA{"204": OA{"description": "删除成功"}},
				},
			},
			"/v1/deployments": OA{
				"get": OA{
					"summary":   "获取部署列表",
					"responses": OA{"200": OA{"description": "部署列表"}},
				},
				"post": OA{
					"summary":   "创建部署",
					"responses": OA{"200": OA{"description": "创建成功"}},
				},
			},
			"/v1/deployments/{id}": OA{
				"get": OA{
					"summary":   "获取指定部署",
					"responses": OA{"200": OA{"description": "部署信息"}},
				},
				"patch": OA{
					"summary":   "更新部署标签",
					"responses": OA{"200": OA{"description": "更新成功"}},
				},
				"delete": OA{
					"summary":   "删除部署",
					"responses": OA{"204": OA{"description": "删除成功"}},
				},
			},
			"/v1/tasks": OA{
				"get": OA{
					"summary":   "获取任务列表",
					"responses": OA{"200": OA{"description": "任务列表"}},
				},
			},
			"/v1/tasks/{id}": OA{
				"get": OA{
					"summary":   "获取指定任务",
					"responses": OA{"200": OA{"description": "任务信息"}},
				},
			},
			"/v1/tasks/stream": OA{
				"get": OA{
					"summary":   "任务事件流",
					"responses": OA{"200": OA{"description": "事件流"}},
				},
			},
			"/v1/tasks/start/{id}": OA{
				"post": OA{
					"summary":   "启动任务",
					"responses": OA{"200": OA{"description": "启动成功"}},
				},
			},
			"/v1/tasks/rerun/{id}": OA{
				"post": OA{
					"summary":   "重新运行任务",
					"responses": OA{"200": OA{"description": "重新运行成功"}},
				},
			},
			"/v1/tasks/cancel/{id}": OA{
				"post": OA{
					"summary":   "取消任务",
					"responses": OA{"200": OA{"description": "取消成功"}},
				},
			},
			"/v1/workflows": OA{
				"get": OA{
					"summary":   "获取工作流列表",
					"responses": OA{"200": OA{"description": "工作流列表"}},
				},
			},
			"/v1/workflows/{id}": OA{
				"get": OA{
					"summary":   "获取指定工作流",
					"responses": OA{"200": OA{"description": "工作流信息"}},
				},
			},
			"/v1/workflow-runs": OA{
				"get": OA{
					"summary":   "获取工作流运行列表",
					"responses": OA{"200": OA{"description": "运行列表"}},
				},
			},
			"/v1/workflow-runs/{id}": OA{
				"get": OA{
					"summary":   "获取指定工作流运行",
					"responses": OA{"200": OA{"description": "运行信息"}},
				},
			},
			"/v1/dag/workflows": OA{
				"get": OA{
					"summary":   "获取DAG工作流列表",
					"responses": OA{"200": OA{"description": "DAG工作流列表"}},
				},
			},
			"/v1/dag/workflows/{id}": OA{
				"get": OA{
					"summary":   "获取指定DAG工作流",
					"responses": OA{"200": OA{"description": "DAG工作流信息"}},
				},
			},
			"/v1/dag/runs/{id}": OA{
				"get": OA{
					"summary":   "获取DAG运行状态",
					"responses": OA{"200": OA{"description": "运行状态"}},
				},
			},
			"/v1/task-defs": OA{
				"get": OA{
					"summary":   "获取任务定义列表",
					"responses": OA{"200": OA{"description": "任务定义列表"}},
				},
			},
			"/v1/task-defs/{id}": OA{
				"get": OA{
					"summary":   "获取指定任务定义",
					"responses": OA{"200": OA{"description": "任务定义信息"}},
				},
			},
			"/v1/kv/{namespace}/{key}": OA{
				"get": OA{
					"summary": "获取键值对",
					"parameters": []OA{
						{"name": "namespace", "in": "path", "required": true, "schema": OA{"type": "string"}, "description": "命名空间"},
						{"name": "key", "in": "path", "required": true, "schema": OA{"type": "string"}, "description": "键名"},
					},
					"responses": OA{"200": OA{"description": "键值对"}},
				},
				"put": OA{
					"summary": "设置键值对",
					"parameters": []OA{
						{"name": "namespace", "in": "path", "required": true, "schema": OA{"type": "string"}, "description": "命名空间"},
						{"name": "key", "in": "path", "required": true, "schema": OA{"type": "string"}, "description": "键名"},
					},
					"requestBody": OA{"required": true, "content": OA{"application/json": OA{"schema": OA{"type": "object"}}}},
					"responses":   OA{"200": OA{"description": "设置成功"}},
				},
				"delete": OA{
					"summary": "删除键值对",
					"parameters": []OA{
						{"name": "namespace", "in": "path", "required": true, "schema": OA{"type": "string"}, "description": "命名空间"},
						{"name": "key", "in": "path", "required": true, "schema": OA{"type": "string"}, "description": "键名"},
					},
					"responses": OA{"204": OA{"description": "删除成功"}},
				},
			},
		},
		"components": OA{
			"schemas": OA{},
		},
	}

	return doc
}

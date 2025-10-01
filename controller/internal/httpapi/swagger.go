package httpapi

import (
	"encoding/json"
	"net/http"
)

// serve a very small Swagger UI wrapper and a minimal OpenAPI doc built from current routes

func handleSwaggerUI(w http.ResponseWriter, r *http.Request) {
	// simple embedded HTML referencing the same host for /swagger/openapi.json
	const html = `<!DOCTYPE html><html><head><meta charset="utf-8"/><title>Plum API - Swagger UI</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui.css" />
    </head><body>
    <div id="swagger"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui-bundle.js"></script>
    <script>window.ui = SwaggerUIBundle({ url: '/swagger/openapi.json', dom_id: '#swagger' });</script>
    </body></html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

func handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	// Expanded OpenAPI 3.0 with params/requestBody/response schemas so Swagger UI shows details
	type OA = map[string]any
	doc := OA{
		"openapi": "3.0.0",
		"info":    OA{"title": "Plum Controller API", "version": "0.1.0"},
		"paths": OA{
			"/healthz": OA{"get": OA{
				"summary":   "Health Check",
				"responses": OA{"200": OA{"description": "ok", "content": OA{"text/plain": OA{"schema": OA{"type": "string"}}}}},
			}},
			"/v1/nodes": OA{"get": OA{
				"summary":   "List nodes",
				"responses": OA{"200": OA{"description": "OK", "content": OA{"application/json": OA{"schema": OA{"type": "array", "items": OA{"$ref": "#/components/schemas/NodeDTO"}}}}}},
			}},
			"/v1/nodes/heartbeat": OA{"post": OA{
				"summary":     "Node heartbeat",
				"requestBody": OA{"required": true, "content": OA{"application/json": OA{"schema": OA{"$ref": "#/components/schemas/NodeHello"}}}},
				"responses":   OA{"200": OA{"description": "OK", "content": OA{"application/json": OA{"schema": OA{"$ref": "#/components/schemas/LeaseAck"}}}}},
			}},
			"/v1/nodes/{id}": OA{
				"get":    OA{"summary": "Get node", "parameters": []any{OA{"name": "id", "in": "path", "required": true, "schema": OA{"type": "string"}}}, "responses": OA{"200": OA{"description": "OK", "content": OA{"application/json": OA{"schema": OA{"$ref": "#/components/schemas/NodeDTO"}}}}, "404": OA{"description": "Not Found"}}},
				"delete": OA{"summary": "Delete node", "parameters": []any{OA{"name": "id", "in": "path", "required": true, "schema": OA{"type": "string"}}}, "responses": OA{"204": OA{"description": "No Content"}, "409": OA{"description": "node in use"}}},
			},
			"/v1/apps":        OA{"get": OA{"summary": "List apps", "responses": OA{"200": OA{"description": "OK", "content": OA{"application/json": OA{"schema": OA{"type": "array", "items": OA{"$ref": "#/components/schemas/AppInfo"}}}}}}}},
			"/v1/apps/upload": OA{"post": OA{"summary": "Upload app (zip)", "requestBody": OA{"required": true, "content": OA{"multipart/form-data": OA{"schema": OA{"type": "object", "properties": OA{"file": OA{"type": "string", "format": "binary"}}, "required": []any{"file"}}}}}, "responses": OA{"200": OA{"description": "OK", "content": OA{"application/json": OA{"schema": OA{"$ref": "#/components/schemas/AppInfo"}}}}}}},
			"/v1/apps/{id}":   OA{"delete": OA{"summary": "Delete app", "parameters": []any{OA{"name": "id", "in": "path", "required": true, "schema": OA{"type": "string"}}}, "responses": OA{"204": OA{"description": "No Content"}, "409": OA{"description": "artifact in use"}}}},
			"/v1/deployments": OA{
				"get":  OA{"summary": "List deployments", "responses": OA{"200": OA{"description": "OK"}}},
				"post": OA{"summary": "Create deployment", "responses": OA{"200": OA{"description": "OK"}}},
			},
			"/v1/deployments/{id}": OA{
				"get":    OA{"summary": "Get deployment (alias)", "responses": OA{"200": OA{"description": "OK"}}},
				"patch":  OA{"summary": "Patch deployment labels (alias)", "responses": OA{"200": OA{"description": "OK"}}},
				"delete": OA{"summary": "Delete deployment (alias)", "responses": OA{"204": OA{"description": "No Content"}}},
			},
			"/v1/assignments": OA{"get": OA{"summary": "List assignments by node", "parameters": []any{OA{"name": "nodeId", "in": "query", "required": true, "schema": OA{"type": "string"}}, OA{"name": "limit", "in": "query", "required": false, "schema": OA{"type": "integer", "format": "int32"}}}, "responses": OA{"200": OA{"description": "OK", "content": OA{"application/json": OA{"schema": OA{"$ref": "#/components/schemas/Assignments"}}}}}}},
			"/v1/assignments/{id}": OA{
				"patch":  OA{"summary": "Update desired", "parameters": []any{OA{"name": "id", "in": "path", "required": true, "schema": OA{"type": "string"}}}, "requestBody": OA{"required": true, "content": OA{"application/json": OA{"schema": OA{"$ref": "#/components/schemas/AssignmentDesiredPatch"}}}}, "responses": OA{"204": OA{"description": "No Content"}}},
				"delete": OA{"summary": "Delete assignment", "parameters": []any{OA{"name": "id", "in": "path", "required": true, "schema": OA{"type": "string"}}}, "responses": OA{"204": OA{"description": "No Content"}}},
			},
			"/v1/instances/status":   OA{"post": OA{"summary": "Append instance status", "requestBody": OA{"required": true, "content": OA{"application/json": OA{"schema": OA{"$ref": "#/components/schemas/StatusUpdate"}}}}, "responses": OA{"204": OA{"description": "No Content"}}}},
			"/v1/services/register":  OA{"post": OA{"summary": "Register/replace service endpoints for an instance", "requestBody": OA{"required": true, "content": OA{"application/json": OA{"schema": OA{"$ref": "#/components/schemas/RegisterRequest"}}}}, "responses": OA{"204": OA{"description": "No Content"}}}},
			"/v1/services/heartbeat": OA{"post": OA{"summary": "Heartbeat for endpoints with health overrides", "requestBody": OA{"required": true, "content": OA{"application/json": OA{"schema": OA{"$ref": "#/components/schemas/HeartbeatRequest"}}}}, "responses": OA{"204": OA{"description": "No Content"}}}},
			"/v1/services":           OA{"delete": OA{"summary": "Delete all endpoints for an instance", "parameters": []any{OA{"name": "instanceId", "in": "query", "required": true, "schema": OA{"type": "string"}}}, "responses": OA{"204": OA{"description": "No Content"}}}},
			"/v1/discovery":          OA{"get": OA{"summary": "Discover endpoints by service", "parameters": []any{OA{"name": "service", "in": "query", "required": true, "schema": OA{"type": "string"}}, OA{"name": "version", "in": "query", "schema": OA{"type": "string"}}, OA{"name": "protocol", "in": "query", "schema": OA{"type": "string"}}, OA{"name": "limit", "in": "query", "schema": OA{"type": "integer"}}}, "responses": OA{"200": OA{"description": "OK", "content": OA{"application/json": OA{"schema": OA{"type": "array", "items": OA{"$ref": "#/components/schemas/EndpointDTO"}}}}}}}},
		},
		"components": OA{"schemas": OA{
			"NodeHello":               OA{"type": "object", "properties": OA{"nodeId": OA{"type": "string"}, "ip": OA{"type": "string"}, "labels": OA{"type": "object", "additionalProperties": OA{"type": "string"}}}, "required": []any{"nodeId"}},
			"NodeDTO":                 OA{"type": "object", "properties": OA{"nodeId": OA{"type": "string"}, "ip": OA{"type": "string"}, "labels": OA{"type": "object", "additionalProperties": OA{"type": "string"}}, "lastSeen": OA{"type": "integer", "format": "int64"}}},
			"LeaseAck":                OA{"type": "object", "properties": OA{"ttlSec": OA{"type": "integer", "format": "int64"}}},
			"AppInfo":                 OA{"type": "object", "properties": OA{"artifactId": OA{"type": "string"}, "name": OA{"type": "string"}, "version": OA{"type": "string"}, "url": OA{"type": "string"}, "sha256": OA{"type": "string"}, "sizeBytes": OA{"type": "integer", "format": "int64"}, "createdAt": OA{"type": "integer", "format": "int64"}}},
			"DeploymentDTO":           OA{"type": "object", "properties": OA{"deploymentId": OA{"type": "string"}, "name": OA{"type": "string"}, "labels": OA{"type": "object", "additionalProperties": OA{"type": "string"}}, "instances": OA{"type": "integer"}}},
			"Deployment":              OA{"type": "object", "properties": OA{"deploymentId": OA{"type": "string"}, "name": OA{"type": "string"}, "labels": OA{"type": "object", "additionalProperties": OA{"type": "string"}}}},
			"DeploymentPatch":         OA{"type": "object", "properties": OA{"name": OA{"type": "string"}, "labels": OA{"type": "object", "additionalProperties": OA{"type": "string"}}}},
			"CreateDeploymentEntry":   OA{"type": "object", "properties": OA{"artifactUrl": OA{"type": "string"}, "startCmd": OA{"type": "string"}, "replicas": OA{"type": "object", "additionalProperties": OA{"type": "integer"}}}, "required": []any{"artifactUrl", "replicas"}},
			"CreateDeploymentRequest": OA{"type": "object", "properties": OA{"name": OA{"type": "string"}, "labels": OA{"type": "object", "additionalProperties": OA{"type": "string"}}, "entries": OA{"type": "array", "items": OA{"$ref": "#/components/schemas/CreateDeploymentEntry"}}, "artifactUrl": OA{"type": "string"}, "startCmd": OA{"type": "string"}, "replicas": OA{"type": "object", "additionalProperties": OA{"type": "integer"}}}, "required": []any{"name"}},
			"Assignment":              OA{"type": "object", "properties": OA{"instanceId": OA{"type": "string"}, "deploymentId": OA{"type": "string"}, "nodeId": OA{"type": "string"}, "desired": OA{"type": "string"}, "artifactUrl": OA{"type": "string"}, "startCmd": OA{"type": "string"}}},
			"AssignmentItem":          OA{"type": "object", "properties": OA{"instanceId": OA{"type": "string"}, "deploymentId": OA{"type": "string"}, "desired": OA{"type": "string"}, "artifactUrl": OA{"type": "string"}, "startCmd": OA{"type": "string"}, "phase": OA{"type": "string"}, "healthy": OA{"type": "boolean"}, "lastReportAt": OA{"type": "integer", "format": "int64"}}},
			"Assignments":             OA{"type": "object", "properties": OA{"items": OA{"type": "array", "items": OA{"$ref": "#/components/schemas/AssignmentItem"}}}},
			"AssignmentDesiredPatch":  OA{"type": "object", "properties": OA{"desired": OA{"type": "string", "enum": []any{"Running", "Stopped"}}}, "required": []any{"desired"}},
			"StatusUpdate":            OA{"type": "object", "properties": OA{"instanceId": OA{"type": "string"}, "phase": OA{"type": "string"}, "exitCode": OA{"type": "integer", "format": "int32"}, "healthy": OA{"type": "boolean"}, "tsUnix": OA{"type": "integer", "format": "int64"}}, "required": []any{"instanceId", "phase", "tsUnix"}},
			"EndpointDTO":             OA{"type": "object", "properties": OA{"serviceName": OA{"type": "string"}, "instanceId": OA{"type": "string"}, "nodeId": OA{"type": "string"}, "ip": OA{"type": "string"}, "port": OA{"type": "integer"}, "protocol": OA{"type": "string"}, "version": OA{"type": "string"}, "labels": OA{"type": "object", "additionalProperties": OA{"type": "string"}}, "healthy": OA{"type": "boolean"}, "lastSeen": OA{"type": "integer", "format": "int64"}}},
			"RegisterRequest":         OA{"type": "object", "properties": OA{"instanceId": OA{"type": "string"}, "nodeId": OA{"type": "string"}, "ip": OA{"type": "string"}, "endpoints": OA{"type": "array", "items": OA{"$ref": "#/components/schemas/EndpointDTO"}}}, "required": []any{"instanceId", "nodeId", "ip", "endpoints"}},
			"HeartbeatRequest":        OA{"type": "object", "properties": OA{"instanceId": OA{"type": "string"}, "health": OA{"type": "array", "items": OA{"$ref": "#/components/schemas/EndpointDTO"}}}, "required": []any{"instanceId"}},
		}},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(doc)
}

package main

// Assignment 部署分配
type Assignment struct {
	InstanceID      string `json:"instanceId"`
	DeploymentID    string `json:"deploymentId"`
	NodeID          string `json:"nodeId"`
	Desired         string `json:"desired"`
	ArtifactURL     string `json:"artifactUrl"`
	StartCmd        string `json:"startCmd"`
	AppName         string `json:"appName"`
	AppVersion      string `json:"appVersion"`
	ArtifactType    string `json:"artifactType,omitempty"` // "zip" or "image"
	ImageRepository string `json:"imageRepository,omitempty"`
	ImageTag        string `json:"imageTag,omitempty"`
	PortMappings    string `json:"portMappings,omitempty"` // JSON string
}

// InstanceStatus 实例状态
type InstanceStatus struct {
	InstanceID string `json:"instanceId"`
	Phase      string `json:"phase"`
	ExitCode   int    `json:"exitCode"`
	Healthy    bool   `json:"healthy"`
	TsUnix     int64  `json:"tsUnix"`
}

// ServiceEndpoint 服务端点
type ServiceEndpoint struct {
	ServiceName string `json:"serviceName"`
	Protocol    string `json:"protocol"`
	Port        int    `json:"port"`
}

// ServiceRegistration 服务注册
type ServiceRegistration struct {
	InstanceID string            `json:"instanceId"`
	NodeID     string            `json:"nodeId"`
	IP         string            `json:"ip"`
	Endpoints  []ServiceEndpoint `json:"endpoints"`
}

// HeartbeatRequest 心跳请求
type HeartbeatRequest struct {
	InstanceID string `json:"instanceId,omitempty"`
}

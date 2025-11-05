package main

import (
	"fmt"
	"os"
	"strings"
)

// AppManager 应用管理器接口
// 统一管理裸应用和容器应用的生命周期
type AppManager interface {
	// StartApp 启动应用
	// instanceID: 实例ID
	// app: 应用配置
	// appDir: 应用目录（已解压的artifact）
	StartApp(instanceID string, app Assignment, appDir string) error

	// StopApp 停止应用
	// instanceID: 实例ID
	StopApp(instanceID string) error

	// IsRunning 检查应用是否正在运行
	// instanceID: 实例ID
	// 返回：是否运行中
	IsRunning(instanceID string) bool

	// GetStatus 获取应用状态
	// instanceID: 实例ID
	// 返回：应用状态信息
	GetStatus(instanceID string) (AppStatus, error)

	// ListRunning 列出所有运行中的实例ID
	// 返回：运行中的实例ID列表
	ListRunning() []string
}

// AppStatus 应用状态
type AppStatus struct {
	InstanceID  string
	Running     bool
	Pid         int    // 进程模式：进程ID；容器模式：容器PID（通常为0）
	ContainerID string // 容器模式：容器ID；进程模式：空
}

// ManagerConfig 管理器配置
type ManagerConfig struct {
	BaseDir    string
	HTTP       *HTTPClient
	Controller string
}

// NewAppManager 创建应用管理器
// mode: 运行模式 "process"（默认）或 "docker"
// config: 管理器配置
func NewAppManager(mode string, config ManagerConfig) (AppManager, error) {
	if mode == "" {
		mode = "process" // 默认模式
	}

	switch mode {
	case "process":
		return NewProcessManager(config), nil
	case "docker":
		dockerManager, err := NewDockerManager(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create docker manager: %w", err)
		}
		return dockerManager, nil
	default:
		return nil, fmt.Errorf("unknown run mode: %s (supported: process, docker)", mode)
	}
}

// GetRunMode 从环境变量获取运行模式
func GetRunMode() string {
	mode := os.Getenv("AGENT_RUN_MODE")
	if mode == "" {
		mode = "process" // 默认进程模式
	}
	return strings.ToLower(mode)
}

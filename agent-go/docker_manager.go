package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

// DockerManager 容器模式管理器
// 使用Docker SDK管理应用容器
type DockerManager struct {
	config     ManagerConfig
	client     *client.Client
	ctx        context.Context
	containers map[string]string // instanceID -> containerID
}

// NewDockerManager 创建Docker管理器
func NewDockerManager(config ManagerConfig) (*DockerManager, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	// 测试连接
	_, err = cli.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to docker daemon: %w (is docker running?)", err)
	}

	log.Printf("Docker manager initialized successfully")
	return &DockerManager{
		config:     config,
		client:     cli,
		ctx:        ctx,
		containers: make(map[string]string),
	}, nil
}

// StartApp 启动应用容器
func (m *DockerManager) StartApp(instanceID string, app Assignment, appDir string) error {
	// 检查容器是否已存在并运行
	if containerID, exists := m.containers[instanceID]; exists {
		info, err := m.client.ContainerInspect(m.ctx, containerID)
		if err == nil {
			if info.State.Running {
				log.Printf("Container %s for instance %s already running", containerID[:12], instanceID)
				return nil
			}
		}
		// 容器已停止，清理记录
		delete(m.containers, instanceID)
	}

	// 容器命名：plum-app-{instanceID}
	containerName := fmt.Sprintf("plum-app-%s", instanceID)

	// 检查是否已有同名容器（可能是之前的残留）
	existing, err := m.client.ContainerInspect(m.ctx, containerName)
	if err == nil {
		// 如果容器存在但不运行，先删除
		if !existing.State.Running {
			log.Printf("Removing existing stopped container %s", containerName)
			m.client.ContainerRemove(m.ctx, containerName, types.ContainerRemoveOptions{Force: true})
		}
	}

	// 获取基础镜像（从环境变量或使用默认）
	baseImage := os.Getenv("PLUM_BASE_IMAGE")
	if baseImage == "" {
		baseImage = "alpine:latest" // 默认基础镜像
	}
	log.Printf("Using base image: %s", baseImage)

	// 准备启动命令
	cmdline := strings.TrimSpace(app.StartCmd)
	if cmdline == "" {
		cmdline = "./start.sh"
	}
	// 将命令分割为命令和参数
	cmdParts := strings.Fields(cmdline)
	if len(cmdParts) == 0 {
		cmdParts = []string{"./start.sh"}
	}

	// 构建环境变量列表
	envVars := []string{
		fmt.Sprintf("PLUM_INSTANCE_ID=%s", app.InstanceID),
		fmt.Sprintf("PLUM_APP_NAME=%s", app.AppName),
		fmt.Sprintf("PLUM_APP_VERSION=%s", app.AppVersion),
	}

	// 添加自定义环境变量（支持多个，格式：PLUM_CONTAINER_ENV_xxx=value）
	// 或者通过 PLUM_CONTAINER_ENV 设置（格式：KEY1=value1,KEY2=value2）
	customEnvStr := os.Getenv("PLUM_CONTAINER_ENV")
	if customEnvStr != "" {
		// 解析逗号分隔的环境变量
		customEnvs := strings.Split(customEnvStr, ",")
		for _, env := range customEnvs {
			env = strings.TrimSpace(env)
			if env != "" {
				envVars = append(envVars, env)
			}
		}
	}

	// 自动添加 LD_LIBRARY_PATH（如果应用目录有lib子目录）
	// 这对于Qt等需要共享库的应用很有用
	if _, err := os.Stat(filepath.Join(appDir, "lib")); err == nil {
		ldLibraryPath := "/app/lib:/usr/lib:/lib"
		// 如果已有LD_LIBRARY_PATH，追加
		hasLdPath := false
		for i, env := range envVars {
			if strings.HasPrefix(env, "LD_LIBRARY_PATH=") {
				envVars[i] = env + ":/app/lib"
				hasLdPath = true
				break
			}
		}
		if !hasLdPath {
			envVars = append(envVars, fmt.Sprintf("LD_LIBRARY_PATH=%s", ldLibraryPath))
		}
		log.Printf("Added LD_LIBRARY_PATH=/app/lib for instance %s", instanceID)
	}

	// 创建容器配置
	containerConfig := &container.Config{
		Image:      baseImage,
		Cmd:        cmdParts,
		WorkingDir: "/app",
		Env:        envVars,
		// 设置容器自动删除（停止后）
		// 但我们需要手动管理，所以不设置AutoRemove
	}

	// 构建挂载列表
	mounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: appDir,
			Target: "/app",
		},
	}

	// 可选：挂载宿主机的库路径（用于共享系统库）
	// 这样可以避免每个应用都自包含相同的库，减少重复
	hostLibPaths := getHostLibraryPaths()
	for _, libPath := range hostLibPaths {
		// 检查宿主机路径是否存在
		if _, err := os.Stat(libPath); err == nil {
			// 映射到容器的相同路径（只读挂载，防止容器修改宿主机的库）
			mounts = append(mounts, mount.Mount{
				Type:     mount.TypeBind,
				Source:   libPath,
				Target:   libPath,
				ReadOnly: true, // 只读挂载，保护宿主机库
			})
			log.Printf("Mounted host library path %s to container for instance %s", libPath, instanceID)
		} else {
			log.Printf("Warning: host library path %s does not exist, skipping mount", libPath)
		}
	}

	// 创建主机配置（挂载应用目录和库路径）
	hostConfig := &container.HostConfig{
		Mounts: mounts,
		// 资源限制（可选，从环境变量读取）
		Resources: container.Resources{
			Memory:   getMemoryLimit(), // 从环境变量或默认值
			NanoCPUs: getCPULimit(),    // CPU限制
		},
		// 网络模式：使用bridge网络（默认）
		NetworkMode: container.NetworkMode("bridge"),
		// 自动重启策略：不自动重启（由Agent管理）
		RestartPolicy: container.RestartPolicy{Name: "no"},
	}

	// 创建容器
	resp, err := m.client.ContainerCreate(
		m.ctx,
		containerConfig,
		hostConfig,
		nil,
		nil,
		containerName,
	)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	containerID := resp.ID
	log.Printf("Created container %s for instance %s", containerID[:12], instanceID)

	// 启动容器
	if err := m.client.ContainerStart(m.ctx, containerID, types.ContainerStartOptions{}); err != nil {
		// 如果启动失败，清理容器
		m.client.ContainerRemove(m.ctx, containerID, types.ContainerRemoveOptions{Force: true})
		return fmt.Errorf("failed to start container: %w", err)
	}

	log.Printf("Started container %s for instance %s", containerID[:12], instanceID)
	m.containers[instanceID] = containerID

	// 延迟检查容器状态（给应用一点启动时间）
	// 如果容器立即退出，说明应用有问题，记录日志帮助调试
	go func() {
		time.Sleep(2 * time.Second)
		info, err := m.client.ContainerInspect(m.ctx, containerID)
		if err == nil && !info.State.Running {
			// 容器已退出，获取最后20行日志
			logs, err := m.client.ContainerLogs(m.ctx, containerID, types.ContainerLogsOptions{
				ShowStdout: true,
				ShowStderr: true,
				Tail:       "20",
			})
			if err == nil && logs != nil {
				var buf bytes.Buffer
				io.Copy(&buf, logs)
				logs.Close()
				logOutput := strings.TrimSpace(buf.String())
				if logOutput != "" {
					log.Printf("⚠️  Container %s for instance %s exited (exitCode: %d). Last logs:\n%s",
						containerID[:12], instanceID, info.State.ExitCode, logOutput)
				} else {
					log.Printf("⚠️  Container %s for instance %s exited (exitCode: %d) - no logs",
						containerID[:12], instanceID, info.State.ExitCode)
				}
			} else {
				log.Printf("⚠️  Container %s for instance %s exited (exitCode: %d) - could not read logs: %v",
					containerID[:12], instanceID, info.State.ExitCode, err)
			}
		}
	}()

	return nil
}

// StopApp 停止应用容器
func (m *DockerManager) StopApp(instanceID string) error {
	containerID, exists := m.containers[instanceID]
	if !exists {
		// 尝试通过容器名查找
		containerName := fmt.Sprintf("plum-app-%s", instanceID)
		info, err := m.client.ContainerInspect(m.ctx, containerName)
		if err != nil {
			return nil // 容器不存在，已经停止
		}
		containerID = info.ID
		if !info.State.Running {
			return nil // 容器已停止
		}
	}

	// 先尝试优雅停止（SIGTERM）
	timeoutSeconds := 5
	if err := m.client.ContainerStop(m.ctx, containerID, container.StopOptions{Timeout: &timeoutSeconds}); err != nil {
		log.Printf("Failed to stop container %s: %v", containerID[:12], err)
		// 如果优雅停止失败，强制删除
		m.client.ContainerRemove(m.ctx, containerID, types.ContainerRemoveOptions{Force: true})
	} else {
		log.Printf("Stopped container %s for instance %s", containerID[:12], instanceID)
	}

	// 删除容器
	if err := m.client.ContainerRemove(m.ctx, containerID, types.ContainerRemoveOptions{Force: true}); err != nil {
		log.Printf("Failed to remove container %s: %v", containerID[:12], err)
	}

	delete(m.containers, instanceID)
	return nil
}

// IsRunning 检查容器是否正在运行
func (m *DockerManager) IsRunning(instanceID string) bool {
	containerID, exists := m.containers[instanceID]
	containerName := fmt.Sprintf("plum-app-%s", instanceID)

	// 如果没有记录，尝试通过容器名查找
	if !exists {
		info, err := m.client.ContainerInspect(m.ctx, containerName)
		if err != nil {
			return false
		}
		containerID = info.ID
		// 检查容器状态
		if !info.State.Running {
			// 容器已停止，清理记录
			delete(m.containers, instanceID)
			log.Printf("Container %s for instance %s is not running (status: %s)",
				containerID[:12], instanceID, info.State.Status)
			return false
		}
		// 更新记录
		m.containers[instanceID] = containerID
		return true
	}

	// 有记录，检查容器状态
	info, err := m.client.ContainerInspect(m.ctx, containerID)
	if err != nil {
		// 容器不存在（可能被删除），清理记录
		delete(m.containers, instanceID)
		log.Printf("Container %s for instance %s not found, cleaned up", containerID[:12], instanceID)
		return false
	}

	// 检查容器是否真的在运行
	if !info.State.Running {
		// 容器已停止，清理记录
		delete(m.containers, instanceID)
		log.Printf("Container %s for instance %s is not running (status: %s, exitCode: %d)",
			containerID[:12], instanceID, info.State.Status, info.State.ExitCode)
		return false
	}

	return true
}

// GetStatus 获取容器状态
func (m *DockerManager) GetStatus(instanceID string) (AppStatus, error) {
	running := m.IsRunning(instanceID)
	if !running {
		return AppStatus{
			InstanceID: instanceID,
			Running:    false,
		}, nil
	}

	containerID := m.containers[instanceID]
	if containerID == "" {
		return AppStatus{
			InstanceID: instanceID,
			Running:    false,
		}, nil
	}

	info, err := m.client.ContainerInspect(m.ctx, containerID)
	if err != nil {
		return AppStatus{
			InstanceID: instanceID,
			Running:    false,
		}, nil
	}

	return AppStatus{
		InstanceID:  instanceID,
		Running:     info.State.Running,
		ContainerID: containerID,
		Pid:         info.State.Pid, // 容器PID（通常是0）
	}, nil
}

// ListRunning 列出所有运行中的实例ID
func (m *DockerManager) ListRunning() []string {
	var running []string
	// 检查所有已知的容器
	for instanceID := range m.containers {
		if m.IsRunning(instanceID) {
			running = append(running, instanceID)
		}
	}
	// 也检查可能通过容器名启动但不在记录中的容器
	// 通过列出所有 plum-app-* 容器来发现
	containerList, err := m.client.ContainerList(m.ctx, types.ContainerListOptions{
		All: true, // 包括已停止的容器
	})
	if err != nil {
		log.Printf("Failed to list containers: %v", err)
		return running
	}
	prefix := "plum-app-"
	for _, container := range containerList {
		for _, name := range container.Names {
			// 容器名格式：/plum-app-{instanceID}
			if strings.HasPrefix(name, "/"+prefix) {
				instanceID := strings.TrimPrefix(name, "/"+prefix)
				// 检查是否已经在 running 列表中
				found := false
				for _, r := range running {
					if r == instanceID {
						found = true
						break
					}
				}
				if !found && container.State == "running" {
					running = append(running, instanceID)
					// 更新容器记录
					m.containers[instanceID] = container.ID
				}
			}
		}
	}
	return running
}

// getMemoryLimit 从环境变量获取内存限制（字节）
func getMemoryLimit() int64 {
	memoryStr := os.Getenv("PLUM_CONTAINER_MEMORY")
	if memoryStr == "" {
		return 0 // 无限制
	}
	// 支持格式：512m, 1g, 2048 (字节)
	// 简单解析（实际应该更完善）
	memoryStr = strings.ToLower(memoryStr)
	var memory int64
	var unit string
	if strings.HasSuffix(memoryStr, "m") {
		fmt.Sscanf(memoryStr, "%d%s", &memory, &unit)
		memory *= 1024 * 1024 // MB to bytes
	} else if strings.HasSuffix(memoryStr, "g") {
		fmt.Sscanf(memoryStr, "%d%s", &memory, &unit)
		memory *= 1024 * 1024 * 1024 // GB to bytes
	} else {
		fmt.Sscanf(memoryStr, "%d", &memory)
	}
	return memory
}

// getCPULimit 从环境变量获取CPU限制（纳秒）
func getCPULimit() int64 {
	cpuStr := os.Getenv("PLUM_CONTAINER_CPUS")
	if cpuStr == "" {
		return 0 // 无限制
	}
	// 支持格式：1.0, 2, 0.5 (CPU核数)
	var cpus float64
	fmt.Sscanf(cpuStr, "%f", &cpus)
	return int64(cpus * 1e9) // 转换为纳秒
}

// getHostLibraryPaths 从环境变量获取宿主机库路径列表
// 格式：PLUM_HOST_LIB_PATHS=/usr/lib,/usr/local/lib,/opt/qt/lib
func getHostLibraryPaths() []string {
	libPathsStr := os.Getenv("PLUM_HOST_LIB_PATHS")
	if libPathsStr == "" {
		return nil // 不挂载任何宿主机库路径
	}

	// 解析逗号分隔的路径列表
	paths := strings.Split(libPathsStr, ",")
	var validPaths []string
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path != "" {
			// 规范化路径（移除尾随斜杠）
			path = strings.TrimSuffix(path, "/")
			validPaths = append(validPaths, path)
		}
	}
	return validPaths
}

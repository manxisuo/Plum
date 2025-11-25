package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
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
		// 注意：删除容器不会删除数据卷中的数据（数据存储在宿主机的数据目录中）
		// 新容器会挂载同一个数据目录，所以业务数据不会丢失
		if !existing.State.Running {
			log.Printf("Removing existing stopped container %s (data will be preserved in host data directory)", containerName)
			m.client.ContainerRemove(m.ctx, containerName, types.ContainerRemoveOptions{Force: true})
		}
	}

	// 确定使用的镜像
	var imageName string
	var isImageApp bool
	if app.ArtifactType == "image" && app.ImageRepository != "" && app.ImageTag != "" {
		// 镜像应用：使用指定的镜像
		imageName = fmt.Sprintf("%s:%s", app.ImageRepository, app.ImageTag)
		isImageApp = true
		log.Printf("Using image: %s", imageName)
	} else {
		// ZIP 应用：使用基础镜像
		baseImage := os.Getenv("PLUM_BASE_IMAGE")
		if baseImage == "" {
			baseImage = "alpine:latest" // 默认基础镜像
		}
		imageName = baseImage
		isImageApp = false
		log.Printf("Using base image: %s", imageName)
	}

	// 准备启动命令
	var cmdParts []string
	if isImageApp {
		// 镜像应用：使用 StartCmd，如果没有则使用镜像的默认命令（nil）
		cmdline := strings.TrimSpace(app.StartCmd)
		if cmdline != "" {
			cmdParts = strings.Fields(cmdline)
		}
		// 如果 cmdParts 为空，Docker 会使用镜像的默认命令
	} else {
		// ZIP 应用：使用 start.sh
		cmdline := strings.TrimSpace(app.StartCmd)
		if cmdline == "" {
			cmdline = "./start.sh"
		}
		cmdParts = strings.Fields(cmdline)
		if len(cmdParts) == 0 {
			cmdParts = []string{"./start.sh"}
		}
	}

	// 构建环境变量列表
	envVars := []string{
		fmt.Sprintf("PLUM_INSTANCE_ID=%s", app.InstanceID),
		fmt.Sprintf("PLUM_APP_NAME=%s", app.AppName),
		fmt.Sprintf("PLUM_APP_VERSION=%s", app.AppVersion),
		fmt.Sprintf("WORKER_NODE_ID=%s", m.config.NodeID),
	}

	// 添加 CONTROLLER_BASE 和 CONTROLLER_GRPC_ADDR 环境变量
	// 从 CONTROLLER_BASE 提取主机部分，转换为 gRPC 地址
	// 同时准备主机映射（ExtraHosts），以便容器内可以解析 Controller 主机名
	controllerBase := m.config.Controller
	var extraHosts []string
	networkMode := getNetworkMode()
	if controllerBase != "" {
		// 设置 CONTROLLER_BASE 环境变量（HTTP API 地址）
		// 在 bridge 模式下，如果原始地址是 localhost/127.0.0.1，需要调整为容器可访问的地址
		controllerBaseForContainer := controllerBase
		if networkMode != container.NetworkMode("host") {
			// bridge 模式下，如果是 localhost/127.0.0.1，需要调整为可访问的地址
			u, err := url.Parse(controllerBase)
			if err == nil {
				originalHost := u.Hostname()
				if originalHost == "localhost" || originalHost == "127.0.0.1" {
					// 使用 Docker 网关 IP 或 PLUM_CONTROLLER_HOST
					overrideHost := os.Getenv("PLUM_CONTROLLER_HOST")
					if overrideHost != "" {
						controllerBaseForContainer = fmt.Sprintf("%s://%s:%s", u.Scheme, overrideHost, u.Port())
					} else {
						// 默认使用 Docker 网关 IP
						port := u.Port()
						if port == "" {
							port = "8080"
						}
						controllerBaseForContainer = fmt.Sprintf("%s://172.17.0.1:%s", u.Scheme, port)
					}
					log.Printf("Adjusted CONTROLLER_BASE from %s to %s for container (bridge mode)", controllerBase, controllerBaseForContainer)
				}
			}
		}
		envVars = append(envVars, fmt.Sprintf("CONTROLLER_BASE=%s", controllerBaseForContainer))
		log.Printf("Set CONTROLLER_BASE=%s for instance %s", controllerBaseForContainer, instanceID)

		if networkMode == container.NetworkMode("host") {
			// host 模式下，容器和宿主机共享网络
			// 根据 CONTROLLER_BASE 的主机名构建 gRPC 地址
			u, err := url.Parse(controllerBase)
			if err == nil {
				originalHost := u.Hostname()
				if originalHost == "localhost" || originalHost == "127.0.0.1" {
					// 如果是 localhost/127.0.0.1，使用 127.0.0.1:9090
					envVars = append(envVars, "CONTROLLER_GRPC_ADDR=127.0.0.1:9090")
					log.Printf("Set CONTROLLER_GRPC_ADDR=127.0.0.1:9090 for instance %s (host network mode, localhost)", instanceID)
				} else {
					// 如果是主机名（如 plum-controller），使用主机名:9090
					grpcAddr := fmt.Sprintf("%s:9090", originalHost)
					envVars = append(envVars, fmt.Sprintf("CONTROLLER_GRPC_ADDR=%s", grpcAddr))
					log.Printf("Set CONTROLLER_GRPC_ADDR=%s for instance %s (host network mode)", grpcAddr, instanceID)
				}
			} else {
				// 解析失败，使用默认值
				envVars = append(envVars, "CONTROLLER_GRPC_ADDR=127.0.0.1:9090")
				log.Printf("Failed to parse CONTROLLER_BASE, using default CONTROLLER_GRPC_ADDR=127.0.0.1:9090 for instance %s", instanceID)
			}
		} else {
			// bridge 模式下，需要解析地址并添加主机映射
			controllerGrpcAddr, hostMapping := extractGrpcAddrWithMapping(controllerBase)
			if controllerGrpcAddr != "" {
				envVars = append(envVars, fmt.Sprintf("CONTROLLER_GRPC_ADDR=%s", controllerGrpcAddr))
				log.Printf("Set CONTROLLER_GRPC_ADDR=%s for instance %s", controllerGrpcAddr, instanceID)
			}
			if hostMapping != "" {
				extraHosts = append(extraHosts, hostMapping)
				log.Printf("Adding host mapping for instance %s: %s", instanceID, hostMapping)
			}
		}
	}

	// 自动注入 MAIN_CONTROL_BASE 环境变量（如果 MainControl 服务可用）
	// 所有应用都可以使用此环境变量来访问 MainControl 服务（如发送进度更新）
	mainControlBase := m.discoverMainControlBase()
	if mainControlBase != "" {
		// 根据网络模式调整 MainControl 地址（类似 CONTROLLER_BASE 的处理）
		mainControlBaseForContainer := mainControlBase
		if networkMode != container.NetworkMode("host") {
			// bridge 模式下，需要确保地址可以从容器访问
			u, err := url.Parse(mainControlBase)
			if err == nil {
				originalHost := u.Hostname()
				// 如果是 localhost/127.0.0.1，需要调整为容器可访问的地址
				if originalHost == "localhost" || originalHost == "127.0.0.1" {
					overrideHost := os.Getenv("PLUM_CONTROLLER_HOST")
					if overrideHost == "" {
						overrideHost = "172.17.0.1" // 默认使用 Docker 网关 IP
					}
					port := u.Port()
					if port == "" {
						port = "4000" // MainControl 默认端口
					}
					mainControlBaseForContainer = fmt.Sprintf("%s://%s:%s", u.Scheme, overrideHost, port)
					log.Printf("Adjusted MAIN_CONTROL_BASE from %s to %s for container (bridge mode)", mainControlBase, mainControlBaseForContainer)
				}
				// 如果服务发现返回的是其他 IP（如 192.168.x.x），在 bridge 模式下应该可以直接访问
				// 但如果 MainControl 运行在 host 模式的容器中，返回的 IP 是宿主机 IP，容器应该可以访问
			}
		}
		envVars = append(envVars, fmt.Sprintf("MAIN_CONTROL_BASE=%s", mainControlBaseForContainer))
		log.Printf("Set MAIN_CONTROL_BASE=%s for instance %s", mainControlBaseForContainer, instanceID)
	}
	// 注意：如果服务发现失败，不注入环境变量，应用可以使用默认值或通过其他方式获取地址

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

	// 自动添加 LD_LIBRARY_PATH（仅对 ZIP 应用，如果应用目录有lib子目录）
	// 这对于Qt等需要共享库的应用很有用
	if !isImageApp && appDir != "" {
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
	}

	// 配置端口映射
	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}

	if isImageApp && app.PortMappings != "" {
		// 镜像应用：从 PortMappings JSON 解析端口映射
		var portMaps []map[string]interface{}
		if err := json.Unmarshal([]byte(app.PortMappings), &portMaps); err == nil {
			for _, pm := range portMaps {
				hostPort, _ := pm["host"].(float64)
				containerPort, _ := pm["container"].(float64)
				if hostPort > 0 && containerPort > 0 {
					port, err := nat.NewPort("tcp", strconv.Itoa(int(containerPort)))
					if err != nil {
						log.Printf("Failed to parse container port %d for instance %s: %v", int(containerPort), instanceID, err)
						continue
					}
					exposedPorts[port] = struct{}{}
					portBindings[port] = []nat.PortBinding{
						{HostIP: "0.0.0.0", HostPort: strconv.Itoa(int(hostPort))},
					}
					log.Printf("Port mapping for instance %s: %d -> %d", instanceID, int(hostPort), int(containerPort))
				}
			}
		} else {
			log.Printf("Failed to parse port mappings for instance %s: %v", instanceID, err)
		}
	} else if !isImageApp && appDir != "" {
		// ZIP 应用：根据 meta.ini 暴露端口
		metaPath := filepath.Join(appDir, "meta.ini")
		if endpoints, err := ParseMetaINI(metaPath); err == nil && len(endpoints) > 0 {
			for _, ep := range endpoints {
				if ep.Port <= 0 {
					continue
				}
				proto := strings.ToLower(ep.Protocol)
				switch proto {
				case "", "http", "https", "grpc":
					proto = "tcp"
				case "tcp", "udp":
					// keep
				default:
					proto = "tcp"
				}
				port, err := nat.NewPort(proto, strconv.Itoa(ep.Port))
				if err != nil {
					log.Printf("Failed to parse service port %d/%s for instance %s: %v", ep.Port, proto, instanceID, err)
					continue
				}
				if _, exists := exposedPorts[port]; exists {
					continue
				}
				exposedPorts[port] = struct{}{}
				portBindings[port] = []nat.PortBinding{
					{HostIP: "0.0.0.0", HostPort: strconv.Itoa(ep.Port)},
				}
			}
		}
	}

	// 创建容器配置
	containerConfig := &container.Config{
		Image:        imageName,
		Env:          envVars,
		ExposedPorts: exposedPorts,
		// 设置容器自动删除（停止后）
		// 但我们需要手动管理，所以不设置AutoRemove
	}

	// 设置启动命令（仅当有命令时）
	if len(cmdParts) > 0 {
		containerConfig.Cmd = cmdParts
	}

	// 设置工作目录（仅对 ZIP 应用）
	if !isImageApp {
		containerConfig.WorkingDir = "/app"
	}

	// 构建挂载列表
	var mounts []mount.Mount

	if !isImageApp && appDir != "" {
		// ZIP 应用：挂载应用目录
		mounts = []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: appDir,
				Target: "/app",
			},
		}
	} else if isImageApp {
		// 镜像应用：为每个实例创建独立的数据目录，用于持久化业务数据
		// 数据目录：{AGENT_DATA_DIR}/{nodeID}/data/{instanceID}
		dataDir := os.Getenv("AGENT_DATA_DIR")
		if dataDir == "" {
			dataDir = "/tmp/plum-agent"
		}
		instanceDataDir := filepath.Join(dataDir, m.config.NodeID, "data", instanceID)

		// 确保数据目录存在
		if err := os.MkdirAll(instanceDataDir, 0755); err != nil {
			log.Printf("Warning: failed to create data directory %s: %v", instanceDataDir, err)
		} else {
			// 挂载数据目录到容器的 /app/data（应用可以在这里存储持久化数据）
			// 如果应用使用其他路径（如 /data），可以通过环境变量配置
			mounts = append(mounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: instanceDataDir,
				Target: "/app/data",
			})
			log.Printf("Mounted data directory %s to /app/data for instance %s", instanceDataDir, instanceID)
		}
	}

	// 可选：挂载宿主机的库路径（仅对 ZIP 应用，用于共享系统库）
	// 这样可以避免每个应用都自包含相同的库，减少重复
	if !isImageApp {
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
	}

	// 创建主机配置（挂载应用目录和库路径）
	hostConfig := &container.HostConfig{
		Mounts: mounts,
		// 资源限制（可选，从环境变量读取）
		Resources: container.Resources{
			Memory:   getMemoryLimit(), // 从环境变量或默认值
			NanoCPUs: getCPULimit(),    // CPU限制
		},
		PortBindings: portBindings,
		// 网络模式：从环境变量读取，默认为 bridge（已在前面定义 networkMode）
		NetworkMode: networkMode,
		// 自动重启策略：不自动重启（由Agent管理）
		RestartPolicy: container.RestartPolicy{Name: "no"},
		// 添加主机映射（用于容器内解析 Controller 主机名，仅在非 host 模式下需要）
		ExtraHosts: extraHosts,
		// 安全选项：禁用 seccomp 以支持 Ubuntu 24.04 的 glibc 2.38（需要 clone3 系统调用）
		// 这对于在旧版 Docker 环境中运行基于 Ubuntu 24.04 的镜像很重要
		SecurityOpt: []string{"seccomp=unconfined"},
	}

	// 如果使用 host 网络模式，端口映射不需要
	// 但 ExtraHosts 仍然需要，因为 Docker 可能会覆盖容器的 /etc/hosts
	if networkMode == container.NetworkMode("host") {
		hostConfig.PortBindings = nil
		// 注意：即使使用 host 网络模式，ExtraHosts 仍然有效，可以确保容器内能解析主机名
		// 如果 extraHosts 为空，则不需要设置
		if len(extraHosts) == 0 {
			hostConfig.ExtraHosts = nil
		}
		log.Printf("Using host network mode for instance %s (port mappings disabled)", instanceID)
	}

	if len(portBindings) > 0 {
		var ports []string
		for port := range portBindings {
			ports = append(ports, port.Port()+"/"+port.Proto())
		}
		log.Printf("Exposing ports for instance %s: %s", instanceID, strings.Join(ports, ", "))
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
		// 如果优雅停止失败，强制停止（但不删除容器，保留以便调试）
		if err := m.client.ContainerKill(m.ctx, containerID, "SIGKILL"); err != nil {
			log.Printf("Failed to kill container %s: %v", containerID[:12], err)
		}
	} else {
		log.Printf("Stopped container %s for instance %s", containerID[:12], instanceID)
	}

	// 注意：不删除容器，保留已停止的容器以便：
	// 1. 查看日志：docker logs plum-app-{instanceID}
	// 2. 调试问题：docker inspect plum-app-{instanceID}
	// 3. 检查退出状态：docker ps -a
	// 容器会在下次启动时被删除并重新创建（如果配置有变化）
	log.Printf("Container %s stopped but kept for debugging (use 'docker rm' to remove manually)", containerID[:12])

	// 检查是否需要自动清理数据目录
	// 如果设置了 PLUM_AUTO_CLEAN_DATA=true，容器停止时自动清理数据目录
	autoCleanData := os.Getenv("PLUM_AUTO_CLEAN_DATA")
	if autoCleanData == "true" || autoCleanData == "1" {
		m.cleanupInstanceData(instanceID)
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
	containerID := m.containers[instanceID]
	containerName := fmt.Sprintf("plum-app-%s", instanceID)

	// 如果没有记录，尝试通过容器名查找
	if containerID == "" {
		info, err := m.client.ContainerInspect(m.ctx, containerName)
		if err != nil {
			return AppStatus{
				InstanceID: instanceID,
				Running:    false,
				ExitCode:   -1, // 容器不存在，视为异常
			}, nil
		}
		containerID = info.ID
		if !info.State.Running {
			// 容器已停止，返回退出码
			return AppStatus{
				InstanceID:  instanceID,
				Running:     false,
				ContainerID: containerID,
				ExitCode:    info.State.ExitCode,
			}, nil
		}
		// 更新记录
		m.containers[instanceID] = containerID
		return AppStatus{
			InstanceID:  instanceID,
			Running:     true,
			ContainerID: containerID,
			Pid:         info.State.Pid,
		}, nil
	}

	info, err := m.client.ContainerInspect(m.ctx, containerID)
	if err != nil {
		// 容器不存在，清理记录
		delete(m.containers, instanceID)
		return AppStatus{
			InstanceID: instanceID,
			Running:    false,
			ExitCode:   -1, // 容器不存在，视为异常
		}, nil
	}

	if !info.State.Running {
		// 容器已停止，返回退出码
		return AppStatus{
			InstanceID:  instanceID,
			Running:     false,
			ContainerID: containerID,
			ExitCode:    info.State.ExitCode,
		}, nil
	}

	return AppStatus{
		InstanceID:  instanceID,
		Running:     info.State.Running,
		ContainerID: containerID,
		Pid:         info.State.Pid,
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

// readFileFromContainer 从容器内读取文件内容
func (m *DockerManager) readFileFromContainer(containerID, filePath string) (string, error) {
	// 使用 docker exec 读取文件（要求容器运行中）
	execConfig := types.ExecConfig{
		Cmd:          []string{"cat", filePath},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := m.client.ContainerExecCreate(m.ctx, containerID, execConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create exec: %w", err)
	}

	// 连接到 exec
	attachResp, err := m.client.ContainerExecAttach(m.ctx, execResp.ID, types.ExecStartCheck{})
	if err != nil {
		return "", fmt.Errorf("failed to attach exec: %w", err)
	}
	defer attachResp.Close()

	// 读取输出（Docker exec 输出格式：8字节头部 + 数据）
	// 头部格式：STREAM_TYPE (1 byte) + PADDING (3 bytes) + LENGTH (4 bytes, big-endian)
	var allData bytes.Buffer
	_, err = io.Copy(&allData, attachResp.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to read exec output: %w", err)
	}

	// 检查 exec 是否成功
	execInspect, err := m.client.ContainerExecInspect(m.ctx, execResp.ID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect exec: %w", err)
	}
	if execInspect.ExitCode != 0 {
		// 尝试提取错误信息（跳过头部）
		data := allData.Bytes()
		errorMsg := ""
		if len(data) > 8 {
			errorMsg = string(data[8:])
		} else {
			errorMsg = allData.String()
		}
		return "", fmt.Errorf("exec failed with exit code %d, output: %s", execInspect.ExitCode, errorMsg)
	}

	// 解析 Docker 流格式：跳过 8 字节头部，提取实际内容
	data := allData.Bytes()
	content := ""
	if len(data) > 8 {
		// 跳过头部，读取所有数据块
		pos := 0
		var parts []string
		for pos < len(data) {
			if pos+8 > len(data) {
				break
			}
			// 读取长度（big-endian，4字节，位置 4-7）
			length := int(data[pos+4])<<24 | int(data[pos+5])<<16 | int(data[pos+6])<<8 | int(data[pos+7])
			if length <= 0 || pos+8+length > len(data) {
				break
			}
			// 提取数据（跳过 8 字节头部）
			part := string(data[pos+8 : pos+8+length])
			parts = append(parts, part)
			pos += 8 + length
		}
		content = strings.Join(parts, "")
	} else {
		// 如果没有头部，直接使用全部内容
		content = allData.String()
	}

	return strings.TrimSpace(content), nil
}

// getNetworkMode 从环境变量获取 Docker 容器网络模式
// 支持的值：host, bridge, none（默认：bridge）
func getNetworkMode() container.NetworkMode {
	mode := os.Getenv("PLUM_CONTAINER_NETWORK_MODE")
	if mode == "" {
		mode = "bridge" // 默认使用 bridge 网络
	}

	switch mode {
	case "host":
		log.Printf("Using host network mode for containers")
		return container.NetworkMode("host")
	case "bridge":
		return container.NetworkMode("bridge")
	case "none":
		return container.NetworkMode("none")
	default:
		log.Printf("Unknown network mode '%s', using bridge", mode)
		return container.NetworkMode("bridge")
	}
}

// extractGrpcAddrWithMapping 从 Controller HTTP 地址提取 gRPC 地址和主机映射
// 输入：http://host:port 或 https://host:port
// 输出：gRPC 地址（host:9090）和主机映射字符串（格式：hostname:ip，用于 ExtraHosts）
// 对于容器环境，如果 Controller 在宿主机上，使用 Docker 网关 IP 或 PLUM_CONTROLLER_HOST
func extractGrpcAddrWithMapping(controllerBase string) (grpcAddr string, hostMapping string) {
	if controllerBase == "" {
		return "", ""
	}

	// 解析 URL
	u, err := url.Parse(controllerBase)
	if err != nil {
		log.Printf("Failed to parse CONTROLLER_BASE: %v", err)
		return "", ""
	}

	originalHost := u.Hostname()
	if originalHost == "" {
		return "", ""
	}

	// 确定 Controller 的实际 IP 地址
	var controllerIP string
	overrideHost := os.Getenv("PLUM_CONTROLLER_HOST")
	if overrideHost != "" {
		// 如果设置了 PLUM_CONTROLLER_HOST，优先使用它
		controllerIP = overrideHost
		log.Printf("Using PLUM_CONTROLLER_HOST=%s for Controller", controllerIP)
	} else if originalHost == "localhost" || originalHost == "127.0.0.1" {
		// localhost/127.0.0.1 在容器中指向容器自己，需要使用 Docker 网关 IP
		controllerIP = "172.17.0.1"
		log.Printf("Controller is on localhost, using Docker gateway IP 172.17.0.1")
	} else {
		// 对于其他主机名（如 plum-controller），需要解析
		// 如果宿主机上配置了 /etc/hosts，Agent 可以解析，但容器内无法解析
		// 所以我们需要添加主机映射，将 originalHost 映射到实际 IP
		// 尝试解析主机名
		controllerIP = resolveControllerHost(originalHost)
		if controllerIP == "" {
			// 无法解析，使用默认网关 IP
			controllerIP = "172.17.0.1"
			log.Printf("Cannot resolve Controller host %s, using Docker gateway IP 172.17.0.1", originalHost)
		} else {
			log.Printf("Resolved Controller host %s to %s", originalHost, controllerIP)
		}
	}

	// 构建 gRPC 地址
	// 如果 originalHost 不是 IP 地址，使用 originalHost（容器内通过 ExtraHosts 映射）
	// 如果 originalHost 是 localhost/127.0.0.1，直接使用 controllerIP
	if originalHost == "localhost" || originalHost == "127.0.0.1" {
		grpcAddr = fmt.Sprintf("%s:9090", controllerIP)
		// 不需要主机映射（因为直接使用 IP）
		hostMapping = ""
	} else {
		// 使用原始主机名，但添加主机映射
		grpcAddr = fmt.Sprintf("%s:9090", originalHost)
		// 添加主机映射：originalHost -> controllerIP
		hostMapping = fmt.Sprintf("%s:%s", originalHost, controllerIP)
	}

	return grpcAddr, hostMapping
}

// resolveControllerHost 尝试解析 Controller 主机名
// 返回 IP 地址，如果无法解析则返回空字符串
func resolveControllerHost(hostname string) string {
	// 如果已经是 IP 地址，直接返回
	if net.ParseIP(hostname) != nil {
		return hostname
	}

	// 尝试通过 DNS 解析（但这可能不够，因为 /etc/hosts 中的条目不会通过 DNS）
	// 更好的方法是读取 /etc/hosts 文件
	hostsFile := "/etc/hosts"
	if file, err := os.Open(hostsFile); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			// 跳过注释和空行
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			// 解析 /etc/hosts 格式：IP hostname1 hostname2 ...
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				ip := fields[0]
				// 检查是否匹配主机名
				for i := 1; i < len(fields); i++ {
					if fields[i] == hostname {
						log.Printf("Found %s -> %s in /etc/hosts", hostname, ip)
						return ip
					}
				}
			}
		}
	}

	// 如果 /etc/hosts 中没有找到，尝试使用 net.LookupHost（DNS 解析）
	// 但这可能返回多个 IP，我们取第一个
	addrs, err := net.LookupHost(hostname)
	if err == nil && len(addrs) > 0 {
		return addrs[0]
	}

	return ""
}

// cleanupInstanceData 清理实例的数据目录
// 当容器停止时，如果启用了自动清理，会删除该实例的数据目录
func (m *DockerManager) cleanupInstanceData(instanceID string) {
	dataDir := os.Getenv("AGENT_DATA_DIR")
	if dataDir == "" {
		dataDir = "/tmp/plum-agent"
	}
	instanceDataDir := filepath.Join(dataDir, m.config.NodeID, "data", instanceID)

	// 检查目录是否存在
	if _, err := os.Stat(instanceDataDir); os.IsNotExist(err) {
		return // 目录不存在，无需清理
	}

	// 删除数据目录
	if err := os.RemoveAll(instanceDataDir); err != nil {
		log.Printf("Warning: failed to cleanup data directory %s for instance %s: %v", instanceDataDir, instanceID, err)
	} else {
		log.Printf("Cleaned up data directory %s for instance %s", instanceDataDir, instanceID)
	}
}

// discoverMainControlBase 通过服务发现获取 MainControl 的地址（DockerManager 方法）
func (m *DockerManager) discoverMainControlBase() string {
	return discoverMainControlBase(m.config.Controller)
}

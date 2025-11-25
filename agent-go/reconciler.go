package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Reconciler 协调器
type Reconciler struct {
	baseDir            string
	http               *HTTPClient
	controller         string
	processManager     AppManager           // 进程模式管理器（用于 ZIP 应用）
	dockerManager      AppManager           // 容器模式管理器（用于镜像应用，以及 ZIP 应用在 docker 模式下）
	stopSentTimes      map[string]int64     // 停止信号发送时间（用于优雅停止）
	restartStartTimes  map[string]time.Time // 性能监控：记录重启开始时间
	knownInstances     map[string]bool      // 已知实例列表（用于故障检测）
	instanceTypes      map[string]string    // 实例类型映射：instanceID -> "image" 或 "zip"
	completedInstances map[string]bool      // 已完成的镜像应用实例（避免重复标记）
	registeredServices map[string]bool      // 已注册服务的实例（避免重复注册）
}

func NewReconciler(baseDir string, http *HTTPClient, controller string, nodeID string) *Reconciler {
	EnsureDir(baseDir)

	// 根据环境变量选择 ZIP 应用的运行模式
	runMode := GetRunMode()
	log.Printf("Using app run mode for ZIP apps: %s", runMode)
	log.Printf("Image apps will always use Docker mode")

	config := ManagerConfig{
		BaseDir:    baseDir,
		HTTP:       http,
		Controller: controller,
		NodeID:     nodeID,
	}

	// 总是创建 DockerManager（镜像应用需要，ZIP 应用在 docker 模式下也需要）
	dockerManager, err := NewDockerManager(config)
	if err != nil {
		log.Printf("Warning: failed to create docker manager: %v (image apps will not work)", err)
		dockerManager = nil
	}

	// 根据环境变量创建 ProcessManager（ZIP 应用在 process 模式下需要）
	var processManager AppManager
	if runMode == "process" {
		processManager = NewProcessManager(config)
	} else {
		// docker 模式下，ZIP 应用也使用 dockerManager
		processManager = dockerManager
	}

	return &Reconciler{
		baseDir:            baseDir,
		http:               http,
		controller:         controller,
		processManager:     processManager,
		dockerManager:      dockerManager,
		stopSentTimes:      make(map[string]int64),
		restartStartTimes:  make(map[string]time.Time),
		knownInstances:     make(map[string]bool),
		instanceTypes:      make(map[string]string),
		completedInstances: make(map[string]bool),
		registeredServices: make(map[string]bool),
	}
}

// getAppManager 根据应用类型返回对应的管理器
// artifactType: "image" 或 "zip"
func (r *Reconciler) getAppManager(artifactType string) AppManager {
	if artifactType == "image" {
		// 镜像应用总是使用 DockerManager
		return r.dockerManager
	}
	// ZIP 应用根据环境变量选择管理器
	return r.processManager
}

// getAppManagerByInstanceID 根据实例ID返回对应的管理器
func (r *Reconciler) getAppManagerByInstanceID(instanceID string) AppManager {
	artifactType, exists := r.instanceTypes[instanceID]
	if !exists {
		// 如果不知道类型，默认尝试两个管理器
		// 先检查 dockerManager（因为镜像应用必须用这个）
		if r.dockerManager != nil && r.dockerManager.IsRunning(instanceID) {
			return r.dockerManager
		}
		// 再检查 processManager
		if r.processManager != nil && r.processManager.IsRunning(instanceID) {
			return r.processManager
		}
		// 如果都不在运行，默认返回 processManager（向后兼容）
		return r.processManager
	}
	return r.getAppManager(artifactType)
}

// Sync 同步状态
func (r *Reconciler) Sync(assignments []Assignment) {
	keep := make(map[string]bool)
	runningCount := 0

	// 更新已知实例列表
	newKnownInstances := make(map[string]bool)
	for _, a := range assignments {
		newKnownInstances[a.InstanceID] = true
		if a.Desired == "Running" {
			keep[a.InstanceID] = true
			runningCount++
		} else {
			// 如果期望状态不是 Running，清除已完成标记（允许重新启动）
			delete(r.completedInstances, a.InstanceID)
		}
	}
	// 清理不再存在的实例的已完成标记
	for instanceID := range r.completedInstances {
		if !newKnownInstances[instanceID] {
			delete(r.completedInstances, instanceID)
		}
	}
	r.knownInstances = newKnownInstances

	// 标记需要停止的实例
	for _, a := range assignments {
		if a.Desired != "Running" {
			r.markForStop(a.InstanceID)
		}
	}

	// 检测已退出的进程（故障检测）
	r.reapExited(assignments)

	// 停止不需要的实例
	r.ensureStoppedExcept(keep)

	// 启动需要的实例
	for _, a := range assignments {
		if a.Desired == "Running" {
			r.ensureRunning(a)
		}
	}
}

// ensureRunning 确保实例运行
func (r *Reconciler) ensureRunning(a Assignment) {
	// 确定 artifact 类型（默认为 zip，向后兼容）
	artifactType := a.ArtifactType
	if artifactType == "" {
		artifactType = "zip"
	}
	log.Printf("DEBUG: ensureRunning - InstanceID=%s, ArtifactType=%s, ImageRepository=%s, ImageTag=%s, ArtifactURL=%s",
		a.InstanceID, artifactType, a.ImageRepository, a.ImageTag, a.ArtifactURL)

	// 根据应用类型选择管理器
	appManager := r.getAppManager(artifactType)
	if appManager == nil {
		log.Printf("ERROR: No suitable app manager for artifact type: %s (InstanceID: %s)", artifactType, a.InstanceID)
		if artifactType == "image" {
			log.Printf("ERROR: DockerManager is nil! Cannot start image-based app.")
		}
		return
	}

	// 检查是否已运行
	if appManager.IsRunning(a.InstanceID) {
		return // 应用已在运行
	}

	// 对于镜像应用，如果已经正常完成（exitCode: 0），不再自动重启
	// 除非用户明确要求（通过重新部署）
	if artifactType == "image" {
		if r.completedInstances[a.InstanceID] {
			// 已经标记为已完成，跳过重启（不打印日志，避免重复）
			return
		}
		// 检查容器状态，如果正常退出，也跳过
		status, err := appManager.GetStatus(a.InstanceID)
		if err == nil && !status.Running && status.ExitCode == 0 {
			r.completedInstances[a.InstanceID] = true // 记录已标记为已完成
			return
		}
	}

	// 实例未运行，需要启动
	log.Printf("Instance %s not running, will start", a.InstanceID)

	// 记录实例类型
	r.instanceTypes[a.InstanceID] = artifactType

	// 性能监控：记录启动开始时间
	r.restartStartTimes[a.InstanceID] = time.Now()

	var appDir string
	var err error

	if artifactType == "image" {
		// 镜像应用：直接使用 Docker 镜像启动，不需要下载 ZIP
		log.Printf("Starting image-based app: %s:%s (InstanceID: %s)", a.ImageRepository, a.ImageTag, a.InstanceID)
		if a.ImageRepository == "" || a.ImageTag == "" {
			log.Printf("ERROR: ImageRepository or ImageTag is empty! ArtifactType=%s, ImageRepository=%s, ImageTag=%s",
				a.ArtifactType, a.ImageRepository, a.ImageTag)
			return
		}
		appDir = "" // 镜像应用不需要 appDir
		err = appManager.StartApp(a.InstanceID, a, appDir)
	} else {
		// ZIP 应用：下载、解压、启动
		instDir := filepath.Join(r.baseDir, a.InstanceID)
		EnsureDir(instDir)

		zipPath := filepath.Join(instDir, "pkg.zip")
		appDir = filepath.Join(instDir, "app")

		// 下载artifact
		if !FileExists(zipPath) {
			artifactURL := a.ArtifactURL
			// 规范化URL
			if !strings.HasPrefix(artifactURL, "http://") && !strings.HasPrefix(artifactURL, "https://") {
				if strings.HasPrefix(artifactURL, "/") {
					artifactURL = r.controller + artifactURL
				} else {
					artifactURL = r.controller + "/" + artifactURL
				}
			}

			data, err := r.http.Get(artifactURL)
			if err != nil {
				log.Printf("Failed to download artifact: %v", err)
				return
			}
			if err := os.WriteFile(zipPath, data, 0644); err != nil {
				log.Printf("Failed to save artifact: %v", err)
				return
			}
			log.Printf("Downloaded artifact to %s, size=%d", zipPath, len(data))
		}

		// 解压
		EnsureDir(appDir)
		startSh := filepath.Join(appDir, "start.sh")
		if !FileExists(startSh) {
			if err := UnzipFile(zipPath, appDir); err != nil {
				log.Printf("Failed to unzip: %v", err)
				return
			}
		}

		// 确保start.sh有执行权限（容器模式也需要）
		if FileExists(startSh) {
			if err := os.Chmod(startSh, 0755); err != nil {
				log.Printf("Warning: failed to chmod start.sh: %v", err)
			}
		}

		// 确保应用可执行文件有执行权限
		// 遍历应用目录，找到所有没有扩展名的文件（很可能是可执行文件）
		// 或者检查文件是否是ELF可执行文件
		if err := ensureExecutablePermissions(appDir); err != nil {
			log.Printf("Warning: failed to set executable permissions: %v", err)
		}

		// 使用AppManager启动应用
		err = appManager.StartApp(a.InstanceID, a, appDir)
	}

	if err != nil {
		log.Printf("Failed to start app: %v", err)
		return
	}

	log.Printf("Started instance %s", a.InstanceID)
	r.postStatus(a.InstanceID, "Running", 0, true)

	// 性能监控：记录重启时间
	if startTime, exists := r.restartStartTimes[a.InstanceID]; exists {
		restartDuration := time.Since(startTime)
		log.Printf("性能监控: 实例 %s 重启耗时 %v", a.InstanceID, restartDuration)
		delete(r.restartStartTimes, a.InstanceID)
	}
}

// ensureStoppedExcept 停止不需要的实例
func (r *Reconciler) ensureStoppedExcept(keep map[string]bool) {
	now := time.Now().Unix()

	// 获取所有运行中的实例（合并两个管理器的结果）
	// 这样可以发现所有运行中的实例，包括那些不在 assignments 中的（已删除的实例）
	allRunning := make(map[string]bool)
	if r.processManager != nil {
		for _, id := range r.processManager.ListRunning() {
			allRunning[id] = true
		}
	}
	if r.dockerManager != nil {
		for _, id := range r.dockerManager.ListRunning() {
			allRunning[id] = true
		}
	}

	// 检查需要停止的实例（已在 stopSentTimes 中的）
	for instanceID := range r.stopSentTimes {
		if keep[instanceID] {
			// 仍在keep列表中，清除停止标记
			delete(r.stopSentTimes, instanceID)
			continue
		}

		appManager := r.getAppManagerByInstanceID(instanceID)
		if appManager == nil {
			continue
		}

		if !appManager.IsRunning(instanceID) {
			// 已经停止，清理状态
			delete(r.stopSentTimes, instanceID)
			delete(r.instanceTypes, instanceID)
			r.postStatus(instanceID, "Stopped", 0, true)
			r.deleteServices(instanceID)
			continue
		}

		// 应用还在运行，需要停止
		if r.stopSentTimes[instanceID] == 0 {
			// 第一次尝试停止
			if err := appManager.StopApp(instanceID); err != nil {
				log.Printf("Failed to stop app %s: %v", instanceID, err)
			} else {
				r.stopSentTimes[instanceID] = now
				log.Printf("Sent stop signal to instance %s", instanceID)
			}
		} else if now-r.stopSentTimes[instanceID] >= 5 {
			// 5秒后强制停止（DockerManager已经处理了强制停止）
			// 这里只需要清理状态
			if !appManager.IsRunning(instanceID) {
				delete(r.stopSentTimes, instanceID)
				delete(r.instanceTypes, instanceID)
				r.postStatus(instanceID, "Stopped", 0, true)
				r.deleteServices(instanceID)
				log.Printf("Stopped instance %s", instanceID)
			}
		}
	}

	// 检查所有运行中的实例，如果不在keep列表中，需要停止
	// 这包括：1) Desired=Stopped 的实例，2) assignment 被删除的实例
	for instanceID := range allRunning {
		if keep[instanceID] {
			continue // 应该在运行，跳过
		}

		// 这个实例不应该运行，但正在运行，需要停止
		// 但是，如果容器刚启动（在已知实例列表中，说明之前有 assignment），
		// 可能是 assignment 同步延迟，给一点时间等待同步
		if r.knownInstances[instanceID] {
			// 这个实例之前存在，但现在 Desired 不是 Running
			// 可能是用户停止了部署，应该停止
			if _, exists := r.stopSentTimes[instanceID]; !exists {
				// 还没有标记停止，现在标记并立即尝试停止
				r.markForStop(instanceID)
				appManager := r.getAppManagerByInstanceID(instanceID)
				if appManager != nil {
					// 立即尝试停止（不等待下一次循环）
					if err := appManager.StopApp(instanceID); err != nil {
						log.Printf("Failed to stop app %s: %v", instanceID, err)
					} else {
						r.stopSentTimes[instanceID] = now
						log.Printf("Sent stop signal to instance %s (not in keep list, was known)", instanceID)
					}
				}
			}
		} else {
			// 这个实例不在已知列表中，可能是：
			// 1. 手动启动的容器（不在 Agent 管理范围内）- 不应该停止
			// 2. 刚启动的容器，assignment 还没有同步到 Agent - 应该等待
			// 3. 旧的残留容器（已经运行很久）- 应该停止
			// 我们通过检查容器启动时间来判断：如果启动时间 < 15秒，可能是刚启动的，等待同步
			appManager := r.getAppManagerByInstanceID(instanceID)
			if appManager != nil {
				// 尝试获取容器启动时间（仅对 Docker 容器有效）
				shouldStop := true
				if dockerMgr, ok := appManager.(*DockerManager); ok {
					containerName := fmt.Sprintf("plum-app-%s", instanceID)
					info, err := dockerMgr.client.ContainerInspect(dockerMgr.ctx, containerName)
					if err == nil && info.State.Running && info.State.StartedAt != "" {
						// 解析容器启动时间（Docker API 返回 RFC3339 格式字符串）
						startedAt, err := time.Parse(time.RFC3339Nano, info.State.StartedAt)
						if err == nil {
							age := time.Since(startedAt)
							if age < 15*time.Second {
								// 容器刚启动不久（< 15秒），可能是 assignment 同步延迟，等待一下
								log.Printf("Instance %s is running but not in known instances, started %v ago, waiting for assignment sync (may be just deployed)", instanceID, age)
								shouldStop = false
							}
						}
					}
				}

				if shouldStop {
					// 容器已经运行了一段时间，或者无法获取启动时间，停止它
					if _, exists := r.stopSentTimes[instanceID]; !exists {
						r.markForStop(instanceID)
						if err := appManager.StopApp(instanceID); err != nil {
							log.Printf("Failed to stop app %s: %v", instanceID, err)
						} else {
							r.stopSentTimes[instanceID] = now
							log.Printf("Sent stop signal to instance %s (not in keep list, not in known instances, may be manually started or old container)", instanceID)
						}
					}
				}
			}
		}
	}
}

// markForStop 标记需要停止的实例
func (r *Reconciler) markForStop(instanceID string) {
	if _, exists := r.stopSentTimes[instanceID]; !exists {
		r.stopSentTimes[instanceID] = 0 // 标记需要停止
	}
}

// reapExited 检测并清理意外退出的应用
// 这是故障检测的核心：检查所有应该运行的实例是否真的在运行
func (r *Reconciler) reapExited(assignments []Assignment) {
	// 检查所有期望运行但实际未运行的实例
	for _, a := range assignments {
		if a.Desired != "Running" {
			continue // 只检查期望运行的实例
		}

		// 检查实例是否真的在运行
		// 根据应用类型选择管理器
		artifactType := a.ArtifactType
		if artifactType == "" {
			artifactType = "zip"
		}
		appManager := r.getAppManager(artifactType)
		if appManager == nil {
			continue
		}

		if !appManager.IsRunning(a.InstanceID) {
			// 实例未运行，但期望运行
			// 这可能是进程意外退出（被kill等）
			// ProcessManager.IsRunning() 已经清理了内部状态，这里只需要上报

			// 检查是否是我们主动停止的
			if _, wasStopping := r.stopSentTimes[a.InstanceID]; !wasStopping {
				// 检查是否已经标记为已完成（避免重复标记）
				if r.completedInstances[a.InstanceID] {
					continue // 已经标记为已完成，跳过
				}

				// 不是主动停止的，获取退出状态
				status, err := appManager.GetStatus(a.InstanceID)
				exitCode := -1
				if err == nil && !status.Running {
					exitCode = status.ExitCode
				}

				// 判断应用类型
				artifactType := a.ArtifactType
				if artifactType == "" {
					artifactType = "zip"
				}

				// 对于镜像应用，如果正常退出（exitCode: 0），视为"已完成"而不是"失败"
				// 对于 ZIP 应用或异常退出，视为"失败"并重启
				if artifactType == "image" && exitCode == 0 {
					// 镜像应用正常退出，标记为已完成（只标记一次）
					phase := "Completed"
					healthy := true
					r.postStatus(a.InstanceID, phase, exitCode, healthy)
					r.completedInstances[a.InstanceID] = true // 记录已标记为已完成
					log.Printf("✅ Instance %s (image app) completed successfully (exitCode: %d), marked as Completed", a.InstanceID, exitCode)
					// 不自动重启已完成的任务
				} else {
					// 异常退出或 ZIP 应用退出，视为失败
					log.Printf("⚠️  Detected instance %s process died unexpectedly (was not stopping, exitCode: %d)", a.InstanceID, exitCode)
					phase := "Failed"
					healthy := false
					r.postStatus(a.InstanceID, phase, exitCode, healthy)
					log.Printf("✅ Reported instance %s as Failed, will restart in next ensureRunning", a.InstanceID)
				}

				// 清理停止标记（如果有）
				delete(r.stopSentTimes, a.InstanceID)
			} else {
				log.Printf("Instance %s is stopping (was marked for stop), skip restart", a.InstanceID)
			}
		}
	}
}

// StopAll 停止所有实例
func (r *Reconciler) StopAll() {
	// 标记所有实例需要停止
	// 由于我们不知道所有实例ID，这里通过清空keep map来触发停止
	r.Sync([]Assignment{})

	// 等待最多7秒，让所有应用停止
	for i := 0; i < 70; i++ {
		r.ensureStoppedExcept(make(map[string]bool))
		time.Sleep(100 * time.Millisecond)
		// 检查是否还有运行中的实例
		// 由于无法直接获取所有实例，这里简化处理
		if len(r.stopSentTimes) == 0 {
			break
		}
	}
}

// postStatus 上报状态
func (r *Reconciler) postStatus(instanceID, phase string, exitCode int, healthy bool) {
	status := InstanceStatus{
		InstanceID: instanceID,
		Phase:      phase,
		ExitCode:   exitCode,
		Healthy:    healthy,
		TsUnix:     time.Now().Unix(),
	}
	url := r.controller + "/v1/instances/status"
	if err := r.http.PostJSON(url, status); err != nil {
		log.Printf("Failed to post status: %v", err)
	}
}

// RegisterServices 注册服务
// 采用增量注册模式：只注册meta.ini中定义的服务端点，不影响手动注册的其他服务
// 这样可以在真实应用实例上手动注册额外服务，而不会被Agent覆盖
// 使用缓存机制避免重复注册：如果实例已经注册过服务且容器还在运行，就跳过注册
func (r *Reconciler) RegisterServices(instanceID, nodeID, ip string, assignment *Assignment) {
	// 检查是否已经注册过服务
	if r.registeredServices[instanceID] {
		// 检查容器是否还在运行（如果容器重启了，需要重新注册）
		if assignment != nil && assignment.ArtifactType == "image" {
			if r.dockerManager != nil && r.dockerManager.IsRunning(instanceID) {
				// 容器还在运行，且已经注册过，跳过注册
				return
			}
			// 容器不在运行，清除缓存，稍后会重新注册
			delete(r.registeredServices, instanceID)
		} else {
			// ZIP 应用，如果已经注册过就跳过
			return
		}
	}

	var endpoints []ServiceEndpoint

	// 优先尝试从本地 meta.ini 读取（ZIP 应用）
	metaPath := filepath.Join(r.baseDir, instanceID, "app", "meta.ini")
	if FileExists(metaPath) {
		eps, err := ParseMetaINI(metaPath)
		if err == nil && len(eps) > 0 {
			endpoints = eps
		}
	}

	// 如果是镜像应用且没有从本地 meta.ini 读取到服务，尝试从容器内读取 meta.ini
	if len(endpoints) == 0 && assignment != nil && assignment.ArtifactType == "image" {
		// 尝试从容器内读取 meta.ini（镜像应用可能在镜像中包含 meta.ini）
		// 常见的 meta.ini 路径：/app/meta.ini, /meta.ini, /app/bin/meta.ini
		metaContent := r.readMetaFromContainer(instanceID)
		if metaContent != "" {
			// 将内容写入临时文件，然后解析
			instDir := filepath.Join(r.baseDir, instanceID)
			EnsureDir(instDir) // 确保目录存在
			tmpFile := filepath.Join(instDir, "meta.ini.tmp")
			if err := os.WriteFile(tmpFile, []byte(metaContent), 0644); err == nil {
				eps, err := ParseMetaINI(tmpFile)
				if err != nil {
					log.Printf("ERROR: Failed to parse meta.ini for instance %s: %v", instanceID, err)
				} else if len(eps) == 0 {
					// 没有服务端点（如 gRPC Worker），这是正常的，不需要警告
					// 标记为已处理，避免重复读取
					r.registeredServices[instanceID] = true
				} else {
					endpoints = eps
					log.Printf("Read meta.ini from container for instance %s, found %d endpoint(s)", instanceID, len(eps))
					for i, ep := range eps {
						log.Printf("  Endpoint %d: %s:%s:%d", i+1, ep.ServiceName, ep.Protocol, ep.Port)
					}
				}
				// 清理临时文件
				os.Remove(tmpFile)
			} else {
				log.Printf("ERROR: Failed to write temp meta.ini file for instance %s: %v", instanceID, err)
			}
		}
	}

	// 如果还是没有读取到服务，且是镜像应用，尝试从 PortMappings 提取
	if len(endpoints) == 0 && assignment != nil && assignment.ArtifactType == "image" && assignment.PortMappings != "" {
		// 从 PortMappings JSON 解析端口映射
		// 支持两种格式：
		// 1. 简单格式：{"host": 4100, "container": 4100} - 使用 AppName 作为服务名称
		// 2. 完整格式：{"serviceName": "planArea", "protocol": "http", "host": 4100, "container": 4100} - 使用指定的服务名称和协议
		var portMaps []map[string]interface{}
		if err := json.Unmarshal([]byte(assignment.PortMappings), &portMaps); err == nil {
			// 默认服务名称（如果端口映射中没有指定）
			defaultServiceName := assignment.AppName
			if defaultServiceName == "" {
				defaultServiceName = "unknown"
			}
			// 为每个端口映射创建一个服务端点
			for _, pm := range portMaps {
				hostPort, _ := pm["host"].(float64)
				containerPort, _ := pm["container"].(float64)
				// 使用 host 端口（因为使用 host 网络模式时，容器端口就是 host 端口）
				// 但如果没有 host 端口，使用 container 端口
				port := int(hostPort)
				if port <= 0 {
					port = int(containerPort)
				}
				if port > 0 {
					// 优先使用端口映射中指定的服务名称和协议（与 meta.ini 格式一致）
					serviceName, _ := pm["serviceName"].(string)
					if serviceName == "" {
						serviceName = defaultServiceName
					}
					protocol, _ := pm["protocol"].(string)
					if protocol == "" {
						protocol = "http" // 默认协议为 http
					}
					endpoints = append(endpoints, ServiceEndpoint{
						ServiceName: serviceName,
						Protocol:    protocol,
						Port:        port,
					})
					log.Printf("Registered service endpoint from port mapping: %s:%s:%d (instance: %s)", serviceName, protocol, port, instanceID)
				}
			}
		} else {
			log.Printf("Failed to parse port mappings for instance %s: %v", instanceID, err)
		}
	}

	// 如果没有找到任何服务端点，标记为已处理（避免重复读取），然后返回
	if len(endpoints) == 0 {
		r.registeredServices[instanceID] = true
		return
	}

	reg := ServiceRegistration{
		InstanceID: instanceID,
		NodeID:     nodeID,
		IP:         ip,
		Endpoints:  endpoints,
	}
	// 使用增量注册模式（不传replace=true），只添加/更新服务端点
	// 不会删除手动注册的其他服务端点
	url := r.controller + "/v1/services/register"
	if err := r.http.PostJSON(url, reg); err != nil {
		log.Printf("ERROR: Failed to register services for instance %s: %v", instanceID, err)
	} else {
		log.Printf("Successfully registered %d service endpoint(s) for instance %s", len(endpoints), instanceID)
		// 标记为已注册，避免重复注册
		r.registeredServices[instanceID] = true
	}
}

// HeartbeatServices 服务心跳
func (r *Reconciler) HeartbeatServices(instanceID string) {
	url := r.controller + "/v1/services/heartbeat"
	if err := r.http.PostJSON(url, HeartbeatRequest{InstanceID: instanceID}); err != nil {
		log.Printf("Failed to heartbeat services: %v", err)
	}
}

// readMetaFromContainer 从容器内读取 meta.ini 文件
// 尝试多个常见路径：/app/meta.ini, /meta.ini, /app/bin/meta.ini
func (r *Reconciler) readMetaFromContainer(instanceID string) string {
	// 需要访问 DockerManager，进行类型断言
	if r.dockerManager == nil {
		return ""
	}

	// 类型断言获取 DockerManager
	dm, ok := r.dockerManager.(*DockerManager)
	if !ok {
		return ""
	}

	// 获取容器ID
	containerID, exists := dm.containers[instanceID]
	if !exists {
		// 尝试通过容器名查找
		containerName := fmt.Sprintf("plum-app-%s", instanceID)
		info, err := dm.client.ContainerInspect(dm.ctx, containerName)
		if err != nil {
			return ""
		}
		containerID = info.ID
		// 注意：如果容器已停止，readFileFromContainer 会使用 docker cp 读取文件
		// 但更重要的是确保容器正常运行，这样服务才能被健康检查通过
	}

	// 尝试多个常见路径
	metaPaths := []string{"/app/meta.ini", "/meta.ini", "/app/bin/meta.ini", "/usr/local/app/meta.ini"}
	for _, metaPath := range metaPaths {
		content, err := dm.readFileFromContainer(containerID, metaPath)
		if err == nil && content != "" {
			log.Printf("Found meta.ini at %s in container %s", metaPath, containerID[:12])
			return content
		}
	}

	return ""
}

// deleteServices 删除服务
func (r *Reconciler) deleteServices(instanceID string) {
	url := fmt.Sprintf("%s/v1/services?instanceId=%s", r.controller, instanceID)
	r.http.Delete(url)
	// 清除注册缓存，这样如果实例重新启动，可以重新注册服务
	delete(r.registeredServices, instanceID)
}

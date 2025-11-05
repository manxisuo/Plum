package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Reconciler 协调器
type Reconciler struct {
	baseDir           string
	http              *HTTPClient
	controller        string
	appManager        AppManager           // 应用管理器（统一接口）
	stopSentTimes     map[string]int64     // 停止信号发送时间（用于优雅停止）
	restartStartTimes map[string]time.Time // 性能监控：记录重启开始时间
	knownInstances    map[string]bool      // 已知实例列表（用于故障检测）
}

func NewReconciler(baseDir string, http *HTTPClient, controller string) *Reconciler {
	EnsureDir(baseDir)

	// 根据环境变量选择运行模式
	runMode := GetRunMode()
	log.Printf("Using app run mode: %s", runMode)

	config := ManagerConfig{
		BaseDir:    baseDir,
		HTTP:       http,
		Controller: controller,
	}

	appManager, err := NewAppManager(runMode, config)
	if err != nil {
		log.Printf("Failed to create app manager, falling back to process mode: %v", err)
		appManager = NewProcessManager(config)
	}

	return &Reconciler{
		baseDir:           baseDir,
		http:              http,
		controller:        controller,
		appManager:        appManager,
		stopSentTimes:     make(map[string]int64),
		restartStartTimes: make(map[string]time.Time),
		knownInstances:    make(map[string]bool),
	}
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
	// 检查是否已运行
	if r.appManager.IsRunning(a.InstanceID) {
		return // 应用已在运行
	}

	// 实例未运行，需要启动
	log.Printf("Instance %s not running, will start", a.InstanceID)

	instDir := filepath.Join(r.baseDir, a.InstanceID)
	EnsureDir(instDir)

	zipPath := filepath.Join(instDir, "pkg.zip")
	appDir := filepath.Join(instDir, "app")

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

	// 性能监控：记录启动开始时间
	r.restartStartTimes[a.InstanceID] = time.Now()

	// 使用AppManager启动应用
	if err := r.appManager.StartApp(a.InstanceID, a, appDir); err != nil {
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

	// 获取所有运行中的实例（通过AppManager.ListRunning）
	// 这样可以发现所有运行中的实例，包括那些不在 assignments 中的（已删除的实例）
	allRunning := r.appManager.ListRunning()

	// 检查需要停止的实例（已在 stopSentTimes 中的）
	for instanceID := range r.stopSentTimes {
		if keep[instanceID] {
			// 仍在keep列表中，清除停止标记
			delete(r.stopSentTimes, instanceID)
			continue
		}

		if !r.appManager.IsRunning(instanceID) {
			// 已经停止，清理状态
			delete(r.stopSentTimes, instanceID)
			r.postStatus(instanceID, "Stopped", 0, true)
			r.deleteServices(instanceID)
			continue
		}

		// 应用还在运行，需要停止
		if r.stopSentTimes[instanceID] == 0 {
			// 第一次尝试停止
			if err := r.appManager.StopApp(instanceID); err != nil {
				log.Printf("Failed to stop app %s: %v", instanceID, err)
			} else {
				r.stopSentTimes[instanceID] = now
				log.Printf("Sent stop signal to instance %s", instanceID)
			}
		} else if now-r.stopSentTimes[instanceID] >= 5 {
			// 5秒后强制停止（DockerManager已经处理了强制停止）
			// 这里只需要清理状态
			if !r.appManager.IsRunning(instanceID) {
				delete(r.stopSentTimes, instanceID)
				r.postStatus(instanceID, "Stopped", 0, true)
				r.deleteServices(instanceID)
				log.Printf("Stopped instance %s", instanceID)
			}
		}
	}

	// 检查所有运行中的实例，如果不在keep列表中，需要停止
	// 这包括：1) Desired=Stopped 的实例，2) assignment 被删除的实例
	for _, instanceID := range allRunning {
		if keep[instanceID] {
			continue // 应该在运行，跳过
		}

		// 这个实例不应该运行，但正在运行，需要停止
		if _, exists := r.stopSentTimes[instanceID]; !exists {
			// 还没有标记停止，现在标记并立即尝试停止
			r.markForStop(instanceID)
			// 立即尝试停止（不等待下一次循环）
			if err := r.appManager.StopApp(instanceID); err != nil {
				log.Printf("Failed to stop app %s: %v", instanceID, err)
			} else {
				r.stopSentTimes[instanceID] = now
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
		if !r.appManager.IsRunning(a.InstanceID) {
			// 实例未运行，但期望运行
			// 这可能是进程意外退出（被kill等）
			// ProcessManager.IsRunning() 已经清理了内部状态，这里只需要上报

			// 检查是否是我们主动停止的
			if _, wasStopping := r.stopSentTimes[a.InstanceID]; !wasStopping {
				// 不是主动停止的，说明是意外退出
				log.Printf("⚠️  Detected instance %s process died unexpectedly (was not stopping)", a.InstanceID)

				// 获取退出状态（尝试）
				status, err := r.appManager.GetStatus(a.InstanceID)
				exitCode := 0
				if err == nil && !status.Running {
					// 可能是非零退出码，但GetStatus可能无法获取
					// 默认认为是失败退出（因为不是正常停止）
					exitCode = -1
				}

				// 上报失败状态
				phase := "Failed"
				healthy := false
				r.postStatus(a.InstanceID, phase, exitCode, healthy)
				log.Printf("✅ Reported instance %s as Failed, will restart in next ensureRunning", a.InstanceID)

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
func (r *Reconciler) RegisterServices(instanceID, nodeID, ip string) {
	metaPath := filepath.Join(r.baseDir, instanceID, "app", "meta.ini")
	if !FileExists(metaPath) {
		return
	}

	endpoints, err := ParseMetaINI(metaPath)
	if err != nil || len(endpoints) == 0 {
		return
	}

	reg := ServiceRegistration{
		InstanceID: instanceID,
		NodeID:     nodeID,
		IP:         ip,
		Endpoints:  endpoints,
	}
	// 使用增量注册模式（不传replace=true），只添加/更新meta.ini中的服务端点
	// 不会删除手动注册的其他服务端点
	url := r.controller + "/v1/services/register"
	if err := r.http.PostJSON(url, reg); err != nil {
		log.Printf("Failed to register services: %v", err)
	}
}

// HeartbeatServices 服务心跳
func (r *Reconciler) HeartbeatServices(instanceID string) {
	url := r.controller + "/v1/services/heartbeat"
	if err := r.http.PostJSON(url, HeartbeatRequest{InstanceID: instanceID}); err != nil {
		log.Printf("Failed to heartbeat services: %v", err)
	}
}

// deleteServices 删除服务
func (r *Reconciler) deleteServices(instanceID string) {
	url := fmt.Sprintf("%s/v1/services?instanceId=%s", r.controller, instanceID)
	r.http.Delete(url)
}

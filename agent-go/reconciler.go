package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// ProcessState 进程状态
type ProcessState struct {
	Cmd          *exec.Cmd
	StopSentTime int64
}

// Reconciler 协调器
type Reconciler struct {
	baseDir           string
	http              *HTTPClient
	controller        string
	instances         map[string]*ProcessState
	restartStartTimes map[string]time.Time // 性能监控：记录重启开始时间
}

func NewReconciler(baseDir string, http *HTTPClient, controller string) *Reconciler {
	EnsureDir(baseDir)
	return &Reconciler{
		baseDir:           baseDir,
		http:              http,
		controller:        controller,
		instances:         make(map[string]*ProcessState),
		restartStartTimes: make(map[string]time.Time),
	}
}

// Sync 同步状态
func (r *Reconciler) Sync(assignments []Assignment) {
	keep := make(map[string]bool)
	runningCount := 0
	for _, a := range assignments {
		if a.Desired == "Running" {
			keep[a.InstanceID] = true
			runningCount++
		}
	}

	r.reapExited()
	r.ensureStoppedExcept(keep)
	for _, a := range assignments {
		if a.Desired == "Running" {
			r.ensureRunning(a)
		}
	}
}

// ensureRunning 确保实例运行
func (r *Reconciler) ensureRunning(a Assignment) {
	// 检查是否已运行 - 优化：使用更快的进程检测机制
	if state, exists := r.instances[a.InstanceID]; exists {
		// 使用/proc/pid/stat进行快速进程检测
		if state.Cmd.Process != nil {
			pid := state.Cmd.Process.Pid
			statData, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
			if err == nil {
				// 解析进程状态
				statStr := string(statData)
				if idx := strings.LastIndex(statStr, ")"); idx != -1 && idx+2 < len(statStr) {
					state := statStr[idx+2]
					// Z = 僵尸进程，其他状态认为进程还活着
					if state != 'Z' {
						return // 进程仍在运行
					}
				}
			} else {
				// 如果无法读取/proc/pid/stat，回退到Signal(0)检测
				err := state.Cmd.Process.Signal(syscall.Signal(0))
				if err == nil {
					return // 进程仍在运行
				}
			}
		}
		// 进程已死，清理状态
		delete(r.instances, a.InstanceID)
		log.Printf("Instance %s process died, will restart", a.InstanceID)

		// 性能监控：记录重启开始时间
		r.restartStartTimes[a.InstanceID] = time.Now()
	}

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

	// 启动进程
	os.Chmod(startSh, 0755)
	cmdline := strings.TrimSpace(a.StartCmd)
	if cmdline == "" {
		cmdline = "./start.sh"
	}

	cmd := exec.Command("sh", "-c", cmdline)
	cmd.Dir = appDir
	cmd.Env = append(os.Environ(),
		"PLUM_INSTANCE_ID="+a.InstanceID,
		"PLUM_APP_NAME="+a.AppName,
		"PLUM_APP_VERSION="+a.AppVersion,
	)
	// 创建新的进程组
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start process: %v", err)
		return
	}

	log.Printf("Started instance %s, PID=%d", a.InstanceID, cmd.Process.Pid)
	r.instances[a.InstanceID] = &ProcessState{Cmd: cmd}
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
	for id, state := range r.instances {
		if keep[id] {
			continue
		}

		if state.StopSentTime == 0 {
			// 发送SIGTERM到进程组
			pgid := -state.Cmd.Process.Pid
			syscall.Kill(pgid, syscall.SIGTERM)
			state.StopSentTime = now
			log.Printf("Sent SIGTERM to instance %s", id)
		} else if now-state.StopSentTime >= 5 {
			// 5秒后强制SIGKILL
			pgid := -state.Cmd.Process.Pid
			syscall.Kill(pgid, syscall.SIGKILL)
			state.Cmd.Wait()
			r.postStatus(id, "Stopped", 0, true)
			r.deleteServices(id)
			delete(r.instances, id)
			log.Printf("Killed instance %s", id)
		}
	}
}

// reapExited 清理已退出的进程
func (r *Reconciler) reapExited() {
	for id, state := range r.instances {
		// 检查进程是否还活着
		if state.Cmd.Process == nil {
			log.Printf("Instance %s has nil Process, skipping", id)
			continue
		}

		// 检查进程状态（读取/proc/pid/stat）
		pid := state.Cmd.Process.Pid
		statData, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		processAlive := false
		if err == nil {
			// stat格式: pid (comm) state ...
			// 提取状态字段（第3个字段）
			statStr := string(statData)
			// 找到第二个')'后的第一个字符就是状态
			if idx := strings.LastIndex(statStr, ")"); idx != -1 && idx+2 < len(statStr) {
				state := statStr[idx+2]
				// Z = 僵尸, 其他状态认为进程还活着
				processAlive = (state != 'Z')
			}
		}

		if !processAlive {
			log.Printf("Detected instance %s process died (PID %d)", id, pid)
			// 进程已退出，尝试Wait获取退出状态
			var exitCode int
			if state.Cmd.ProcessState != nil {
				exitCode = state.Cmd.ProcessState.ExitCode()
			} else {
				// ProcessState为nil，手动Wait（非阻塞）
				go state.Cmd.Wait() // 异步Wait，避免阻塞
				time.Sleep(100 * time.Millisecond)
				if state.Cmd.ProcessState != nil {
					exitCode = state.Cmd.ProcessState.ExitCode()
				} else {
					exitCode = -1
				}
			}

			if state.StopSentTime > 0 {
				r.postStatus(id, "Stopped", 0, true)
				log.Printf("Reaped instance %s (stopped), exit=%d", id, exitCode)
			} else {
				phase := "Exited"
				healthy := exitCode == 0
				if !healthy {
					phase = "Failed"
				}
				r.postStatus(id, phase, exitCode, healthy)
				log.Printf("Reaped instance %s (died), exit=%d, phase=%s", id, exitCode, phase)
			}
			delete(r.instances, id)
		}
	}
}

// StopAll 停止所有实例
func (r *Reconciler) StopAll() {
	r.Sync([]Assignment{})
	// 等待最多7秒
	for i := 0; i < 70; i++ {
		r.ensureStoppedExcept(make(map[string]bool))
		r.reapExited()
		if len(r.instances) == 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
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

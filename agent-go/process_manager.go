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

// ProcessManager 进程模式管理器
// 封装现有的进程启动逻辑
type ProcessManager struct {
	config    ManagerConfig
	processes map[string]*exec.Cmd // instanceID -> cmd
}

// NewProcessManager 创建进程管理器
func NewProcessManager(config ManagerConfig) *ProcessManager {
	return &ProcessManager{
		config:    config,
		processes: make(map[string]*exec.Cmd),
	}
}

// findRunningProcessByInstanceID 通过实例ID查找运行中的进程
// 返回进程PID，如果没找到返回0
func (m *ProcessManager) findRunningProcessByInstanceID(instanceID string) int {
	// 方法1：检查 processes map 中的进程
	if cmd, exists := m.processes[instanceID]; exists {
		if cmd != nil && cmd.Process != nil {
			if m.IsRunning(instanceID) {
				return cmd.Process.Pid
			}
		}
	}

	// 方法2：通过环境变量 PLUM_INSTANCE_ID 查找系统中实际运行的进程
	// 使用 ps 命令查找：ps -eo pid,cmd | grep PLUM_INSTANCE_ID=instanceID
	// 注意：这个方法会查找所有包含 PLUM_INSTANCE_ID=instanceID 的进程
	psCmd := exec.Command("sh", "-c", fmt.Sprintf("ps -eo pid,cmd | grep 'PLUM_INSTANCE_ID=%s' | grep -v grep | head -1 | awk '{print $1}'", instanceID))
	output, err := psCmd.Output()
	if err == nil && len(output) > 0 {
		var pid int
		if _, err := fmt.Sscanf(string(output), "%d", &pid); err == nil && pid > 0 {
			// 验证进程确实存在
			if err := syscall.Kill(pid, syscall.Signal(0)); err == nil {
				log.Printf("Found running process for instance %s (PID=%d, not in processes map)", instanceID, pid)
				return pid
			}
		}
	}

	return 0
}

// StartApp 启动应用进程
func (m *ProcessManager) StartApp(instanceID string, app Assignment, appDir string) error {
	// 检查是否已运行（包括 processes map 和系统中实际运行的进程）
	if pid := m.findRunningProcessByInstanceID(instanceID); pid > 0 {
		log.Printf("Instance %s already running, PID=%d", instanceID, pid)
		// 如果进程不在 processes map 中，尝试恢复记录（但无法恢复 exec.Cmd，只能记录 PID）
		// 这里只记录日志，不恢复 exec.Cmd，因为无法从 PID 创建 exec.Cmd
		// 但至少可以防止重复启动
		return nil
	}

	// 清理 processes map 中的无效记录
	delete(m.processes, instanceID)

	startSh := filepath.Join(appDir, "start.sh")
	if err := os.Chmod(startSh, 0755); err != nil {
		log.Printf("Failed to chmod start.sh: %v", err)
	}

	cmdline := strings.TrimSpace(app.StartCmd)
	if cmdline == "" {
		cmdline = "./start.sh"
	}

	cmd := exec.Command("sh", "-c", cmdline)
	cmd.Dir = appDir
	cmd.Env = append(os.Environ(),
		"PLUM_INSTANCE_ID="+app.InstanceID,
		"PLUM_APP_NAME="+app.AppName,
		"PLUM_APP_VERSION="+app.AppVersion,
	)
	// 创建新的进程组
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	log.Printf("Started instance %s, PID=%d", instanceID, cmd.Process.Pid)
	m.processes[instanceID] = cmd
	return nil
}

// StopApp 停止应用进程
func (m *ProcessManager) StopApp(instanceID string) error {
	// 查找所有运行中的进程（包括 processes map 和系统中实际运行的）
	var pids []int

	// 从 processes map 中获取 PID
	if cmd, exists := m.processes[instanceID]; exists {
		if cmd != nil && cmd.Process != nil {
			pid := cmd.Process.Pid
			// 验证进程确实存在
			if err := syscall.Kill(pid, syscall.Signal(0)); err == nil {
				pids = append(pids, pid)
			}
		}
	}

	// 通过环境变量查找系统中实际运行的进程
	psCmd := exec.Command("sh", "-c", fmt.Sprintf("ps -eo pid,cmd | grep 'PLUM_INSTANCE_ID=%s' | grep -v grep | awk '{print $1}'", instanceID))
	output, err := psCmd.Output()
	if err == nil && len(output) > 0 {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			var pid int
			if _, err := fmt.Sscanf(strings.TrimSpace(line), "%d", &pid); err == nil && pid > 0 {
				// 检查是否已经在 pids 列表中
				found := false
				for _, existingPid := range pids {
					if existingPid == pid {
						found = true
						break
					}
				}
				if !found {
					// 验证进程确实存在
					if err := syscall.Kill(pid, syscall.Signal(0)); err == nil {
						pids = append(pids, pid)
						log.Printf("Found additional running process for instance %s (PID=%d)", instanceID, pid)
					}
				}
			}
		}
	}

	if len(pids) == 0 {
		log.Printf("No running process found for instance %s", instanceID)
		delete(m.processes, instanceID)
		return nil // 已经停止
	}

	// 停止所有找到的进程
	for _, pid := range pids {
		// 发送SIGTERM到进程组
		pgid := -pid
		if err := syscall.Kill(pgid, syscall.SIGTERM); err != nil {
			log.Printf("Failed to send SIGTERM to process group %d (instance %s): %v", pgid, instanceID, err)
		} else {
			log.Printf("Sent SIGTERM to process group %d (instance %s, PID=%d)", pgid, instanceID, pid)
		}
	}

	// 对于 processes map 中的进程，使用 cmd.Wait() 异步等待
	waitDone := make(chan struct{})
	if cmd, exists := m.processes[instanceID]; exists && cmd != nil && cmd.Process != nil {
		go func() {
			cmd.Wait()
			waitDone <- struct{}{}
		}()
	}

	// 等待进程退出（最多5秒）
	// 使用动态检查间隔：开始时频繁（200ms），逐渐变慢（500ms）以减少开销
	timeout := 5 * time.Second
	startTime := time.Now()
	initialInterval := 200 * time.Millisecond
	maxInterval := 500 * time.Millisecond

	checkInterval := initialInterval
	lastCheck := time.Now()

	for time.Since(startTime) < timeout {
		// 检查 cmd.Wait() 是否已完成
		select {
		case <-waitDone:
			// 进程已退出（通过 cmd.Wait() 检测到）
			log.Printf("Process for instance %s stopped gracefully (detected via Wait)", instanceID)
			goto cleanup
		default:
		}

		// 每 checkInterval 检查一次进程状态（减少轮询频率）
		if time.Since(lastCheck) >= checkInterval {
			allStopped := true
			for _, pid := range pids {
				if err := syscall.Kill(pid, syscall.Signal(0)); err == nil {
					// 进程还在运行
					allStopped = false
					break
				}
			}

			if allStopped {
				// 所有进程都已退出
				log.Printf("All processes for instance %s stopped gracefully", instanceID)
				goto cleanup
			}

			// 动态增加检查间隔（减少开销）
			if checkInterval < maxInterval {
				checkInterval = time.Duration(float64(checkInterval) * 1.2) // 每次增加20%
				if checkInterval > maxInterval {
					checkInterval = maxInterval
				}
			}

			lastCheck = time.Now()
		}

		// 短暂休眠，避免 CPU 占用过高
		time.Sleep(50 * time.Millisecond)
	}

cleanup:
	// 检查是否还有进程在运行，如果有则强制 SIGKILL
	for _, pid := range pids {
		if err := syscall.Kill(pid, syscall.Signal(0)); err == nil {
			// 进程还在运行，强制 SIGKILL
			pgid := -pid
			syscall.Kill(pgid, syscall.SIGKILL)
			log.Printf("Force killed process group %d (instance %s, PID=%d)", pgid, instanceID, pid)
			// 等待进程退出
			time.Sleep(100 * time.Millisecond)
		} else {
			elapsed := time.Since(startTime)
			log.Printf("Process %d (instance %s) stopped gracefully (took %v)", pid, instanceID, elapsed)
		}
	}

	// 如果有 processes map 中的记录，尝试 Wait
	if cmd, exists := m.processes[instanceID]; exists {
		if cmd != nil && cmd.Process != nil {
			// 不等待，因为可能已经通过其他方式停止了
			go cmd.Wait() // 异步等待，避免阻塞
		}
	}

	delete(m.processes, instanceID)
	log.Printf("Stopped all processes for instance %s", instanceID)
	return nil
}

// IsRunning 检查应用是否正在运行
func (m *ProcessManager) IsRunning(instanceID string) bool {
	cmd, exists := m.processes[instanceID]
	if !exists || cmd.Process == nil {
		return false
	}

	pid := cmd.Process.Pid

	// 方法1：使用 /proc/<pid>/stat 检测进程状态（最可靠）
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	statData, err := os.ReadFile(statPath)
	if err != nil {
		// 无法读取 /proc/<pid>/stat，说明进程已死
		delete(m.processes, instanceID)
		log.Printf("Instance %s process died (PID %d): cannot read /proc/%d/stat", instanceID, pid, pid)
		return false
	}

	// 解析进程状态：stat格式为 "pid (comm) state ..."
	// 找到第二个 ')' 后的第一个字符就是状态
	statStr := string(statData)
	idx := strings.LastIndex(statStr, ")")
	if idx == -1 || idx+2 >= len(statStr) {
		// 格式异常，回退到 Signal(0)
		if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
			delete(m.processes, instanceID)
			log.Printf("Instance %s process died (PID %d): invalid stat format, Signal(0) failed", instanceID, pid)
			return false
		}
		return true
	}

	state := statStr[idx+2]
	// Z = 僵尸进程（已死），其他状态认为进程还活着
	if state == 'Z' {
		// 僵尸进程，已经死亡
		delete(m.processes, instanceID)
		log.Printf("Instance %s process died (PID %d): zombie process", instanceID, pid)
		// 尝试 Wait() 回收僵尸进程
		go cmd.Wait()
		return false
	}

	// 方法2：双重检查，使用 Signal(0) 作为补充验证
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		// Signal(0) 失败，但 /proc 显示进程存在，可能是权限问题
		// 这种情况认为进程已死（更安全）
		delete(m.processes, instanceID)
		log.Printf("Instance %s process died (PID %d): Signal(0) failed (stat shows state=%c)", instanceID, pid, state)
		return false
	}

	return true
}

// GetStatus 获取应用状态
func (m *ProcessManager) GetStatus(instanceID string) (AppStatus, error) {
	cmd, exists := m.processes[instanceID]
	if !exists {
		return AppStatus{
			InstanceID: instanceID,
			Running:    false,
		}, nil
	}

	if cmd.Process == nil {
		return AppStatus{
			InstanceID: instanceID,
			Running:    false,
		}, nil
	}

	// 检查进程是否还活着
	running := m.IsRunning(instanceID)
	pid := 0
	if running {
		pid = cmd.Process.Pid
	}

	return AppStatus{
		InstanceID: instanceID,
		Running:    running,
		Pid:        pid,
	}, nil
}

// ListRunning 列出所有运行中的实例ID
func (m *ProcessManager) ListRunning() []string {
	runningMap := make(map[string]bool)

	// 方法1：检查 processes map 中的进程
	for instanceID, cmd := range m.processes {
		if cmd != nil && cmd.Process != nil {
			// 检查进程是否真的在运行
			if m.IsRunning(instanceID) {
				runningMap[instanceID] = true
			}
		}
	}

	// 方法2：通过环境变量查找系统中实际运行的进程
	// 使用 ps 命令查找所有包含 PLUM_INSTANCE_ID 的进程
	psCmd := exec.Command("sh", "-c", "ps -eo pid,cmd | grep 'PLUM_INSTANCE_ID=' | grep -v grep")
	output, err := psCmd.Output()
	if err == nil && len(output) > 0 {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			// 解析行：提取 PLUM_INSTANCE_ID=xxx
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				for _, part := range parts[1:] {
					if strings.HasPrefix(part, "PLUM_INSTANCE_ID=") {
						instanceID := strings.TrimPrefix(part, "PLUM_INSTANCE_ID=")
						// 验证进程确实存在（通过提取 PID）
						var pid int
						if _, err := fmt.Sscanf(parts[0], "%d", &pid); err == nil && pid > 0 {
							if err := syscall.Kill(pid, syscall.Signal(0)); err == nil {
								runningMap[instanceID] = true
								log.Printf("Found running process for instance %s (PID=%d, not in processes map)", instanceID, pid)
							}
						}
						break
					}
				}
			}
		}
	}

	// 转换为列表
	var running []string
	for instanceID := range runningMap {
		running = append(running, instanceID)
	}
	return running
}

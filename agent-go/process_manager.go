package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

func (m *ProcessManager) pidFile(instanceID string) string {
	return filepath.Join(m.config.BaseDir, instanceID, "runtime.pid")
}

func (m *ProcessManager) writePID(instanceID string, pid int) {
	path := m.pidFile(instanceID)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		log.Printf("Failed to create pid directory for %s: %v", instanceID, err)
		return
	}
	if err := os.WriteFile(path, []byte(fmt.Sprintf("%d\n", pid)), 0644); err != nil {
		log.Printf("Failed to write pid file for %s: %v", instanceID, err)
	}
}

func (m *ProcessManager) readPID(instanceID string) (int, error) {
	data, err := os.ReadFile(m.pidFile(instanceID))
	if err != nil {
		return 0, err
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return 0, fmt.Errorf("pid file empty")
	}
	pid, err := strconv.Atoi(text)
	if err != nil {
		return 0, err
	}
	return pid, nil
}

func (m *ProcessManager) removePID(instanceID string) {
	if err := os.Remove(m.pidFile(instanceID)); err != nil && !os.IsNotExist(err) {
		log.Printf("Failed to remove pid file for %s: %v", instanceID, err)
	}
}

// NewProcessManager 创建进程管理器
func NewProcessManager(config ManagerConfig) *ProcessManager {
	return &ProcessManager{
		config:    config,
		processes: make(map[string]*exec.Cmd),
	}
}

func (m *ProcessManager) findPIDByEnv(instanceID string) int {
	for _, pid := range m.findPIDsByEnv(instanceID) {
		if err := syscall.Kill(pid, syscall.Signal(0)); err == nil {
			return pid
		}
	}
	return 0
}

func (m *ProcessManager) findPIDsByEnv(instanceID string) []int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}

	key := []byte("PLUM_INSTANCE_ID=" + instanceID)
	var pids []int

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		pid, err := strconv.Atoi(name)
		if err != nil || pid <= 0 {
			continue
		}

		data, err := os.ReadFile(filepath.Join("/proc", name, "environ"))
		if err != nil {
			continue
		}

		if bytes.Contains(data, key) {
			pids = append(pids, pid)
		}
	}

	return pids
}

// findRunningProcessByInstanceID 通过实例ID查找运行中的进程
// 返回进程PID，如果没找到返回0
// 优化：优先检查 processes map，使用与 IsRunning 相同的高效查找方法
func (m *ProcessManager) findRunningProcessByInstanceID(instanceID string) int {
	if pid, err := m.readPID(instanceID); err == nil && pid > 0 {
		if err := syscall.Kill(pid, syscall.Signal(0)); err == nil {
			return pid
		}
		m.removePID(instanceID)
	}

	if cmd, exists := m.processes[instanceID]; exists {
		if cmd != nil && cmd.Process != nil {
			pid := cmd.Process.Pid
			if err := syscall.Kill(pid, syscall.Signal(0)); err == nil {
				m.writePID(instanceID, pid)
				return pid
			}
			delete(m.processes, instanceID)
		}
	}

	if pid := m.findPIDByEnv(instanceID); pid > 0 {
		m.writePID(instanceID, pid)
		return pid
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
		"WORKER_NODE_ID="+m.config.NodeID,
	)
	// 创建新的进程组
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	log.Printf("Started instance %s, PID=%d", instanceID, cmd.Process.Pid)
	m.processes[instanceID] = cmd

	actualPID := cmd.Process.Pid
	time.Sleep(100 * time.Millisecond)
	if envPID := m.findPIDByEnv(instanceID); envPID > 0 {
		actualPID = envPID
	}
	m.writePID(instanceID, actualPID)
	return nil
}

// StopApp 停止应用进程
// 优化：优先使用 processes map，避免不必要的系统查找
func (m *ProcessManager) StopApp(instanceID string) error {
	log.Printf("StopApp: preparing to stop instance %s", instanceID)
	// 查找所有运行中的进程（包括 processes map 和系统中实际运行的）
	var (
		pids   []int
		pidSet = make(map[int]bool)
	)
	addPID := func(pid int, source string) {
		if pid > 0 && !pidSet[pid] {
			log.Printf("StopApp: instance %s found pid %d via %s", instanceID, pid, source)
			pids = append(pids, pid)
			pidSet[pid] = true
		}
	}

	if pid, err := m.readPID(instanceID); err == nil && pid > 0 {
		if err := syscall.Kill(pid, syscall.Signal(0)); err == nil {
			addPID(pid, "pid_file")
		} else {
			log.Printf("StopApp: instance %s pid %d from pid_file not running (%v)", instanceID, pid, err)
			m.removePID(instanceID)
		}
	}

	if cmd, exists := m.processes[instanceID]; exists {
		if cmd != nil && cmd.Process != nil {
			pid := cmd.Process.Pid
			if err := syscall.Kill(pid, syscall.Signal(0)); err == nil {
				addPID(pid, "processes_map")
			} else {
				log.Printf("StopApp: instance %s pid %d from processes_map not running (%v)", instanceID, pid, err)
			}
		}
	}

	for _, pid := range m.findPIDsByEnv(instanceID) {
		if err := syscall.Kill(pid, syscall.Signal(0)); err == nil {
			addPID(pid, "env_scan")
		}
	}

	if len(pids) == 0 {
		log.Printf("StopApp: instance %s no running pid found, nothing to stop", instanceID)
		delete(m.processes, instanceID)
		m.removePID(instanceID)
		return nil
	}

	for _, pid := range pids {
		if err := syscall.Kill(-pid, syscall.SIGTERM); err == nil {
			log.Printf("StopApp: instance %s sending SIGTERM to process group %d", instanceID, -pid)
		} else if err != syscall.ESRCH {
			log.Printf("StopApp: instance %s failed SIGTERM group %d: %v", instanceID, -pid, err)
		}
		if err := syscall.Kill(pid, syscall.SIGTERM); err == nil {
			log.Printf("StopApp: instance %s sending SIGTERM to pid %d", instanceID, pid)
		} else if err != syscall.ESRCH {
			log.Printf("StopApp: instance %s failed SIGTERM pid %d: %v", instanceID, pid, err)
		}
	}

	waitDone := make(chan struct{})
	if cmd, exists := m.processes[instanceID]; exists && cmd != nil && cmd.Process != nil {
		go func() {
			cmd.Wait()
			waitDone <- struct{}{}
		}()
	}

	timeout := 3 * time.Second
	startTime := time.Now()
	initialInterval := 50 * time.Millisecond
	maxInterval := 200 * time.Millisecond

	checkInterval := initialInterval
	lastCheck := time.Now()

	for time.Since(startTime) < timeout {
		select {
		case <-waitDone:
			log.Printf("StopApp: instance %s cmd.Wait() completed", instanceID)
			goto cleanup
		default:
		}

		if time.Since(lastCheck) >= checkInterval {
			allStopped := true
			for _, pid := range pids {
				if err := syscall.Kill(pid, syscall.Signal(0)); err == nil {
					allStopped = false
					break
				}
			}

			if allStopped {
				log.Printf("StopApp: instance %s all pids exited during grace period", instanceID)
				goto cleanup
			}

			if checkInterval < maxInterval {
				checkInterval = time.Duration(float64(checkInterval) * 1.3)
				if checkInterval > maxInterval {
					checkInterval = maxInterval
				}
			}

			lastCheck = time.Now()
		}

		time.Sleep(20 * time.Millisecond)
	}

cleanup:
	remaining := false
	for _, pid := range pids {
		if err := syscall.Kill(pid, syscall.Signal(0)); err == nil {
			remaining = true
			log.Printf("StopApp: instance %s forcing SIGKILL to process group %d", instanceID, -pid)
			syscall.Kill(-pid, syscall.SIGKILL)
			if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
				log.Printf("StopApp: instance %s failed SIGKILL pid %d: %v", instanceID, pid, err)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	if cmd, exists := m.processes[instanceID]; exists {
		if cmd != nil && cmd.Process != nil {
			go cmd.Wait()
		}
	}

	delete(m.processes, instanceID)
	if remaining {
		log.Printf("StopApp: instance %s still has remaining processes after SIGKILL", instanceID)
	}
	m.removePID(instanceID)
	log.Printf("StopApp: instance %s stop sequence complete", instanceID)
	return nil
}

// IsRunning 检查应用是否正在运行
// 优化：优先使用 processes map，只在必要时才进行系统查找，避免每次都执行复杂的 ps 命令
func (m *ProcessManager) IsRunning(instanceID string) bool {
	if pid, err := m.readPID(instanceID); err == nil && pid > 0 {
		if err := syscall.Kill(pid, syscall.Signal(0)); err == nil {
			return true
		}
		m.removePID(instanceID)
	}

	if cmd, exists := m.processes[instanceID]; exists && cmd != nil && cmd.Process != nil {
		pid := cmd.Process.Pid

		statPath := fmt.Sprintf("/proc/%d/stat", pid)
		statData, err := os.ReadFile(statPath)
		if err != nil {
			delete(m.processes, instanceID)
			return false
		}

		idx := strings.LastIndex(string(statData), ")")
		if idx == -1 || idx+2 >= len(statData) {
			if err := syscall.Kill(pid, syscall.Signal(0)); err != nil {
				delete(m.processes, instanceID)
				return false
			}
			m.writePID(instanceID, pid)
			return true
		}

		state := statData[idx+2]
		if state == 'Z' {
			delete(m.processes, instanceID)
			go cmd.Wait()
			return false
		}

		if err := syscall.Kill(pid, syscall.Signal(0)); err != nil {
			delete(m.processes, instanceID)
			return false
		}

		m.writePID(instanceID, pid)
		return true
	}

	instanceDir := filepath.Join(m.config.BaseDir, instanceID)
	if _, err := os.Stat(instanceDir); os.IsNotExist(err) {
		return false
	}

	if pid := m.findPIDByEnv(instanceID); pid > 0 {
		m.writePID(instanceID, pid)
		return true
	}

	return false
}

// GetStatus 获取应用状态
func (m *ProcessManager) GetStatus(instanceID string) (AppStatus, error) {
	cmd, exists := m.processes[instanceID]
	if !exists {
		return AppStatus{
			InstanceID: instanceID,
			Running:    false,
			ExitCode:   -1, // 进程不存在，视为异常
		}, nil
	}

	if cmd.Process == nil {
		return AppStatus{
			InstanceID: instanceID,
			Running:    false,
			ExitCode:   -1, // 进程不存在，视为异常
		}, nil
	}

	// 检查进程是否还活着
	running := m.IsRunning(instanceID)
	pid := 0
	exitCode := -1
	if running {
		pid = cmd.Process.Pid
	} else {
		// 进程已退出，尝试获取退出码
		// 注意：如果进程已经退出，cmd.Wait() 可能已经被调用过
		// 这里我们无法获取退出码，默认设为 -1（异常退出）
		exitCode = -1
	}

	return AppStatus{
		InstanceID: instanceID,
		Running:    running,
		Pid:        pid,
		ExitCode:   exitCode,
	}, nil
}

// ListRunning 列出所有运行中的实例ID
func (m *ProcessManager) ListRunning() []string {
	runningMap := make(map[string]bool)

	for instanceID, cmd := range m.processes {
		if cmd != nil && cmd.Process != nil {
			if m.IsRunning(instanceID) {
				runningMap[instanceID] = true
			}
		}
	}

	entries, err := os.ReadDir(m.config.BaseDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			instanceID := entry.Name()
			if runningMap[instanceID] {
				continue
			}
			if m.IsRunning(instanceID) {
				runningMap[instanceID] = true
			}
		}
	}

	var running []string
	for instanceID := range runningMap {
		running = append(running, instanceID)
	}
	return running
}

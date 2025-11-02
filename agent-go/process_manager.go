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

// StartApp 启动应用进程
func (m *ProcessManager) StartApp(instanceID string, app Assignment, appDir string) error {
	// 检查是否已运行
	if cmd, exists := m.processes[instanceID]; exists {
		if cmd.Process != nil {
			// 检查进程是否还活着
			if err := cmd.Process.Signal(syscall.Signal(0)); err == nil {
				log.Printf("Instance %s already running, PID=%d", instanceID, cmd.Process.Pid)
				return nil
			}
		}
		// 进程已死，清理状态
		delete(m.processes, instanceID)
	}

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
	cmd, exists := m.processes[instanceID]
	if !exists || cmd.Process == nil {
		return nil // 已经停止
	}

	// 发送SIGTERM到进程组
	pgid := -cmd.Process.Pid
	if err := syscall.Kill(pgid, syscall.SIGTERM); err != nil {
		log.Printf("Failed to send SIGTERM to instance %s: %v", instanceID, err)
	} else {
		log.Printf("Sent SIGTERM to instance %s", instanceID)
	}

	// 等待最多5秒
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-done:
		log.Printf("Instance %s stopped gracefully", instanceID)
	case <-time.After(5 * time.Second):
		// 5秒后强制SIGKILL
		syscall.Kill(pgid, syscall.SIGKILL)
		cmd.Wait()
		log.Printf("Force killed instance %s", instanceID)
	}

	delete(m.processes, instanceID)
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

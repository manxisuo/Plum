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
// 优化：优先检查 processes map，使用与 IsRunning 相同的高效查找方法
func (m *ProcessManager) findRunningProcessByInstanceID(instanceID string) int {
	// 方法1：检查 processes map 中的进程（最快）
	if cmd, exists := m.processes[instanceID]; exists {
		if cmd != nil && cmd.Process != nil {
			pid := cmd.Process.Pid
			// 快速验证：使用 Signal(0)，不调用 IsRunning（避免递归和系统查找）
			if err := syscall.Kill(pid, syscall.Signal(0)); err == nil {
				return pid
			}
			// 进程已死，清理记录
			delete(m.processes, instanceID)
		}
	}

	// 方法2：processes map 中没有，使用与 IsRunning 相同的优化方法
	instanceDir := fmt.Sprintf("/tmp/plum-agent/nodeA/%s", instanceID)
	if _, err := os.Stat(instanceDir); os.IsNotExist(err) {
		return 0 // 实例目录不存在，肯定没有运行
	}

	// 使用 grep 直接查找环境变量（比遍历所有进程快）
	psCmd := exec.Command("sh", "-c", fmt.Sprintf("ps -eo pid --no-headers | head -100 | xargs -I {} sh -c 'test -r /proc/{}/environ && grep -z \"PLUM_INSTANCE_ID=%s\" /proc/{}/environ >/dev/null 2>&1 && echo {}'", instanceID))
	output, err := psCmd.Output()
	if err == nil && len(output) > 0 {
		var pid int
		line := strings.TrimSpace(strings.Split(string(output), "\n")[0])
		if _, err := fmt.Sscanf(line, "%d", &pid); err == nil && pid > 0 {
			if err := syscall.Kill(pid, syscall.Signal(0)); err == nil {
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
// 优化：优先使用 processes map，避免不必要的系统查找
func (m *ProcessManager) StopApp(instanceID string) error {
	// 查找所有运行中的进程（包括 processes map 和系统中实际运行的）
	var pids []int

	// 从 processes map 中获取 PID（最快路径）
	if cmd, exists := m.processes[instanceID]; exists {
		if cmd != nil && cmd.Process != nil {
			pid := cmd.Process.Pid
			// 验证进程确实存在
			if err := syscall.Kill(pid, syscall.Signal(0)); err == nil {
				pids = append(pids, pid)
			}
		}
	}

	// 只有在 processes map 中没有找到进程时，才进行系统查找（Agent重启后的情况）
	// 使用与 IsRunning 相同的优化方法：直接 grep 环境变量
	if len(pids) == 0 {
		instanceDir := fmt.Sprintf("/tmp/plum-agent/nodeA/%s", instanceID)
		if _, err := os.Stat(instanceDir); err == nil {
			// 实例目录存在，使用 grep 直接查找环境变量
			psCmd := exec.Command("sh", "-c", fmt.Sprintf("ps -eo pid --no-headers | head -100 | xargs -I {} sh -c 'test -r /proc/{}/environ && grep -z \"PLUM_INSTANCE_ID=%s\" /proc/{}/environ >/dev/null 2>&1 && echo {}'", instanceID))
			output, err := psCmd.Output()
			if err == nil && len(output) > 0 {
				lines := strings.Split(strings.TrimSpace(string(output)), "\n")
				for _, line := range lines {
					var pid int
					if _, err := fmt.Sscanf(strings.TrimSpace(line), "%d", &pid); err == nil && pid > 0 {
						if err := syscall.Kill(pid, syscall.Signal(0)); err == nil {
							pids = append(pids, pid)
							break // 只取第一个匹配的进程
						}
					}
				}
			}
		}
	}

	if len(pids) == 0 {
		delete(m.processes, instanceID)
		return nil // 已经停止
	}

	// 停止所有找到的进程（并行发送信号，更快）
	for _, pid := range pids {
		// 先尝试发送 SIGTERM 到进程本身
		syscall.Kill(pid, syscall.SIGTERM)
		// 同时尝试发送到进程组（如果进程在新进程组中）
		syscall.Kill(-pid, syscall.SIGTERM)
	}

	// 对于 processes map 中的进程，使用 cmd.Wait() 异步等待
	waitDone := make(chan struct{})
	if cmd, exists := m.processes[instanceID]; exists && cmd != nil && cmd.Process != nil {
		go func() {
			cmd.Wait()
			waitDone <- struct{}{}
		}()
	}

	// 等待进程退出（最多3秒，减少等待时间）
	// 使用更短的初始检查间隔，更快检测到进程退出
	timeout := 3 * time.Second
	startTime := time.Now()
	initialInterval := 50 * time.Millisecond // 从50ms开始，更快响应
	maxInterval := 200 * time.Millisecond    // 最大间隔也减少到200ms

	checkInterval := initialInterval
	lastCheck := time.Now()

	for time.Since(startTime) < timeout {
		// 优先检查 cmd.Wait() 是否已完成（最快）
		select {
		case <-waitDone:
			// 进程已退出（通过 cmd.Wait() 检测到）
			goto cleanup
		default:
		}

		// 每 checkInterval 检查一次进程状态
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
				goto cleanup
			}

			// 动态增加检查间隔（减少开销）
			if checkInterval < maxInterval {
				checkInterval = time.Duration(float64(checkInterval) * 1.3) // 每次增加30%
				if checkInterval > maxInterval {
					checkInterval = maxInterval
				}
			}

			lastCheck = time.Now()
		}

		// 更短的休眠时间，更快响应
		time.Sleep(20 * time.Millisecond)
	}

cleanup:
	// 检查是否还有进程在运行，如果有则强制 SIGKILL
	for _, pid := range pids {
		if err := syscall.Kill(pid, syscall.Signal(0)); err == nil {
			// 进程还在运行，强制 SIGKILL（并行发送，更快）
			syscall.Kill(pid, syscall.SIGKILL)
			syscall.Kill(-pid, syscall.SIGKILL)
			// 等待进程退出（减少等待时间）
			time.Sleep(100 * time.Millisecond)
		}
	}

	// 如果有 processes map 中的记录，异步等待
	if cmd, exists := m.processes[instanceID]; exists {
		if cmd != nil && cmd.Process != nil {
			go cmd.Wait() // 异步等待，避免阻塞
		}
	}

	delete(m.processes, instanceID)
	return nil
}

// IsRunning 检查应用是否正在运行
// 优化：优先使用 processes map，只在必要时才进行系统查找，避免每次都执行复杂的 ps 命令
func (m *ProcessManager) IsRunning(instanceID string) bool {
	// 方法1：检查 processes map 中的进程（最快路径）
	if cmd, exists := m.processes[instanceID]; exists && cmd != nil && cmd.Process != nil {
		pid := cmd.Process.Pid

		// 使用 /proc/<pid>/stat 检测进程状态（最可靠，能检测僵尸进程）
		statPath := fmt.Sprintf("/proc/%d/stat", pid)
		statData, err := os.ReadFile(statPath)
		if err != nil {
			// 无法读取 /proc/<pid>/stat，说明进程已死
			delete(m.processes, instanceID)
			return false
		}

		// 解析进程状态：stat格式为 "pid (comm) state ..."
		// 找到第二个 ')' 后的第一个字符就是状态
		statStr := string(statData)
		idx := strings.LastIndex(statStr, ")")
		if idx == -1 || idx+2 >= len(statStr) {
			// 格式异常，回退到 Signal(0)
			if err := syscall.Kill(pid, syscall.Signal(0)); err != nil {
				delete(m.processes, instanceID)
				return false
			}
			return true
		}

		state := statStr[idx+2]
		// Z = 僵尸进程（已死），其他状态认为进程还活着
		if state == 'Z' {
			// 僵尸进程，已经死亡
			delete(m.processes, instanceID)
			// 尝试 Wait() 回收僵尸进程
			go cmd.Wait()
			return false
		}

		// 双重检查，使用 Signal(0) 作为补充验证
		if err := syscall.Kill(pid, syscall.Signal(0)); err != nil {
			// Signal(0) 失败，但 /proc 显示进程存在，可能是权限问题
			// 这种情况认为进程已死（更安全）
			delete(m.processes, instanceID)
			return false
		}

		return true
	}

	// 方法2：processes map 中没有，但可能存在于系统中（Agent重启后）
	// 使用更高效的方法：直接通过环境变量查找，避免遍历所有进程
	// 先检查实例目录是否存在（快速过滤）
	instanceDir := fmt.Sprintf("/tmp/plum-agent/nodeA/%s", instanceID)
	if _, err := os.Stat(instanceDir); os.IsNotExist(err) {
		return false // 实例目录不存在，肯定没有运行
	}

	// 实例目录存在，使用 grep 直接查找环境变量（比遍历所有进程快）
	// 限制只检查前100个进程，避免遍历所有进程（通常 Plum 管理的进程不会很多）
	psCmd := exec.Command("sh", "-c", fmt.Sprintf("ps -eo pid --no-headers | head -100 | xargs -I {} sh -c 'test -r /proc/{}/environ && grep -z \"PLUM_INSTANCE_ID=%s\" /proc/{}/environ >/dev/null 2>&1 && echo {}'", instanceID))
	output, err := psCmd.Output()
	if err == nil && len(output) > 0 {
		var pid int
		line := strings.TrimSpace(strings.Split(string(output), "\n")[0])
		if _, err := fmt.Sscanf(line, "%d", &pid); err == nil && pid > 0 {
			if err := syscall.Kill(pid, syscall.Signal(0)); err == nil {
				return true
			}
		}
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

	// 方法2：查找所有 /tmp/plum-agent/nodeA/ 下的进程
	// 由于 exec 会替换进程，cmdline 可能只是相对路径（如 ./worker-demo）
	// 所以需要通过 CWD、EXE 或 environ 来识别 instanceID

	// 使用更简单可靠的方法：直接遍历所有进程，检查 CWD 或 EXE
	// 避免使用 xargs，因为它可能在某些情况下返回非零退出码
	var allPids []int
	psCmd := exec.Command("sh", "-c", "ps -eo pid --no-headers")
	psOutput, err := psCmd.Output()
	if err == nil && len(psOutput) > 0 {
		lines := strings.Split(strings.TrimSpace(string(psOutput)), "\n")
		for _, line := range lines {
			var pid int
			if _, err := fmt.Sscanf(strings.TrimSpace(line), "%d", &pid); err == nil && pid > 0 {
				// 检查进程的 CWD 或 EXE 是否在 /tmp/plum-agent/nodeA/ 下
				cwdPath := fmt.Sprintf("/proc/%d/cwd", pid)
				exePath := fmt.Sprintf("/proc/%d/exe", pid)
				cwd, _ := os.Readlink(cwdPath)
				exe, _ := os.Readlink(exePath)
				if strings.Contains(cwd, "/tmp/plum-agent/nodeA/") || strings.Contains(exe, "/tmp/plum-agent/nodeA/") {
					allPids = append(allPids, pid)
				}
			}
		}
	}

	// 处理找到的进程
	for _, pid := range allPids {
		// 验证进程确实存在
		if err := syscall.Kill(pid, syscall.Signal(0)); err != nil {
			continue // 进程不存在，跳过
		}

		instanceID := ""

		// 方法1：从进程环境变量读取（最可靠）
		environPath := fmt.Sprintf("/proc/%d/environ", pid)
		if environData, err := os.ReadFile(environPath); err == nil {
			environStr := string(environData)
			for _, envLine := range strings.Split(environStr, "\x00") {
				if strings.HasPrefix(envLine, "PLUM_INSTANCE_ID=") {
					instanceID = strings.TrimPrefix(envLine, "PLUM_INSTANCE_ID=")
					break
				}
			}
		}

		// 方法2：如果环境变量读取失败，从 CWD 提取
		if instanceID == "" {
			if cwd, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid)); err == nil {
				if strings.Contains(cwd, "/tmp/plum-agent/nodeA/") {
					pathParts := strings.Split(cwd, "/tmp/plum-agent/nodeA/")
					if len(pathParts) > 1 {
						remaining := pathParts[1]
						pathParts2 := strings.Split(remaining, "/")
						if len(pathParts2) > 0 {
							instanceID = pathParts2[0]
						}
					}
				}
			}
		}

		// 方法3：如果 CWD 也失败，从 EXE 提取
		if instanceID == "" {
			if exe, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid)); err == nil {
				if strings.Contains(exe, "/tmp/plum-agent/nodeA/") {
					pathParts := strings.Split(exe, "/tmp/plum-agent/nodeA/")
					if len(pathParts) > 1 {
						remaining := pathParts[1]
						pathParts2 := strings.Split(remaining, "/")
						if len(pathParts2) > 0 {
							instanceID = pathParts2[0]
						}
					}
				}
			}
		}

		if instanceID != "" {
			runningMap[instanceID] = true
		}
	}

	// 转换为列表
	var running []string
	for instanceID := range runningMap {
		running = append(running, instanceID)
	}
	return running
}

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

	// 方法2：通过环境变量、CWD 或 EXE 查找系统中实际运行的进程
	// 由于 exec 会替换进程，cmdline 可能只是相对路径（如 ./worker-demo）
	// 所以需要通过 environ、CWD 或 EXE 来识别 instanceID

	// 查找所有 /tmp/plum-agent/nodeA/ 下的进程
	psCmd := exec.Command("sh", "-c", "ps -eo pid | xargs -I {} sh -c 'test -d /proc/{} && (readlink -f /proc/{}/cwd 2>/dev/null | grep -q \"/tmp/plum-agent/nodeA/\" || readlink -f /proc/{}/exe 2>/dev/null | grep -q \"/tmp/plum-agent/nodeA/\") && echo {}'")
	output, err := psCmd.Output()

	// 如果上面的命令失败，回退到简单的 ps 命令
	if err != nil || len(output) == 0 {
		psCmd2 := exec.Command("sh", "-c", "ps -eo pid,cmd | grep '/tmp/plum-agent/nodeA/' | grep -v grep | awk '{print $1}'")
		output2, err2 := psCmd2.Output()
		if err2 == nil && len(output2) > 0 {
			output = output2
		}
	}

	// 即使命令返回错误（如 exit status 123），只要输出不为空，也应该处理
	// 因为 xargs 在某些情况下可能返回非零退出码，但仍然有有效输出
	if len(output) > 0 {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			var pid int
			if _, err := fmt.Sscanf(strings.TrimSpace(line), "%d", &pid); err == nil && pid > 0 {
				// 验证进程确实存在
				if err := syscall.Kill(pid, syscall.Signal(0)); err != nil {
					continue // 进程不存在，跳过
				}

				// 方法2a：从环境变量读取（最可靠）
				environPath := fmt.Sprintf("/proc/%d/environ", pid)
				if environData, err := os.ReadFile(environPath); err == nil {
					environStr := string(environData)
					for _, envLine := range strings.Split(environStr, "\x00") {
						if envLine == fmt.Sprintf("PLUM_INSTANCE_ID=%s", instanceID) {
							log.Printf("Found running process for instance %s (PID=%d, found by environ, not in processes map)", instanceID, pid)
							return pid
						}
					}
				}

				// 方法2b：如果环境变量匹配失败，从 CWD 检查
				if cwd, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid)); err == nil {
					if strings.Contains(cwd, fmt.Sprintf("/tmp/plum-agent/nodeA/%s/", instanceID)) {
						log.Printf("Found running process for instance %s (PID=%d, found by cwd, not in processes map)", instanceID, pid)
						return pid
					}
				}

				// 方法2c：如果 CWD 也失败，从 EXE 检查
				if exe, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid)); err == nil {
					if strings.Contains(exe, fmt.Sprintf("/tmp/plum-agent/nodeA/%s/", instanceID)) {
						log.Printf("Found running process for instance %s (PID=%d, found by exe, not in processes map)", instanceID, pid)
						return pid
					}
				}
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

	// 通过环境变量、CWD 或 EXE 查找系统中实际运行的进程
	// 由于 exec 会替换进程，cmdline 可能只是相对路径（如 ./worker-demo）
	// 所以需要通过 environ、CWD 或 EXE 来识别 instanceID

	// 查找所有 /tmp/plum-agent/nodeA/ 下的进程，然后通过 environ、CWD 或 EXE 匹配 instanceID
	psCmd := exec.Command("sh", "-c", "ps -eo pid | xargs -I {} sh -c 'test -d /proc/{} && (readlink -f /proc/{}/cwd 2>/dev/null | grep -q \"/tmp/plum-agent/nodeA/\" || readlink -f /proc/{}/exe 2>/dev/null | grep -q \"/tmp/plum-agent/nodeA/\") && echo {}'")
	output, err := psCmd.Output()

	// 如果上面的命令失败，回退到简单的 ps 命令
	if err != nil || len(output) == 0 {
		psCmd2 := exec.Command("sh", "-c", "ps -eo pid,cmd | grep '/tmp/plum-agent/nodeA/' | grep -v grep | awk '{print $1}'")
		output2, err2 := psCmd2.Output()
		if err2 == nil && len(output2) > 0 {
			output = output2
		}
	}

	// 即使命令返回错误（如 exit status 123），只要输出不为空，也应该处理
	// 因为 xargs 在某些情况下可能返回非零退出码，但仍然有有效输出
	if len(output) > 0 {
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
				if found {
					continue
				}

				// 验证进程确实存在
				if err := syscall.Kill(pid, syscall.Signal(0)); err != nil {
					continue // 进程不存在，跳过
				}

				// 方法1：从环境变量读取（最可靠）
				matched := false
				environPath := fmt.Sprintf("/proc/%d/environ", pid)
				if environData, err := os.ReadFile(environPath); err == nil {
					environStr := string(environData)
					for _, envLine := range strings.Split(environStr, "\x00") {
						if envLine == fmt.Sprintf("PLUM_INSTANCE_ID=%s", instanceID) {
							matched = true
							break
						}
					}
				}

				// 方法2：如果环境变量匹配失败，从 CWD 检查
				if !matched {
					cwdPath := fmt.Sprintf("/proc/%d/cwd", pid)
					if cwd, err := os.Readlink(cwdPath); err == nil {
						pattern := fmt.Sprintf("/tmp/plum-agent/nodeA/%s/", instanceID)
						if strings.Contains(cwd, pattern) {
							matched = true
						}
					}
				}

				// 方法3：如果 CWD 也失败，从 EXE 检查
				if !matched {
					exePath := fmt.Sprintf("/proc/%d/exe", pid)
					if exe, err := os.Readlink(exePath); err == nil {
						pattern := fmt.Sprintf("/tmp/plum-agent/nodeA/%s/", instanceID)
						if strings.Contains(exe, pattern) {
							matched = true
						}
					}
				}

				if matched {
					pids = append(pids, pid)
				}
			}
		}
	}

	if len(pids) == 0 {
		delete(m.processes, instanceID)
		return nil // 已经停止
	}

	// 停止所有找到的进程
	for _, pid := range pids {
		// 先尝试发送 SIGTERM 到进程本身
		if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
			log.Printf("Failed to send SIGTERM to process %d (instance %s): %v", pid, instanceID, err)
		} else {
			log.Printf("Sent SIGTERM to process %d (instance %s)", pid, instanceID)
		}

		// 同时尝试发送到进程组（如果进程在新进程组中）
		// 注意：如果进程通过 exec 启动，可能不在新进程组中，所以先尝试进程本身
		pgid := -pid
		if err := syscall.Kill(pgid, syscall.SIGTERM); err == nil {
			log.Printf("Also sent SIGTERM to process group %d (instance %s, PID=%d)", pgid, instanceID, pid)
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
			// 先尝试进程本身
			if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
				log.Printf("Failed to send SIGKILL to process %d (instance %s): %v", pid, instanceID, err)
			} else {
				log.Printf("Force killed process %d (instance %s)", pid, instanceID)
			}
			// 同时尝试进程组
			pgid := -pid
			syscall.Kill(pgid, syscall.SIGKILL)
			// 等待进程退出
			time.Sleep(200 * time.Millisecond)
			// 再次检查
			if err := syscall.Kill(pid, syscall.Signal(0)); err == nil {
				log.Printf("Warning: Process %d (instance %s) still running after SIGKILL", pid, instanceID)
			}
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
	// 方法1：检查 processes map 中的进程
	if cmd, exists := m.processes[instanceID]; exists && cmd != nil && cmd.Process != nil {
		pid := cmd.Process.Pid

		// 使用 /proc/<pid>/stat 检测进程状态（最可靠）
		statPath := fmt.Sprintf("/proc/%d/stat", pid)
		statData, err := os.ReadFile(statPath)
		if err != nil {
			// 无法读取 /proc/<pid>/stat，说明进程已死
			delete(m.processes, instanceID)
			log.Printf("Instance %s process died (PID %d): cannot read /proc/%d/stat", instanceID, pid, pid)
			// 继续检查系统中是否还有该进程
		} else {
			// 解析进程状态：stat格式为 "pid (comm) state ..."
			// 找到第二个 ')' 后的第一个字符就是状态
			statStr := string(statData)
			idx := strings.LastIndex(statStr, ")")
			if idx == -1 || idx+2 >= len(statStr) {
				// 格式异常，回退到 Signal(0)
				if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
					delete(m.processes, instanceID)
					log.Printf("Instance %s process died (PID %d): invalid stat format, Signal(0) failed", instanceID, pid)
					// 继续检查系统中是否还有该进程
				} else {
					return true
				}
			} else {
				state := statStr[idx+2]
				// Z = 僵尸进程（已死），其他状态认为进程还活着
				if state == 'Z' {
					// 僵尸进程，已经死亡
					delete(m.processes, instanceID)
					log.Printf("Instance %s process died (PID %d): zombie process", instanceID, pid)
					// 尝试 Wait() 回收僵尸进程
					go cmd.Wait()
					// 继续检查系统中是否还有该进程
				} else {
					// 双重检查，使用 Signal(0) 作为补充验证
					if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
						// Signal(0) 失败，但 /proc 显示进程存在，可能是权限问题
						// 这种情况认为进程已死（更安全）
						delete(m.processes, instanceID)
						log.Printf("Instance %s process died (PID %d): Signal(0) failed (stat shows state=%c)", instanceID, pid, state)
						// 继续检查系统中是否还有该进程
					} else {
						return true
					}
				}
			}
		}
	}

	// 方法2：通过环境变量、CWD 或 EXE 查找系统中实际运行的进程
	// 由于 exec 会替换进程，cmdline 可能只是相对路径（如 ./worker-demo）
	// 所以需要通过 environ、CWD 或 EXE 来识别 instanceID

	// 查找所有 /tmp/plum-agent/nodeA/ 下的进程
	psCmd := exec.Command("sh", "-c", "ps -eo pid | xargs -I {} sh -c 'test -d /proc/{} && (readlink -f /proc/{}/cwd 2>/dev/null | grep -q \"/tmp/plum-agent/nodeA/\" || readlink -f /proc/{}/exe 2>/dev/null | grep -q \"/tmp/plum-agent/nodeA/\") && echo {}'")
	output, err := psCmd.Output()

	// 如果上面的命令失败，回退到简单的 ps 命令
	if err != nil || len(output) == 0 {
		psCmd2 := exec.Command("sh", "-c", "ps -eo pid,cmd | grep '/tmp/plum-agent/nodeA/' | grep -v grep | awk '{print $1}'")
		output2, err2 := psCmd2.Output()
		if err2 == nil && len(output2) > 0 {
			output = output2
		}
	}

	// 即使命令返回错误（如 exit status 123），只要输出不为空，也应该处理
	// 因为 xargs 在某些情况下可能返回非零退出码，但仍然有有效输出
	if len(output) > 0 {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			var pid int
			if _, err := fmt.Sscanf(strings.TrimSpace(line), "%d", &pid); err == nil && pid > 0 {
				// 验证进程确实存在
				if err := syscall.Kill(pid, syscall.Signal(0)); err != nil {
					continue // 进程不存在，跳过
				}

				// 方法2a：从环境变量读取（最可靠）
				environPath := fmt.Sprintf("/proc/%d/environ", pid)
				if environData, err := os.ReadFile(environPath); err == nil {
					environStr := string(environData)
					for _, envLine := range strings.Split(environStr, "\x00") {
						if envLine == fmt.Sprintf("PLUM_INSTANCE_ID=%s", instanceID) {
							log.Printf("Instance %s is running (PID=%d, found by environ, not in processes map)", instanceID, pid)
							return true
						}
					}
				}

				// 方法2b：如果环境变量匹配失败，从 CWD 检查
				if cwd, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid)); err == nil {
					if strings.Contains(cwd, fmt.Sprintf("/tmp/plum-agent/nodeA/%s/", instanceID)) {
						log.Printf("Instance %s is running (PID=%d, found by cwd, not in processes map)", instanceID, pid)
						return true
					}
				}

				// 方法2c：如果 CWD 也失败，从 EXE 检查
				if exe, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid)); err == nil {
					if strings.Contains(exe, fmt.Sprintf("/tmp/plum-agent/nodeA/%s/", instanceID)) {
						log.Printf("Instance %s is running (PID=%d, found by exe, not in processes map)", instanceID, pid)
						return true
					}
				}
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

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getExeDir() string {
	exePath, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exePath)
}

func main() {
	// 加载.env文件（优先级：环境变量 > .env > 默认值）
	// 查找顺序：程序目录 → 程序上级目录 → 当前工作目录
	var envPath string
	for _, path := range []string{
		filepath.Join(getExeDir(), ".env"),    // 程序目录（agent-go/）
		filepath.Join(getExeDir(), "../.env"), // 上级目录（项目根）
		".env",                                // 当前工作目录
	} {
		if _, err := os.Stat(path); err == nil {
			envPath = path
			break
		}
	}

	if envPath != "" {
		if err := godotenv.Load(envPath); err != nil {
			log.Printf("Note: failed to load %s: %v", envPath, err)
		} else {
			log.Printf("Loaded configuration from %s", envPath)
		}
	}

	nodeID := getEnv("AGENT_NODE_ID", "nodeA")
	controller := getEnv("CONTROLLER_BASE", "http://127.0.0.1:8080")
	dataDir := getEnv("AGENT_DATA_DIR", "/tmp/plum-agent")

	log.Printf("Starting Plum Agent")
	log.Printf("  NodeID: %s", nodeID)
	log.Printf("  Controller: %s", controller)
	log.Printf("  DataDir: %s", dataDir)

	httpClient := NewHTTPClient()
	reconciler := NewReconciler(fmt.Sprintf("%s/%s", dataDir, nodeID), httpClient, controller)

	// 信号处理
	stopCh := make(chan bool, 1)
	nudgeCh := make(chan bool, 100)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGTSTP)

	go func() {
		<-sigCh
		log.Println("Received stop signal")
		stopCh <- true
	}()

	// SSE监听
	go func() {
		for {
			select {
			case <-stopCh:
				return
			default:
			}

			url := fmt.Sprintf("%s/v1/stream?nodeId=%s", controller, nodeID)
			resp, err := http.Get(url)
			if err != nil {
				log.Printf("SSE connection failed: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			scanner := bufio.NewScanner(resp.Body)
			for scanner.Scan() {
				select {
				case nudgeCh <- true:
				default:
				}
			}
			resp.Body.Close()

			select {
			case <-stopCh:
				return
			default:
				time.Sleep(1 * time.Second)
			}
		}
	}()

	// 主循环
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		// 发送心跳
		heartbeat := map[string]string{
			"nodeId": nodeID,
			"ip":     "127.0.0.1",
		}
		url := controller + "/v1/nodes/heartbeat"
		if err := httpClient.PostJSON(url, heartbeat); err != nil {
			log.Printf("Heartbeat failed: %v", err)
		}

		// 获取分配
		assignURL := fmt.Sprintf("%s/v1/assignments?nodeId=%s", controller, nodeID)
		data, err := httpClient.Get(assignURL)
		if err != nil {
			log.Printf("Failed to get assignments: %v", err)
		}
		if err == nil && len(data) > 0 {
			var result struct {
				Items []Assignment `json:"items"`
			}
			if err := json.Unmarshal(data, &result); err != nil {
				log.Printf("Failed to parse assignments: %v", err)
			} else {
				assignments := result.Items
				// 同步状态
				reconciler.Sync(assignments)

				// 注册服务
				for _, a := range assignments {
					if a.Desired == "Running" {
						reconciler.RegisterServices(a.InstanceID, nodeID, "127.0.0.1")
						reconciler.HeartbeatServices(a.InstanceID)
					}
				}
			}
		}

		// 等待5秒或被SSE唤醒
		select {
		case <-stopCh:
			log.Println("Stopping agent...")
			reconciler.StopAll()
			return
		case <-nudgeCh:
			// SSE事件触发，立即执行下一轮
		case <-ticker.C:
			// 定时触发
		}
	}
}

package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/manxisuo/plum/controller/internal/failover"
	"github.com/manxisuo/plum/controller/internal/httpapi"
	"github.com/manxisuo/plum/controller/internal/store"
	sqlitestore "github.com/manxisuo/plum/controller/internal/store/sqlite"
	"github.com/manxisuo/plum/controller/internal/tasks"
)

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

	addr := os.Getenv("CONTROLLER_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	// init sqlite store (pure go driver)
	dbPath := os.Getenv("CONTROLLER_DB")
	if dbPath == "" {
		dbPath = "file:controller.db?_pragma=busy_timeout(5000)"
	}
	s, err := sqlitestore.New(dbPath)
	if err != nil {
		log.Fatalf("init db error: %v", err)
	}
	store.SetCurrent(s)

	// 确保数据库连接在程序退出时正确关闭
	defer func() {
		if closer, ok := s.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				log.Printf("Error closing database: %v", err)
			} else {
				log.Println("Database connection closed properly")
			}
		}
	}()

	// Initialize builtin task definitions
	if err := store.InitBuiltinTaskDefs(store.Current); err != nil {
		log.Printf("Warning: failed to init builtin tasks: %v", err)
	}

	mux := http.NewServeMux()
	httpapi.RegisterRoutes(mux)
	// start failover loop
	failover.Start()
	// start tasks scheduler (minimal)
	tasks.Start()
	// start DAG orchestrator
	httpapi.InitDAGOrchestrator(store.Current)

	// static file server for artifacts
	dataDir := os.Getenv("CONTROLLER_DATA_DIR")
	if dataDir == "" {
		dataDir = "."
	}
	fs := http.FileServer(http.Dir(dataDir + "/artifacts"))
	mux.Handle("/artifacts/", http.StripPrefix("/artifacts/", fs))

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动服务器
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		log.Printf("controller listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// 等待信号
	<-sigChan
	log.Println("Received shutdown signal, gracefully shutting down...")

	// 停止DAG编排器
	httpapi.StopDAGOrchestrator()

	// 这里可以添加更多的清理逻辑，比如停止任务调度器等

	log.Println("Controller shutdown complete")
}

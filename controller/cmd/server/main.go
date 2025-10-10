package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/manxisuo/plum/controller/internal/failover"
	"github.com/manxisuo/plum/controller/internal/httpapi"
	"github.com/manxisuo/plum/controller/internal/store"
	sqlitestore "github.com/manxisuo/plum/controller/internal/store/sqlite"
	"github.com/manxisuo/plum/controller/internal/tasks"
)

func main() {
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

	mux := http.NewServeMux()
	httpapi.RegisterRoutes(mux)
	// start failover loop
	failover.Start()
	// start tasks scheduler (minimal)
	tasks.Start()

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

	// 这里可以添加更多的清理逻辑，比如停止任务调度器等

	log.Println("Controller shutdown complete")
}

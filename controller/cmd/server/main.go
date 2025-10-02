package main

import (
	"log"
	"net/http"
	"os"

	"plum/controller/internal/failover"
	"plum/controller/internal/httpapi"
	"plum/controller/internal/store"
	sqlitestore "plum/controller/internal/store/sqlite"
	"plum/controller/internal/tasks"
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

	log.Printf("controller listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

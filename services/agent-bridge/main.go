package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	cfg := GetConfig()
	
	// 启动 WebSocket Server (供 Sandbox 连接) 及 HTTP API
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	http.HandleFunc("/ws/sandbox", HandleSandboxConnection)
	http.HandleFunc("/api/dispatch", HandleDispatch)
	http.HandleFunc("/api/sandboxes", HandleListSandboxes)
	
	log.Printf("🚀 Agent Bridge Server starting on %s", addr)
	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// 连接主 Hub (作为 Client)
	log.Printf("Connecting to Hub at %s...", cfg.Hub.URL)
	HubClient.Connect(cfg.Hub.URL)
}

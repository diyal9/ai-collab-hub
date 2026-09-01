package main

import (
	"flag"
	"log"
	"os"
)

func main() {
	var bridgeURL string
	var sandboxID string

	flag.StringVar(&bridgeURL, "bridge", "ws://localhost:8087/agent-sandbox", "Agent Bridge WebSocket URL")
	flag.StringVar(&sandboxID, "id", "sandbox-node-1", "Unique Sandbox ID")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// 确保日志目录存在
	os.MkdirAll("/tmp/agent-workspace", 0755)

	log.Printf("AgentSandbox starting... ID=%s Bridge=%s", sandboxID, bridgeURL)

	client := NewClient(bridgeURL, sandboxID)
	client.Start()
}

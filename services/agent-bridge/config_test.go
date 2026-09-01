package main

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConfigParse(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgFile, []byte(`
server:
  host: "127.0.0.1"
  port: 8088
hub:
  url: "ws://localhost:8087/ws/agent"
  agent_id: "test-bridge"
`), 0644); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(cfgFile)
	if err != nil {
		t.Fatal(err)
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	if c.Server.Host != "127.0.0.1" || c.Server.Port != 8088 {
		t.Fatalf("server %+v", c.Server)
	}
	if c.Hub.AgentID != "test-bridge" {
		t.Fatalf("hub %+v", c.Hub)
	}
}

package main

import (
	"flag"
	"log"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config 定义 Bridge 的配置结构
type Config struct {
	Server ServerConfig `yaml:"server"`
	Hub    HubConfig    `yaml:"hub"`
	Log    LogConfig    `yaml:"log"`
}

type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

type HubConfig struct {
	URL      string `yaml:"url"`
	AgentID  string `yaml:"agent_id"`
	APIToken string `yaml:"api_token"`
}

type LogConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

var (
	configPath string
	cfg        Config
	configOnce sync.Once
)

func init() {
	flag.StringVar(&configPath, "config", "config.yaml", "Path to config.yaml")
}

func GetConfig() Config {
	configOnce.Do(func() {
		if !flag.Parsed() {
			flag.Parse()
		}
		loadConfig()
	})
	return cfg
}

func loadConfig() {
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Failed to read config file %s: %v", configPath, err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8087
	}
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Hub.AgentID == "" {
		cfg.Hub.AgentID = "default-bridge-01"
	}
}

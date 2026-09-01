package config

import (
	"os"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port       int    `yaml:"port"`
		JWTSecret  string `yaml:"jwt_secret"`
		PublicURL  string `yaml:"public_url"` // 公网地址，用于审批回调
	} `yaml:"server"`
	Database struct {
		Driver string `yaml:"driver"`
		DSN    string `yaml:"dsn"`
	} `yaml:"database"`
	Storage struct {
		UploadDir string `yaml:"upload_dir"`
		MaxSizeMB int    `yaml:"max_size_mb"`
	} `yaml:"storage"`
	Webhook struct {
		Secret string `yaml:"secret"`
	} `yaml:"webhook"`
	Langfuse struct {
		URL       string `yaml:"url"`
		PublicKey string `yaml:"public_key"`
		SecretKey string `yaml:"secret_key"`
	} `yaml:"langfuse"`
}

var Cfg Config

func Load(path string) error {
	f, err := os.ReadFile(path)
	if err != nil { return err }
	return yaml.Unmarshal(f, &Cfg)
}

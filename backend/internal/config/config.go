package config

import (
	"os"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port      int    `yaml:"port"`
		JWTSecret string `yaml:"jwt_secret"`
	} `yaml:"server"`
	Database struct {
		Driver string `yaml:"driver"`
		DSN    string `yaml:"dsn"`
	} `yaml:"database"`
	Storage struct {
		UploadDir   string `yaml:"upload_dir"`
		MaxSizeMB   int    `yaml:"max_size_mb"`
	} `yaml:"storage"`
	Webhook struct {
		Secret string `yaml:"secret"`
	} `yaml:"webhook"`
}

var Cfg Config

func Load(path string) error {
	f, err := os.ReadFile(path)
	if err != nil { return err }
	return yaml.Unmarshal(f, &Cfg)
}

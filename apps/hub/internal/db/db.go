package db

import (
	"log"
	"ai-collab-hub/internal/config"
	"ai-collab-hub/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init() {
	var err error
	dsn := config.Cfg.Database.DSN
	switch config.Cfg.Database.Driver {
	case "mysql":
		DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	case "sqlite3", "sqlite":
		DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	default:
		log.Fatal("Unsupported DB driver")
	}
	if err != nil {
		log.Fatal("DB connection failed:", err)
	}
	log.Println("Database connected:", config.Cfg.Database.Driver)
	DB.AutoMigrate(&model.User{}, &model.Agent{}, &model.Task{}, &model.TaskStep{}, &model.File{})
}

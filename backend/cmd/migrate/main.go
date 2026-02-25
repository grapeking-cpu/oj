package main

import (
	"log"
	"os"

	"github.com/oj/oj-backend/internal/config"
	"github.com/oj/oj-backend/internal/model"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化数据库
	db, err := config.InitDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}

	// 自动迁移
	err = db.AutoMigrate(
		&model.User{},
		&model.Language{},
		&model.Problem{},
		&model.Submission{},
		&model.Contest{},
		&model.ContestParticipant{},
	)
	if err != nil {
		log.Fatalf("Failed to migrate: %v", err)
	}

	log.Println("Migration completed!")

	// 检查是否需要初始化默认数据
	initDefaultData(db)
}

func initDefaultData(db interface{}) {
	// 这里可以添加默认语言等初始化数据
	// 具体实现取决于 db 的类型
	log.Println("Default data initialization skipped (use SQL)")
	os.Exit(0)
}

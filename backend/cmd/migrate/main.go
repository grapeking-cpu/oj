package main

import (
	"log"
	"os"

	"github.com/oj/oj-backend/internal/config"
	"github.com/oj/oj-backend/internal/model"
	"gorm.io/gorm"
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

	// 初始化默认数据
	initDefaultData(db)
}

func initDefaultData(db *gorm.DB) {
	// 检查是否已有语言数据
	var count int64
	db.Model(&model.Language{}).Count(&count)
	if count > 0 {
		log.Println("Default data already exists, skipping seed")
		os.Exit(0)
	}

	// 默认语言
	languages := []model.Language{
		{
			Name:           "GNU C++17",
			Slug:           "cpp17",
			DisplayOrder:   1,
			SourceFilename: "main.cpp",
			CompileCmd:     `["g++", "-std=c++17", "-O2", "-pipe", "-static", "-s", "-o", "main", "main.cpp"]`,
			CompileTimeout: 10,
			RunCmd:         "./main",
			RunTimeout:     2,
			DockerImage:    "oj-compiler-cpp:latest",
			TimeFactor:     1.0,
			MemoryFactor:   1.0,
			OutputLimit:    65536,
			PidsLimit:      64,
			Enabled:        true,
		},
		{
			Name:           "GNU C11",
			Slug:           "c11",
			DisplayOrder:   2,
			SourceFilename: "main.c",
			CompileCmd:     `["gcc", "-std=c11", "-O2", "-pipe", "-static", "-s", "-o", "main", "main.c"]`,
			CompileTimeout: 10,
			RunCmd:         "./main",
			RunTimeout:     2,
			DockerImage:    "oj-compiler-c:latest",
			TimeFactor:     1.0,
			MemoryFactor:   1.0,
			OutputLimit:    65536,
			PidsLimit:      64,
			Enabled:        true,
		},
		{
			Name:           "Python 3",
			Slug:           "python3",
			DisplayOrder:   3,
			SourceFilename: "main.py",
			CompileCmd:     "",
			CompileTimeout: 0,
			RunCmd:         "python3 main.py",
			RunTimeout:     5,
			DockerImage:    "oj-compiler-python:latest",
			TimeFactor:     2.0,
			MemoryFactor:   1.0,
			OutputLimit:    65536,
			PidsLimit:      32,
			Enabled:        true,
		},
		{
			Name:           "Go 1.21",
			Slug:           "go",
			DisplayOrder:   4,
			SourceFilename: "main.go",
			CompileCmd:     `["go", "build", "-o", "main", "main.go"]`,
			CompileTimeout: 10,
			RunCmd:         "./main",
			RunTimeout:     2,
			DockerImage:    "oj-compiler-go:latest",
			TimeFactor:     1.0,
			MemoryFactor:   1.0,
			OutputLimit:    65536,
			PidsLimit:      64,
			Enabled:        true,
		},
		{
			Name:           "Java 17",
			Slug:           "java17",
			DisplayOrder:   5,
			SourceFilename: "Main.java",
			CompileCmd:     `["javac", "-encoding", "UTF-8", "Main.java"]`,
			CompileTimeout: 10,
			RunCmd:         "java Main",
			RunTimeout:     3,
			DockerImage:    "oj-compiler-java:latest",
			TimeFactor:     1.5,
			MemoryFactor:   1.5,
			OutputLimit:    65536,
			PidsLimit:      64,
			Enabled:        true,
		},
		{
			Name:           "Rust 1.75",
			Slug:           "rust",
			DisplayOrder:   6,
			SourceFilename: "main.rs",
			CompileCmd:     `["rustc", "-O", "-o", "main", "main.rs"]`,
			CompileTimeout: 15,
			RunCmd:         "./main",
			RunTimeout:     2,
			DockerImage:    "oj-compiler-rust:latest",
			TimeFactor:     1.0,
			MemoryFactor:   1.0,
			OutputLimit:    65536,
			PidsLimit:      64,
			Enabled:        true,
		},
	}

	for _, lang := range languages {
		if err := db.Create(&lang).Error; err != nil {
			log.Printf("Failed to create language %s: %v", lang.Name, err)
		}
	}

	log.Printf("Created %d default languages", len(languages))

	// 创建默认管理员账户
	admin := model.User{
		Username:     "admin",
		Email:        "admin@oj.local",
		PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy", // password: admin123
		Nickname:     "Administrator",
		Role:         "admin",
		Status:       "active",
		Rating:       3000,
	}
	if err := db.Create(&admin).Error; err != nil {
		log.Printf("Failed to create admin user: %v", err)
	} else {
		log.Println("Created admin user (username: admin, password: admin123)")
	}

	log.Println("Default data initialization completed!")
}

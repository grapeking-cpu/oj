package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config 应用配置
type Config struct {
	// Server
	Port string

	// Database
	DatabaseURL string

	// Redis
	RedisURL string

	// NATS
	NATSURL string

	// MinIO
	MinIOEndpoint    string
	MinIOAccessKey   string
	MinIOSecretKey   string
	MinIOBucket      string

	// JWT
	JWTSecret string

	// Judge
	JudgeTimeout     int
	JudgeMaxMemory   int64
}

// Load 加载配置
func Load() *Config {
	return &Config{
		Port:            getEnv("PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://oj:oj_password@localhost:5432/oj?sslmode=disable"),
		RedisURL:        getEnv("REDIS_URL", "redis://localhost:6379"),
		NATSURL:         getEnv("NATS_URL", "nats://localhost:4222"),
		MinIOEndpoint:   getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey:  getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey:  getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinIOBucket:     getEnv("MINIO_BUCKET", "oj"),
		JWTSecret:       getEnv("JWT_SECRET", "your-jwt-secret-change-in-production"),
		JudgeTimeout:    getEnvInt("JUDGE_TIMEOUT", 30),
		JudgeMaxMemory:  getEnvInt64("JUDGE_MAX_MEMORY", 512),
	}
}

// InitDB 初始化数据库
func InitDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// 连接池配置
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("Database connected")
	return db, nil
}

// InitRedis 初始化 Redis
func InitRedis(url string) *redis.Client {
	opt, err := redis.ParseURL(url)
	if err != nil {
		log.Fatalf("Failed to parse redis url: %v", err)
	}

	rdb := redis.NewClient(opt)
	if err := rdb.Ping().Err(); err != nil {
		log.Fatalf("Failed to connect redis: %v", err)
	}

	log.Println("Redis connected")
	return rdb
}

// InitNATS 初始化 NATS
func InitNATS(url string) (*nats.Conn, jetstream.JetStream) {
	nc, err := nats.Connect(url)
	if err != nil {
		log.Fatalf("Failed to connect nats: %v", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatalf("Failed to create jetstream: %v", err)
	}

	log.Println("NATS connected")
	return nc, js
}

// InitMinIO 初始化 MinIO
func InitMinIO(endpoint, accessKey, secretKey string) *minio.Client {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		log.Fatalf("Failed to connect minio: %v", err)
	}

	log.Println("MinIO connected")
	return client
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var v int
		if _, err := fmt.Sscanf(value, "%d", &v); err == nil {
			return v
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		var v int64
		if _, err := fmt.Sscanf(value, "%d", &v); err == nil {
			return v
		}
	}
	return defaultValue
}

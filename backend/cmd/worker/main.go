package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	// "time"

	// "github.com/nats-io/nats.go"
	"github.com/oj/oj-backend/internal/config"
	"github.com/oj/oj-backend/internal/queue"
	"github.com/oj/oj-backend/internal/repository"
	"github.com/oj/oj-backend/internal/service/judge"
)

func main() {
	// 加载配置
	workerID := getEnv("WORKER_ID", "judge-worker-1")
	consumerName := getEnv("CONSUMER", "judge.tasks.light")
	concurrency := getEnvInt("CONCURRENCY", 2)

	log.Printf("Starting judge worker: %s, consumer: %s, concurrency: %d",
		workerID, consumerName, concurrency)

	// 初始化数据库
	db, err := config.InitDB(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}

	// 初始化 NATS
	nc, js := config.InitNATS(os.Getenv("NATS_URL"))

	// 初始化 MinIO
	minioClient := config.InitMinIO(
		os.Getenv("MINIO_ENDPOINT"),
		os.Getenv("MINIO_ACCESS_KEY"),
		os.Getenv("MINIO_SECRET_KEY"),
	)

	// 初始化 Repository
	repos := &repository.Repositories{
		Submit: repository.NewSubmitRepo(db),
	}

	// 初始化 Judge Service
	judgeService := judge.NewJudgeService(minioClient, os.Getenv("MINIO_BUCKET"))

	// 创建消费者
	consumer := queue.NewConsumer(js, consumerName, workerID)

	// 创建 Worker Pool
	pool := judge.NewWorkerPool(concurrency, judgeService, repos.Submit, js)

	// 启动 Worker Pool
	pool.Start()

	// 启动消费
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = consumer.Consume(ctx, func(task *queue.JudgeTask) error {
		pool.Submit(task)
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to consume: %v", err)
	}

	// 优雅退出
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("Shutting down...")
	pool.Stop()
	nc.Drain()
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

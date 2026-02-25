package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// JudgeTask 评测任务消息
type JudgeTask struct {
	SubmitID       string    `json:"submit_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	Problem        Problem   `json:"problem"`
	Language       Language  `json:"language"`
	Code           string    `json:"code"`
	Contest        *Contest  `json:"contest"`
	User           User      `json:"user"`
	RetryCount     int       `json:"retry_count"`
	CreatedAt      time.Time `json:"created_at"`
}

// Problem 题目信息
type Problem struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	TimeLimit    int    `json:"time_limit"`
	MemoryLimit  int    `json:"memory_limit"`
	StackLimit   int    `json:"stack_limit"`
	IsSPJ        bool   `json:"is_spj"`
	TestDataZip  string `json:"test_data_zip"`
	TestDataHash string `json:"test_data_hash"`
}

// Language 语言信息
type Language struct {
	ID             int64   `json:"id"`
	Slug           string  `json:"slug"`
	SourceFilename string  `json:"source_filename"`
	CompileCmd     string  `json:"compile_cmd"`
	CompileTimeout int     `json:"compile_timeout"`
	RunCmd         string  `json:"run_cmd"`
	RunTimeout     int     `json:"run_timeout"`
	DockerImage    string  `json:"docker_image"`
	TimeFactor     float64 `json:"time_factor"`
	MemoryFactor   float64 `json:"memory_factor"`
	OutputLimit    int     `json:"output_limit"`
	PidsLimit      int     `json:"pids_limit"`
}

// User 用户信息
type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

// Contest 比赛信息
type Contest struct {
	ID             int64  `json:"id"`
	Type           string `json:"type"`
	PenaltyMinutes int    `json:"penalty_minutes"`
	FrozenMinutes  int    `json:"frozen_minutes"`
	IsVirtual      bool   `json:"is_virtual"`
}

// JudgeResult 评测结果
type JudgeResult struct {
	Status       string     `json:"status"` // PENDING/RUNNING/FINISHED/SYSTEM_ERROR/DLQ
	Score        int        `json:"score"`
	AcceptedTest int        `json:"accepted_test"`
	TotalTest    int        `json:"total_test"`
	TimeMs       int        `json:"time_ms"`
	MemoryKB     int        `json:"memory_kb"`
	Cases        []TestCase `json:"cases"`
	Error        string     `json:"error"`
	RetryCount   int        `json:"retry_count"`
	WorkerID     string     `json:"worker_id,omitempty"`
	StartTime    *time.Time `json:"start_time,omitempty"`
	FinishTime   *time.Time `json:"finish_time,omitempty"`
}

// TestCase 单个测试点结果
type TestCase struct {
	ID         int    `json:"id"`
	Status     string `json:"status"` // AC/WA/TLE/MLE/RE/CE
	TimeMs     int    `json:"time_ms"`
	MemoryKB   int    `json:"memory_kb"`
	Score      int    `json:"score"`
	InputFile  string `json:"input_file,omitempty"`
	OutputFile string `json:"output_file,omitempty"`
}

// Client NATS 客户端
type Client struct {
	nc *nats.Conn
	js jetstream.JetStream
}

// NewClient 创建 NATS 客户端
func NewClient(natsURL string) (*Client, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to nats: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("failed to create jetstream: %w", err)
	}

	return &Client{nc: nc, js: js}, nil
}

// Publish 发布任务
func (c *Client) Publish(subject string, task *JudgeTask) error {
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}

	_, err = c.js.Publish(context.Background(), subject, data)
	return err
}

// Consume 消费任务
func (c *Client) Consume(ctx context.Context, stream, consumer string, handler func(*JudgeTask) error) error {
	cons, err := c.js.Consumer(ctx, stream, consumer)
	if err != nil {
		return err
	}

	_, err = cons.Consume(func(msg jetstream.Msg) {
		var task JudgeTask
		if err := json.Unmarshal(msg.Data(), &task); err != nil {
			log.Printf("Failed to unmarshal task: %v", err)
			msg.Nak()
			return
		}

		if err := handler(&task); err != nil {
			log.Printf("Failed to process task: %v", err)
			msg.NakWithDelay(time.Minute) // 延迟重试
			return
		}

		msg.Ack()
	})
	return err
}

// Close 关闭连接
func (c *Client) Close() {
	if c.nc != nil {
		c.nc.Close()
	}
}

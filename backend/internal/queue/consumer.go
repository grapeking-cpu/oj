package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// Consumer NATS 消费者
type Consumer struct {
	js           jetstream.JetStream
	consumerName string
	workerID     string
	stream       string
}

// NewConsumer 创建消费者
func NewConsumer(js jetstream.JetStream, consumerName, workerID string) *Consumer {
	return &Consumer{
		js:           js,
		consumerName: consumerName,
		workerID:     workerID,
		stream:       "OJ",
	}
}

// Consume 消费任务
func (c *Consumer) Consume(ctx context.Context, handler func(*JudgeTask) error) error {
	// 确保 stream 存在
	if err := c.ensureStream(ctx); err != nil {
		return fmt.Errorf("failed to ensure stream: %w", err)
	}

	// 确保 consumer 存在
	if err := c.ensureConsumer(ctx); err != nil {
		return fmt.Errorf("failed to ensure consumer: %w", err)
	}

	cons, err := c.js.Consumer(ctx, c.stream, c.consumerName)
	if err != nil {
		return fmt.Errorf("failed to get consumer: %w", err)
	}

	log.Printf("Consumer %s started, listening on subject %s", c.consumerName, c.consumerName+"*")

	_, err = cons.Consume(func(msg jetstream.Msg) {
		var task JudgeTask
		if err := json.Unmarshal(msg.Data(), &task); err != nil {
			log.Printf("Failed to unmarshal task: %v", err)
			msg.Nak()
			return
		}

		log.Printf("Received task: %s", task.SubmitID)

		if err := handler(&task); err != nil {
			log.Printf("Failed to process task %s: %v", task.SubmitID, err)
			msg.NakWithDelay(time.Minute)
			return
		}

		msg.Ack()
	})

	return err
}

func (c *Consumer) ensureStream(ctx context.Context) error {
	_, err := c.js.Stream(ctx, c.stream)
	if err == nil {
		return nil
	}

	// Stream 不存在，创建它
	_, err = c.js.CreateStream(ctx, jetstream.StreamConfig{
		Name:      c.stream,
		Subjects:  []string{"judge.tasks.*"},
		Retention: jetstream.WorkQueuePolicy,
		MaxBytes:  100 * 1024 * 1024, // 100MB
		MaxAge:    24 * time.Hour,
		Storage:   jetstream.MemoryStorage,
	})
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}

	log.Printf("Created stream: %s", c.stream)
	return nil
}

func (c *Consumer) ensureConsumer(ctx context.Context) error {
	// 检查 consumer 是否存在
	cons, err := c.js.Consumer(ctx, c.stream, c.consumerName)
	if err == nil {
		// Consumer 存在，获取信息确认
		_, err = cons.Info(ctx)
		if err == nil {
			return nil
		}
	}

	// Consumer 不存在，创建它
	// 将 consumer name 中的下划线转换为点，生成 filter subject
	filterSubject := "judge.tasks." + strings.TrimPrefix(c.consumerName, "judge_tasks_")
	_, err = c.js.CreateConsumer(ctx, c.stream, jetstream.ConsumerConfig{
		Name:          c.consumerName,
		Durable:       c.consumerName,
		FilterSubject: filterSubject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       30 * time.Minute,
		MaxDeliver:    4,
		BackOff:       []time.Duration{1 * time.Second, 5 * time.Second, 30 * time.Second, 2 * time.Minute},
	})
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	log.Printf("Created consumer: %s", c.consumerName)
	return nil
}

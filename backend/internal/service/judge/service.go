package judge

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/oj/oj-backend/internal/model"
	"github.com/oj/oj-backend/internal/queue"
)

// WorkerPool 评测工作池
type WorkerPool struct {
	concurrency int
	queue       chan *queue.JudgeTask
	wg          sync.WaitGroup
	service     *JudgeService

	submitRepo interface {
		UpdateStatus(id string, status string, workerID string, startTime time.Time) error
		UpdateResult(id string, result *queue.JudgeResult) error
		GetSubmitByID(id string) (*model.Submission, error)
	}

	// ✅ 修复：签名必须和 jetstream.JetStream.Publish 完全一致
	js interface {
		Publish(ctx context.Context, subject string, data []byte, opts ...jetstream.PublishOpt) (*jetstream.PubAck, error)
	}
}

// NewWorkerPool 创建工作池
func NewWorkerPool(
	concurrency int,
	service *JudgeService,
	submitRepo interface {
		UpdateStatus(id string, status string, workerID string, startTime time.Time) error
		UpdateResult(id string, result *queue.JudgeResult) error
		GetSubmitByID(id string) (*model.Submission, error)
	},
	js interface {
		Publish(ctx context.Context, subject string, data []byte, opts ...jetstream.PublishOpt) (*jetstream.PubAck, error)
	},
) *WorkerPool {
	return &WorkerPool{
		concurrency: concurrency,
		queue:       make(chan *queue.JudgeTask, concurrency*2),
		service:     service,
		submitRepo:  submitRepo,
		js:          js,
	}
}

// Start 启动工作池
func (p *WorkerPool) Start() {
	for i := 0; i < p.concurrency; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	log.Printf("Worker pool started with %d workers", p.concurrency)
}

// Stop 停止工作池
func (p *WorkerPool) Stop() {
	close(p.queue)
	p.wg.Wait()
	log.Println("Worker pool stopped")
}

// Submit 提交任务
func (p *WorkerPool) Submit(task *queue.JudgeTask) {
	p.queue <- task
}

func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()
	log.Printf("Worker %d started", id)

	for task := range p.queue {
		p.processTask(task)
	}

	log.Printf("Worker %d stopped", id)
}

func (p *WorkerPool) processTask(task *queue.JudgeTask) {
	log.Printf("Processing task %s", task.SubmitID)

	// 更新状态为 RUNNING
	now := time.Now()
	if err := p.submitRepo.UpdateStatus(task.SubmitID, "RUNNING", "", now); err != nil {
		log.Printf("Failed to update status to RUNNING: %v", err)
		return
	}

	// 处理任务
	result, err := p.service.ProcessTask(task)
	if err != nil {
		log.Printf("Task %s failed: %v", task.SubmitID, err)

		// 重试逻辑
		if task.RetryCount < 3 {
			task.RetryCount++

			// 延迟重试
			time.Sleep(time.Duration(task.RetryCount) * 5 * time.Second)

			// 重新发布到队列
			data, mErr := json.Marshal(task)
			if mErr != nil {
				log.Printf("Failed to marshal retry task %s: %v", task.SubmitID, mErr)
				return
			}

			subject := "judge.tasks.light"
			if task.Language.Slug == "cpp17" || task.Language.Slug == "java17" {
				subject = "judge.tasks.heavy"
			}

			if _, pubErr := p.js.Publish(context.Background(), subject, data); pubErr != nil {
				log.Printf("Failed to republish task %s to %s: %v", task.SubmitID, subject, pubErr)
				return
			}
		} else {
			// 进入 DLQ（这里你现在只是标记结果为 DLQ，后续建议再真正 publish 到 judge.dlq）
			result.Status = "DLQ"
			result.Error = err.Error()
			_ = p.submitRepo.UpdateResult(task.SubmitID, result)
		}
		return
	}

	// 更新结果
	if err := p.submitRepo.UpdateResult(task.SubmitID, result); err != nil {
		log.Printf("Failed to update result: %v", err)
	}

	log.Printf("Task %s completed with status %s", task.SubmitID, result.Status)
}

// JudgeService 评测服务
type JudgeService struct {
	minioClient *minio.Client
	bucketName  string
}

// NewJudgeService 创建评测服务
func NewJudgeService(minioClient *minio.Client, bucketName string) *JudgeService {
	return &JudgeService{
		minioClient: minioClient,
		bucketName:  bucketName,
	}
}

// ProcessTask 处理评测任务
func (s *JudgeService) ProcessTask(task *queue.JudgeTask) (*queue.JudgeResult, error) {
	// 1. 创建工作目录
	workspace, err := s.createWorkspace(task)
	if err != nil {
		return &queue.JudgeResult{Status: "SYSTEM_ERROR", Error: err.Error()}, err
	}
	defer s.cleanup(workspace)

	// 2. 编译
	compileResult, err := s.compile(task, workspace)
	if err != nil {
		return &queue.JudgeResult{
			Status:       "FINISHED",
			Score:        0,
			AcceptedTest: 0,
			TotalTest:    len(task.Problem.TestDataZip),
			Error:        compileResult.Error,
		}, nil
	}

	// 3. 运行测试
	results := s.runTestCases(task, workspace)

	// 4. 聚合结果
	return s.aggregateResults(results), nil
}

func (s *JudgeService) createWorkspace(task *queue.JudgeTask) (string, error) {
	// 实际实现中创建临时目录
	return fmt.Sprintf("/tmp/oj-%s", task.SubmitID), nil
}

func (s *JudgeService) cleanup(workspace string) {
	// 清理临时文件
}

func (s *JudgeService) compile(task *queue.JudgeTask, workspace string) (*queue.JudgeResult, error) {
	// 检查是否需要编译
	if task.Language.CompileCmd == "" {
		return &queue.JudgeResult{}, nil
	}

	// TODO: 实现编译逻辑
	return &queue.JudgeResult{}, nil
}

func (s *JudgeService) runTestCases(task *queue.JudgeTask, workspace string) []queue.TestCase {
	// TODO: 实现测试运行
	// 1. 拉取测试数据
	// 2. 运行 Docker 容器
	// 3. 比对输出
	// 4. 收集结果
	return []queue.TestCase{
		{ID: 1, Status: "AC", TimeMs: 10, MemoryKB: 1024, Score: 100},
	}
}

func (s *JudgeService) aggregateResults(cases []queue.TestCase) *queue.JudgeResult {
	accepted := 0
	totalTime := 0
	maxMemory := 0
	totalScore := 0

	for _, c := range cases {
		if c.Status == "AC" {
			accepted++
		}
		totalTime += c.TimeMs
		if c.MemoryKB > maxMemory {
			maxMemory = c.MemoryKB
		}
		totalScore += c.Score
	}

	return &queue.JudgeResult{
		Status:       "FINISHED",
		Score:        totalScore,
		AcceptedTest: accepted,
		TotalTest:    len(cases),
		TimeMs:       totalTime,
		MemoryKB:     maxMemory,
		Cases:        cases,
		FinishTime:   timePtr(time.Now()),
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}

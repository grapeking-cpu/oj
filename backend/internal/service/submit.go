package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/oj/oj-backend/internal/model"
	"github.com/oj/oj-backend/internal/queue"
	"github.com/oj/oj-backend/internal/repository"
)

type SubmitService struct {
	repo       *repository.SubmitRepo
	langRepo   *repository.LanguageRepo
	js         jetstream.JetStream
	minio      *minio.Client
	codeBucket string
}

func NewSubmitService(repo *repository.SubmitRepo, langRepo *repository.LanguageRepo, js jetstream.JetStream, minioClient *minio.Client, codeBucket string) *SubmitService {
	return &SubmitService{
		repo:       repo,
		langRepo:   langRepo,
		js:         js,
		minio:      minioClient,
		codeBucket: codeBucket,
	}
}

type SubmitParams struct {
	ProblemID      int64  `json:"problem_id" binding:"required"`
	LanguageID     int64  `json:"language_id" binding:"required"`
	Code           string `json:"code" binding:"required"`
	ContestID      *int64 `json:"contest_id"`
	IdempotencyKey string `json:"idempotency_key"`
}

func (s *SubmitService) Create(userID int64, params SubmitParams) (*model.Submission, error) {
	// 幂等检查
	if params.IdempotencyKey != "" {
		exists, err := s.repo.CheckIdempotency(params.IdempotencyKey)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, fmt.Errorf("duplicate submission")
		}
	}

	submitID := uuid.New().String()

	submission := &model.Submission{
		SubmitID:    submitID,
		UserID:      userID,
		ProblemID:   params.ProblemID,
		LanguageID:  params.LanguageID,
		ContestID:   params.ContestID,
		Code:        params.Code, // 直接存DB
		CodeLength:  len(params.Code),
		JudgeResult: `{"status":"PENDING"}`,
		IsContest:   params.ContestID != nil,
		CreatedAt:   time.Now(),
	}

	if params.IdempotencyKey != "" {
		submission.IdempotencyKey = params.IdempotencyKey
	}

	// 创建提交记录
	if err := s.repo.Create(submission); err != nil {
		return nil, err
	}

	// 增加题目提交数
	if s.langRepo != nil {
		// 获取语言用于计数
	}

	// 发布评测任务
	if err := s.publishJudgeTask(submission); err != nil {
		return nil, err
	}

	return submission, nil
}

func (s *SubmitService) GetBySubmitID(submitID string) (*model.Submission, error) {
	return s.repo.GetBySubmitID(submitID)
}

func (s *SubmitService) ListByUser(userID int64, problemID *int64, status *string, page, pageSize int) ([]model.Submission, int64, error) {
	return s.repo.ListByUser(userID, problemID, status, page, pageSize)
}

// CreateContest 创建比赛提交
func (s *SubmitService) CreateContest(userID, contestID int64, params SubmitParams) (*model.Submission, error) {
	params.ContestID = &contestID
	return s.Create(userID, params)
}

// saveCode 保存代码到 MinIO (可选功能)
func (s *SubmitService) saveCode(submitID, code string) error {
	if s.minio == nil {
		return nil // 未配置 MinIO 时跳过
	}
	_, err := s.minio.PutObject(
		context.Background(),
		s.codeBucket,
		fmt.Sprintf("submissions/%s/code", submitID),
		&stringReader{s: code},
		int64(len(code)),
		minio.PutObjectOptions{ContentType: "text/plain"},
	)
	return err
}

func (s *SubmitService) publishJudgeTask(submission *model.Submission) error {
	// 获取语言信息用于分流
	lang, err := s.langRepo.GetByID(submission.LanguageID)
	if err != nil {
		return fmt.Errorf("failed to get language: %w", err)
	}

	// 构建轻量级任务消息 (只带 submit_id)
	task := &queue.JudgeTask{
		SubmitID:       submission.SubmitID,
		IdempotencyKey: submission.IdempotencyKey,
		Language: queue.Language{
			ID:   lang.ID,
			Slug: lang.Slug,
		},
		RetryCount: 0,
		CreatedAt:  time.Now(),
	}

	data, err := json.Marshal(task)
	if err != nil {
		return err
	}

	// 根据语言分流
	subject := getTaskSubject(lang.Slug)

	_, err = s.js.Publish(context.Background(), subject, data)
	return err
}

// getTaskSubject 根据语言 slug 返回对应的 NATS subject
func getTaskSubject(langSlug string) string {
	heavyLangs := map[string]bool{
		"cpp17":  true,
		"c11":    true,
		"java17": true,
	}
	if heavyLangs[langSlug] {
		return "judge.tasks.heavy"
	}
	return "judge.tasks.light"
}

// stringReader 实现 io.Reader
type stringReader struct {
	s   string
	pos int
}

func (r *stringReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.s) {
		return 0, io.EOF
	}
	n = copy(p, r.s[r.pos:])
	r.pos += n
	return n, nil
}

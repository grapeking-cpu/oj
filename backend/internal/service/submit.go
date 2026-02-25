package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/oj/oj-backend/internal/model"
	"github.com/oj/oj-backend/internal/queue"
	"github.com/oj/oj-backend/internal/repository"
)

type SubmitService struct {
	repo      *repository.SubmitRepo
	js        jetstream.JetStream
	minio     *minio.Client
	bucket    string
	problemRepo *repository.ProblemRepo
}

func NewSubmitService(repo *repository.SubmitRepo, js jetstream.JetStream, minioClient *minio.Client) *SubmitService {
	return &SubmitService{
		repo:   repo,
		js:     js,
		minio:  minioClient,
		bucket: "oj-code",
	}
}

type SubmitParams struct {
	ProblemID    int64  `json:"problem_id" binding:"required"`
	LanguageID   int64  `json:"language_id" binding:"required"`
	Code         string `json:"code" binding:"required"`
	ContestID    *int64 `json:"contest_id"`
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
		SubmitID:     submitID,
		UserID:       userID,
		ProblemID:    params.ProblemID,
		LanguageID:   params.LanguageID,
		ContestID:   params.ContestID,
		Code:        params.Code,
		CodeLength:  len(params.Code),
		JudgeResult: `{"status":"PENDING"}`,
		IsContest:   params.ContestID != nil,
		CreatedAt:   time.Now(),
	}

	if params.IdempotencyKey != "" {
		submission.IdempotencyKey = params.IdempotencyKey
	}

	// 保存代码到 MinIO
	if err := s.saveCode(submission.SubmitID, params.Code); err != nil {
		return nil, err
	}

	// 创建提交记录
	if err := s.repo.Create(submission); err != nil {
		return nil, err
	}

	// 增加题目提交数
	if s.problemRepo != nil {
		s.problemRepo.IncrementSubmitCount(params.ProblemID)
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

func (s *SubmitService) saveCode(submitID, code string) error {
	_, err := s.minio.PutObject(
		context.Background(),
		s.bucket,
		fmt.Sprintf("submissions/%s/code", submitID),
		stringToReader(code),
		int64(len(code)),
		minio.PutObjectOptions{ContentType: "text/plain"},
	)
	return err
}

func (s *SubmitService) publishJudgeTask(submission *model.Submission) error {
	// 构建任务消息
	task := &queue.JudgeTask{
		SubmitID: submission.SubmitID,
		IdempotencyKey: submission.IdempotencyKey,
		Code: submission.Code,
		RetryCount: 0,
		CreatedAt: time.Now(),
	}

	data, err := json.Marshal(task)
	if err != nil {
		return err
	}

	// 根据语言选择队列
	subject := "judge.tasks.light"

	_, err = s.js.Publish(context.Background(), subject, data)
	return err
}

func stringToReader(s string) *stringReader {
	return &stringReader{s: s}
}

type stringReader struct {
	s   string
	pos int
}

func (r *stringReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.s) {
		return 0, nil
	}
	n = copy(p, r.s[r.pos:])
	r.pos += n
	return n, nil
}

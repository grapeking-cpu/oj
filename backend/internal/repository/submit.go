package repository

import (
	"encoding/json"
	"time"

	"github.com/oj/oj-backend/internal/model"
	"github.com/oj/oj-backend/internal/queue"
	"gorm.io/gorm"
)

type SubmitRepo struct {
	db *gorm.DB
}

func NewSubmitRepo(db *gorm.DB) *SubmitRepo {
	return &SubmitRepo{db: db}
}

func (r *SubmitRepo) Create(submission *model.Submission) error {
	return r.db.Create(submission).Error
}

func (r *SubmitRepo) GetBySubmitID(submitID string) (*model.Submission, error) {
	var submission model.Submission
	err := r.db.Where("submit_id = ?", submitID).First(&submission).Error
	if err != nil {
		return nil, err
	}
	return &submission, nil
}

func (r *SubmitRepo) GetByID(id int64) (*model.Submission, error) {
	var submission model.Submission
	err := r.db.First(&submission, id).Error
	if err != nil {
		return nil, err
	}
	return &submission, nil
}

func (r *SubmitRepo) Update(submission *model.Submission) error {
	return r.db.Save(submission).Error
}

func (r *SubmitRepo) UpdateStatus(submitID, status, workerID string, startTime time.Time) error {
	updates := map[string]interface{}{
		"judge_result": map[string]interface{}{
			"status":    status,
			"worker_id": workerID,
		},
	}

	if status == "RUNNING" {
		updates["worker_id"] = workerID
		updates["start_time"] = startTime
	}

	return r.db.Model(&model.Submission{}).
		Where("submit_id = ? AND (judge_result->>'status' IN ('PENDING', 'RUNNING') OR judge_result->>'status' IS NULL)", submitID).
		Updates(updates).Error
}

func (r *SubmitRepo) UpdateResult(submitID string, result *queue.JudgeResult) error {
	resultJSON, _ := json.Marshal(result)

	updates := map[string]interface{}{
		"judge_result": string(resultJSON),
		"finish_time":  time.Now(),
	}

	// 只有在成功时才更新排名
	if result.Status == "FINISHED" && result.Score > 0 {
		// Contest rank update handled elsewhere
	}

	return r.db.Model(&model.Submission{}).
		Where("submit_id = ?", submitID).
		Updates(updates).Error
}

func (r *SubmitRepo) ListByUser(userID int64, problemID *int64, status *string, page, pageSize int) ([]model.Submission, int64, error) {
	var submissions []model.Submission
	var total int64

	query := r.db.Model(&model.Submission{}).Where("user_id = ?", userID)

	if problemID != nil {
		query = query.Where("problem_id = ?", *problemID)
	}
	if status != nil && *status != "" {
		query = query.Where("judge_result->>'status' = ?", *status)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&submissions).Error

	return submissions, total, err
}

func (r *SubmitRepo) CheckIdempotency(key string) (bool, error) {
	var count int64
	err := r.db.Model(&model.Submission{}).Where("idempotency_key = ?", key).Count(&count).Error
	return count > 0, err
}

// GetSubmitByID fetches a submission by its ID
func (r *SubmitRepo) GetSubmitByID(id string) (*model.Submission, error) {
    var sub model.Submission
    if err := r.db.Preload("Language").Preload("Problem").
        Where("id = ?", id).
        First(&sub).Error; err != nil {
        return nil, err
    }
    return &sub, nil
}
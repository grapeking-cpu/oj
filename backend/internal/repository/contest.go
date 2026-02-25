package repository

import (
	"github.com/oj/oj-backend/internal/model"
	"gorm.io/gorm"
)

type ContestRepo struct {
	db *gorm.DB
}

func NewContestRepo(db *gorm.DB) *ContestRepo {
	return &ContestRepo{db: db}
}

func (r *ContestRepo) Create(contest *model.Contest) error {
	return r.db.Create(contest).Error
}

func (r *ContestRepo) GetByID(id int64) (*model.Contest, error) {
	var contest model.Contest
	err := r.db.First(&contest, id).Error
	if err != nil {
		return nil, err
	}
	return &contest, nil
}

func (r *ContestRepo) GetBySlug(slug string) (*model.Contest, error) {
	var contest model.Contest
	err := r.db.Where("slug = ?", slug).First(&contest).Error
	if err != nil {
		return nil, err
	}
	return &contest, nil
}

func (r *ContestRepo) GetByCFID(cfID int) (*model.Contest, error) {
	var contest model.Contest
	err := r.db.Where("cf_contest_id = ?", cfID).First(&contest).Error
	if err != nil {
		return nil, err
	}
	return &contest, nil
}

func (r *ContestRepo) Update(contest *model.Contest) error {
	return r.db.Save(contest).Error
}

func (r *ContestRepo) Delete(id int64) error {
	return r.db.Delete(&model.Contest{}, id).Error
}

type ListContestParams struct {
	Page   int
	PageSize int
	Type   string
	Status string
}

func (r *ContestRepo) List(params ListContestParams) ([]model.Contest, int64, error) {
	var contests []model.Contest
	var total int64

	query := r.db.Model(&model.Contest{})

	if params.Type != "" {
		query = query.Where("type = ?", params.Type)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (params.Page - 1) * params.PageSize
	err = query.Offset(offset).Limit(params.PageSize).Order("start_time DESC").Find(&contests).Error

	return contests, total, err
}

func (r *ContestRepo) Join(contestID, userID int64) error {
	participant := &model.ContestParticipant{
		ContestID: contestID,
		UserID:    userID,
	}
	return r.db.Create(participant).Error
}

func (r *ContestRepo) GetParticipant(contestID, userID int64) (*model.ContestParticipant, error) {
	var p model.ContestParticipant
	err := r.db.Where("contest_id = ? AND user_id = ?", contestID, userID).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ContestRepo) ListParticipants(contestID int64, limit int) ([]model.ContestParticipant, error) {
	var participants []model.ContestParticipant
	err := r.db.Where("contest_id = ?", contestID).
		Order("rank ASC").
		Limit(limit).
		Find(&participants).Error
	return participants, err
}

func (r *ContestRepo) UpdateParticipant(p *model.ContestParticipant) error {
	return r.db.Save(p).Error
}

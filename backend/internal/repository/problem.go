package repository

import (
	"strings"

	"github.com/oj/oj-backend/internal/model"
	"gorm.io/gorm"
)

type ProblemRepo struct {
	db *gorm.DB
}

func NewProblemRepo(db *gorm.DB) *ProblemRepo {
	return &ProblemRepo{db: db}
}

func (r *ProblemRepo) Create(problem *model.Problem) error {
	return r.db.Create(problem).Error
}

func (r *ProblemRepo) GetByID(id int64) (*model.Problem, error) {
	var problem model.Problem
	err := r.db.First(&problem, id).Error
	if err != nil {
		return nil, err
	}
	return &problem, nil
}

func (r *ProblemRepo) GetBySlug(slug string) (*model.Problem, error) {
	var problem model.Problem
	err := r.db.Where("slug = ?", slug).First(&problem).Error
	if err != nil {
		return nil, err
	}
	return &problem, nil
}

func (r *ProblemRepo) Update(problem *model.Problem) error {
	return r.db.Save(problem).Error
}

func (r *ProblemRepo) Delete(id int64) error {
	return r.db.Delete(&model.Problem{}, id).Error
}

type ListProblemParams struct {
	Page       int
	PageSize   int
	Difficulty int
	Tags       string
	Search     string
	IsPublic   *bool
}

func (r *ProblemRepo) List(params ListProblemParams) ([]model.Problem, int64, error) {
	var problems []model.Problem
	var total int64

	query := r.db.Model(&model.Problem{})

	if params.Difficulty > 0 {
		query = query.Where("difficulty = ?", params.Difficulty)
	}
	if params.Tags != "" {
		// tags 是 PostgreSQL 数组，使用 @> 进行 contains 查询
		tagList := strings.Split(params.Tags, ",")
		for _, tag := range tagList {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				query = query.Where("? = ANY(tags)", tag)
			}
		}
	}
	if params.Search != "" {
		search := "%" + params.Search + "%"
		query = query.Where("title ILIKE ?", search)
	}
	if params.IsPublic != nil {
		query = query.Where("is_public = ?", *params.IsPublic)
	}

	query = query.Where("visible = ?", true)

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (params.Page - 1) * params.PageSize
	err = query.Offset(offset).Limit(params.PageSize).
		Order("id DESC").
		Find(&problems).Error

	return problems, total, err
}

func (r *ProblemRepo) IncrementSubmitCount(id int64) error {
	return r.db.Model(&model.Problem{}).Where("id = ?", id).
		UpdateColumn("submit_count", gorm.Expr("submit_count + ?", 1)).Error
}

func (r *ProblemRepo) IncrementAcceptCount(id int64) error {
	// 分两步更新，避免 GORM 多列问题
	err := r.db.Model(&model.Problem{}).Where("id = ?", id).
		UpdateColumn("accept_count", gorm.Expr("accept_count + ?", 1)).Error
	if err != nil {
		return err
	}
	// 更新 accept_rate
	return r.db.Model(&model.Problem{}).Where("id = ?", id).
		Update("accept_rate",
			gorm.Expr("CAST(accept_count AS FLOAT) / NULLIF(submit_count, 0)")).Error
}

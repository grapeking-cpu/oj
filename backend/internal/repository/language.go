package repository

import (
	"github.com/oj/oj-backend/internal/model"
	"gorm.io/gorm"
)

type LanguageRepo struct {
	db *gorm.DB
}

func NewLanguageRepo(db *gorm.DB) *LanguageRepo {
	return &LanguageRepo{db: db}
}

func (r *LanguageRepo) GetByID(id int64) (*model.Language, error) {
	var lang model.Language
	err := r.db.First(&lang, id).Error
	if err != nil {
		return nil, err
	}
	return &lang, nil
}

func (r *LanguageRepo) GetBySlug(slug string) (*model.Language, error) {
	var lang model.Language
	err := r.db.Where("slug = ?", slug).First(&lang).Error
	if err != nil {
		return nil, err
	}
	return &lang, nil
}

func (r *LanguageRepo) List(enabled *bool) ([]model.Language, error) {
	var languages []model.Language

	query := r.db.Order("display_order ASC")
	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}

	err := query.Find(&languages).Error
	return languages, err
}

func (r *LanguageRepo) Create(lang *model.Language) error {
	return r.db.Create(lang).Error
}

func (r *LanguageRepo) Update(lang *model.Language) error {
	return r.db.Save(lang).Error
}

func (r *LanguageRepo) Delete(id int64) error {
	return r.db.Delete(&model.Language{}, id).Error
}

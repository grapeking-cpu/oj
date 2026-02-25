package service

import (
	"github.com/oj/oj-backend/internal/model"
	"github.com/oj/oj-backend/internal/repository"
)

type LanguageService struct {
	repo *repository.LanguageRepo
}

func NewLanguageService(repo *repository.LanguageRepo) *LanguageService {
	return &LanguageService{repo: repo}
}

func (s *LanguageService) GetByID(id int64) (*model.Language, error) {
	return s.repo.GetByID(id)
}

func (s *LanguageService) GetBySlug(slug string) (*model.Language, error) {
	return s.repo.GetBySlug(slug)
}

func (s *LanguageService) List(enabled *bool) ([]model.Language, error) {
	return s.repo.List(enabled)
}

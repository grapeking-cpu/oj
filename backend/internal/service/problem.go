package service

import (
	"errors"

	"github.com/minio/minio-go/v7"
	"github.com/oj/oj-backend/internal/model"
	"github.com/oj/oj-backend/internal/repository"
)

var ErrProblemNotFound = errors.New("problem not found")

type ProblemService struct {
	repo  *repository.ProblemRepo
	minio *minio.Client
}

func NewProblemService(repo *repository.ProblemRepo, minioClient *minio.Client) *ProblemService {
	return &ProblemService{
		repo:  repo,
		minio: minioClient,
	}
}

func (s *ProblemService) GetByID(id int64) (*model.Problem, error) {
	return s.repo.GetByID(id)
}

func (s *ProblemService) List(params repository.ListProblemParams) ([]model.Problem, int64, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20
	}
	params.IsPublic = new(bool)
	*params.IsPublic = true

	return s.repo.List(params)
}

func (s *ProblemService) Create(problem *model.Problem, userID int64) error {
	problem.CreatedBy = &userID
	problem.UpdatedBy = &userID
	return s.repo.Create(problem)
}

func (s *ProblemService) Update(id int64, problem *model.Problem, userID int64) error {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return ErrProblemNotFound
	}
	_ = existing // suppress unused variable warning

	problem.ID = id
	problem.UpdatedBy = &userID
	return s.repo.Update(problem)
}

func (s *ProblemService) Delete(id int64) error {
	return s.repo.Delete(id)
}

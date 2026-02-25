package service

import (
	"errors"
	"time"

	"github.com/oj/oj-backend/internal/model"
	"github.com/oj/oj-backend/internal/repository"
)

var ErrContestNotFound = errors.New("contest not found")

type ContestService struct {
	repo       *repository.ContestRepo
	submitRepo *repository.SubmitRepo
}

func NewContestService(contestRepo *repository.ContestRepo, submitRepo *repository.SubmitRepo) *ContestService {
	return &ContestService{
		repo:       contestRepo,
		submitRepo: submitRepo,
	}
}

func (s *ContestService) GetByID(id int64) (*model.Contest, error) {
	contest, err := s.repo.GetByID(id)
	if err != nil {
		return nil, ErrContestNotFound
	}

	// 更新状态
	s.updateContestStatus(contest)

	return contest, nil
}

func (s *ContestService) List(params repository.ListContestParams) ([]model.Contest, int64, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20
	}

	return s.repo.List(params)
}

func (s *ContestService) Create(contest *model.Contest, userID int64) error {
	contest.CreatedBy = &userID
	contest.Status = "upcoming"
	return s.repo.Create(contest)
}

func (s *ContestService) Update(id int64, contest *model.Contest) error {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return ErrContestNotFound
	}
	_ = existing // suppress unused variable warning

	contest.ID = id
	return s.repo.Update(contest)
}

func (s *ContestService) Delete(id int64) error {
	return s.repo.Delete(id)
}

func (s *ContestService) Join(contestID, userID int64, password string) error {
	contest, err := s.repo.GetByID(contestID)
	if err != nil {
		return ErrContestNotFound
	}

	// 验证密码
	if contest.Password != "" && contest.Password != password {
		return errors.New("invalid password")
	}

	// 检查是否已报名
	_, err = s.repo.GetParticipant(contestID, userID)
	if err == nil {
		return errors.New("already joined")
	}

	return s.repo.Join(contestID, userID)
}

func (s *ContestService) GetRank(contestID int64) ([]RankItem, error) {
	participants, err := s.repo.ListParticipants(contestID, 100)
	if err != nil {
		return nil, err
	}

	items := make([]RankItem, 0, len(participants))
	for _, p := range participants {
		items = append(items, RankItem{
			UserID:  p.UserID,
			Rank:    p.Rank,
			Score:   p.Score,
			Penalty: p.Penalty,
		})
	}

	return items, nil
}

type RankItem struct {
	UserID  int64 `json:"user_id"`
	Rank    int   `json:"rank"`
	Score   int   `json:"score"`
	Penalty int   `json:"penalty"`
}

func (s *ContestService) updateContestStatus(contest *model.Contest) {
	now := time.Now()

	if now.Before(contest.StartTime) {
		contest.Status = "upcoming"
	} else if now.After(contest.EndTime) {
		contest.Status = "ended"
	} else {
		contest.Status = "running"
	}

	// 更新到数据库
	s.repo.Update(contest)
}

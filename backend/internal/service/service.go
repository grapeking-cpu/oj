package service

import (
	"github.com/minio/minio-go/v7"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/oj/oj-backend/internal/repository"
	"github.com/redis/go-redis/v9"
)

// Services 所有 Service 的集合
type Services struct {
	User    *UserService
	Problem *ProblemService
	Submit  *SubmitService
	Contest *ContestService
	Lang    *LanguageService
}

// NewServices 创建 Service 集合
func NewServices(repos *repository.Repositories, rdb *redis.Client, js jetstream.JetStream, minioClient *minio.Client, jwtSecret string) *Services {
	return &Services{
		User:    NewUserService(repos.User, rdb, jwtSecret),
		Problem: NewProblemService(repos.Problem, minioClient),
		Submit:  NewSubmitService(repos.Submit, repos.Lang, js, minioClient, jwtSecret),
		Contest: NewContestService(repos.Contest, repos.Submit),
		Lang:    NewLanguageService(repos.Lang),
	}
}

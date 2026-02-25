package handler

import "github.com/oj/oj-backend/internal/service"

// Handlers 所有 Handler 的集合
type Handlers struct {
	User    *UserHandler
	Problem *ProblemHandler
	Submit  *SubmitHandler
	Contest *ContestHandler
	Lang    *LanguageHandler
}

// NewHandlers 创建 Handler 集合
func NewHandlers(services *service.Services) *Handlers {
	return &Handlers{
		User:    NewUserHandler(services.User),
		Problem: NewProblemHandler(services.Problem),
		Submit:  NewSubmitHandler(services.Submit),
		Contest: NewContestHandler(services.Contest),
		Lang:    NewLanguageHandler(services.Lang),
	}
}

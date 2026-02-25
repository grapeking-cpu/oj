package repository

import "gorm.io/gorm"

// Repositories 所有 Repository 的集合
type Repositories struct {
	User    *UserRepo
	Problem *ProblemRepo
	Submit  *SubmitRepo
	Contest *ContestRepo
	Lang    *LanguageRepo
}

// NewRepositories 创建 Repository 集合
func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		User:    NewUserRepo(db),
		Problem: NewProblemRepo(db),
		Submit:  NewSubmitRepo(db),
		Contest: NewContestRepo(db),
		Lang:    NewLanguageRepo(db),
	}
}

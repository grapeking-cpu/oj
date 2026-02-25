package repository

import (
	"github.com/oj/oj-backend/internal/model"
	"gorm.io/gorm"
)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(user *model.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepo) GetByID(id int64) (*model.User, error) {
	var user model.User
	err := r.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) GetByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) GetByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) Update(user *model.User) error {
	return r.db.Save(user).Error
}

func (r *UserRepo) List(page, pageSize int, role, status string) ([]model.User, int64, error) {
	var users []model.User
	var total int64

	query := r.db.Model(&model.User{})

	if role != "" {
		query = query.Where("role = ?", role)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&users).Error

	return users, total, err
}

func (r *UserRepo) UpdateRating(userID int64, rating int) error {
	return r.db.Model(&model.User{}).Where("id = ?", userID).Update("rating", rating).Error
}

func (r *UserRepo) IncrementSubmitCount(userID int64) error {
	return r.db.Model(&model.User{}).Where("id = ?", userID).
		UpdateColumn("submit_count", gorm.Expr("submit_count + ?", 1)).Error
}

func (r *UserRepo) IncrementAcceptCount(userID int64) error {
	return r.db.Model(&model.User{}).Where("id = ?", userID).
		UpdateColumn("accept_count", gorm.Expr("accept_count + ?", 1)).Error
}

package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/oj/oj-backend/internal/model"
	"github.com/oj/oj-backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrInvalidToken       = errors.New("invalid token")
	ErrCaptchaExpired     = errors.New("captcha expired")
	ErrCaptchaMismatch   = errors.New("captcha mismatch")
)

type UserService struct {
	repo       *repository.UserRepo
	redis      *redis.Client
	jwtSecret  string
}

func NewUserService(repo *repository.UserRepo, redisClient *redis.Client, jwtSecret string) *UserService {
	return &UserService{
		repo:      repo,
		redis:     redisClient,
		jwtSecret: jwtSecret,
	}
}

func (s *UserService) Register(username, email, password, captchaKey, captchaCode string, ip string) (*model.User, string, error) {
	// 验证验证码
	if err := s.verifyCaptcha(captchaKey, captchaCode); err != nil {
		return nil, "", err
	}

	// 检查用户名和邮箱是否存在
	if _, err := s.repo.GetByUsername(username); err == nil {
		return nil, "", ErrUserAlreadyExists
	}
	if _, err := s.repo.GetByEmail(email); err == nil {
		return nil, "", ErrUserAlreadyExists
	}

	// 加密密码
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	user := &model.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		Nickname:     username,
		IPRegister:   ip,
		Role:         "user",
		Status:       "active",
	}

	if err := s.repo.Create(user); err != nil {
		return nil, "", err
	}

	// 生成 Token
	token, err := s.generateToken(user)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (s *UserService) Login(username, password, ip string) (*model.User, string, error) {
	user, err := s.repo.GetByUsername(username)
	if err != nil {
		return nil, "", ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", ErrInvalidPassword
	}

	// 更新最后登录信息
	user.IPLastLogin = ip
	now := time.Now()
	user.LastLoginAt = &now
	s.repo.Update(user)

	// 生成 Token
	token, err := s.generateToken(user)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (s *UserService) GetUserInfo(userID int64) (*model.User, error) {
	return s.repo.GetByID(userID)
}

func (s *UserService) UpdateProfile(userID int64, nickname, avatar string) error {
	user, err := s.repo.GetByID(userID)
	if err != nil {
		return err
	}

	if nickname != "" {
		user.Nickname = nickname
	}
	if avatar != "" {
		user.Avatar = avatar
	}

	return s.repo.Update(user)
}

func (s *UserService) ChangePassword(userID int64, oldPassword, newPassword string) error {
	user, err := s.repo.GetByID(userID)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrInvalidPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.PasswordHash = string(hash)
	return s.repo.Update(user)
}

func (s *UserService) ListUsers(page, pageSize int, role, status string) ([]model.User, int64, error) {
	return s.repo.List(page, pageSize, role, status)
}

func (s *UserService) BanUser(userID int64, ban bool) error {
	user, err := s.repo.GetByID(userID)
	if err != nil {
		return err
	}

	if ban {
		user.Status = "banned"
	} else {
		user.Status = "active"
	}

	return s.repo.Update(user)
}

func (s *UserService) generateToken(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"jti":      uuid.New().String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *UserService) ValidateToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	return token.Claims.(jwt.MapClaims), nil
}

// Captcha
func (s *UserService) GenerateCaptcha() (string, string, error) {
	key := uuid.New().String()
	code := generateCaptchaCode(6)

	ctx := context.Background()
	err := s.redis.Set(ctx, "captcha:"+key, code, 5*time.Minute).Err()
	if err != nil {
		return "", "", err
	}

	// 实际应该返回图片，这里简化返回 key 和一个占位符
	// 真实实现需要使用 captcha 库生成图片
	return key, "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciPjwvc3ZnPg==", nil
}

func (s *UserService) verifyCaptcha(key, code string) error {
	ctx := context.Background()
	stored, err := s.redis.Get(ctx, "captcha:"+key).Result()
	if err == redis.Nil {
		return ErrCaptchaExpired
	}
	if err != nil {
		return err
	}
	if stored != code {
		return ErrCaptchaMismatch
	}
	s.redis.Del(ctx, "captcha:"+key)
	return nil
}

func generateCaptchaCode(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return fmt.Sprintf("%06d", int(b[0])*100000+int(b[1])%100000)[:length]
}

func generateRandomString(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:length]
}

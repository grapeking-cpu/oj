package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/oj/oj-backend/internal/model"
	"github.com/oj/oj-backend/internal/repository"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrInvalidToken      = errors.New("invalid token")
	ErrCaptchaExpired    = errors.New("captcha expired")
	ErrCaptchaMismatch   = errors.New("captcha mismatch")
	ErrAccountLocked     = errors.New("account locked")
	ErrTooManyAttempts   = errors.New("too many attempts")
	ErrInvalidParameters = errors.New("invalid parameters")
)

// 限流配置
const (
	LoginMaxAttempts    = 5       // 5次失败
	LoginBlockDuration  = 15 * time.Minute // 锁定15分钟
	LoginFailWindow     = 5 * time.Minute  // 5分钟窗口
	RegisterMaxPerIP    = 3       // 每小时最多注册3次
	RegisterIPWindow    = 1 * time.Hour
)

type UserService struct {
	repo           *repository.UserRepo
	redis          *redis.Client
	jwtSecret      string
	cookieSecure   bool // 生产环境应为 true
	cookieSameSite string // "Strict" 或 "Lax"
}

func NewUserService(repo *repository.UserRepo, redisClient *redis.Client, jwtSecret string) *UserService {
	return &UserService{
		repo:           repo,
		redis:          redisClient,
		jwtSecret:      jwtSecret,
		cookieSecure:   false, // 本地开发 false
		cookieSameSite: "Lax",
	}
}

// SetCookieSettings 设置 Cookie 安全配置
func (s *UserService) SetCookieSettings(secure bool, sameSite string) {
	s.cookieSecure = secure
	s.cookieSameSite = sameSite
}

func (s *UserService) Register(username, email, password, captchaKey, captchaCode string, ip string) (*model.User, string, error) {
	// 验证验证码
	if err := s.verifyCaptcha(captchaKey, captchaCode); err != nil {
		return nil, "", err
	}

	// 检查用户名格式
	if len(username) < 3 || len(username) > 50 {
		return nil, "", ErrInvalidParameters
	}
	if !isValidUsername(username) {
		return nil, "", ErrInvalidParameters
	}

	// 检查密码复杂度（字母+数字，8-50位）
	if !isValidPassword(password) {
		return nil, "", ErrInvalidParameters
	}

	// 注册防刷：检查 IP 注册频率
	if err := s.checkRegisterRateLimit(ip); err != nil {
		return nil, "", err
	}

	// 检查用户名和邮箱是否存在（不暴露具体哪个存在）
	_, errUser := s.repo.GetByUsername(username)
	_, errEmail := s.repo.GetByEmail(email)
	if errUser == nil || errEmail == nil {
		// 统一错误，不区分是用户名还是邮箱已存在
		return nil, "", errors.New("registration failed")
	}

	// 加密密码（bcrypt cost=12）
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

	// 记录注册次数
	s.incrRegisterCount(ip)

	// 生成 Token
	token, err := s.generateToken(user)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

// isValidUsername 检查用户名格式（字母数字下划线）
func isValidUsername(username string) bool {
	for _, c := range username {
		if !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') && !(c >= '0' && c <= '9') && c != '_' {
			return false
		}
	}
	return true
}

// isValidPassword 检查密码复杂度（字母+数字，8-50位）
func isValidPassword(password string) bool {
	if len(password) < 8 || len(password) > 50 {
		return false
	}
	hasLetter := false
	hasDigit := false
	for _, c := range password {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' {
			hasLetter = true
		}
		if c >= '0' && c <= '9' {
			hasDigit = true
		}
	}
	return hasLetter && hasDigit
}

func (s *UserService) Login(username, password, ip string) (*model.User, string, error) {
	// 登录防刷：检查是否被锁定
	if s.isIPLocked(ip) {
		return nil, "", ErrAccountLocked
	}

	// 获取用户（支持用户名或邮箱登录）
	user, err := s.repo.GetByUsername(username)
	if err != nil {
		// 再尝试邮箱
		user, err = s.repo.GetByEmail(username)
	}

	if err != nil {
		// 记录失败次数（IP维度）
		s.incrLoginFailCount(ip, "ip")
		// 统一错误提示
		return nil, "", errors.New("invalid credentials")
	}

	// 检查密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		// 记录失败次数（IP + 用户名维度）
		s.incrLoginFailCount(ip, "ip")
		s.incrLoginFailCount(username, "user")
		// 统一错误提示
		return nil, "", errors.New("invalid credentials")
	}

	// 检查用户状态
	if user.Status != "active" {
		return nil, "", errors.New("account locked")
	}

	// 登录成功：清除失败记录
	s.clearLoginFailCount(ip, "ip")
	s.clearLoginFailCount(username, "user")

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

// 登录失败计数
func (s *UserService) incrLoginFailCount(key, keyType string) {
	ctx := context.Background()
	redisKey := fmt.Sprintf("login:fail:%s:%s", keyType, key)
	count, err := s.redis.Incr(ctx, redisKey).Result()
	if err == nil && count == 1 {
		s.redis.Expire(ctx, redisKey, LoginFailWindow)
	}
	if count > LoginMaxAttempts {
		// 超过阈值，锁定
		lockKey := fmt.Sprintf("login:lock:%s", key)
		s.redis.Set(ctx, lockKey, "1", LoginBlockDuration)
	}
}

func (s *UserService) clearLoginFailCount(key, keyType string) {
	ctx := context.Background()
	s.redis.Del(ctx, fmt.Sprintf("login:fail:%s:%s", keyType, key))
}

func (s *UserService) isIPLocked(ip string) bool {
	ctx := context.Background()
	lockKey := fmt.Sprintf("login:lock:ip:%s", ip)
	exists, err := s.redis.Exists(ctx, lockKey).Result()
	return err == nil && exists == 1
}

// 注册频率限制
func (s *UserService) checkRegisterRateLimit(ip string) error {
	ctx := context.Background()
	redisKey := fmt.Sprintf("register:count:ip:%s", ip)
	count, err := s.redis.Get(ctx, redisKey).Int()
	if err == redis.Nil {
		s.redis.Set(ctx, redisKey, 1, RegisterIPWindow)
		return nil
	}
	if err != nil {
		return err
	}
	if count >= RegisterMaxPerIP {
		return errors.New("too many registration attempts")
	}
	s.redis.Incr(ctx, redisKey)
	return nil
}

func (s *UserService) incrRegisterCount(ip string) {
	// 注册成功后不需要额外处理，因为 checkRegisterRateLimit 已经设置
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

// Captcha 生成验证码图片
func (s *UserService) GenerateCaptcha() (string, string, error) {
	key := uuid.New().String()
	code := generateCaptchaCode(4) // 4位数字

	ctx := context.Background()
	err := s.redis.Set(ctx, "captcha:"+key, code, 5*time.Minute).Err()
	if err != nil {
		return "", "", err
	}

	// 生成图片
	img := generateCaptchaImage(code)
	imgBase64, err := imageToBase64(img)
	if err != nil {
		return "", "", err
	}

	return key, "data:image/png;base64," + imgBase64, nil
}

// 生成验证码图片
func generateCaptchaImage(code string) image.Image {
	width := 120
	height := 40
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// 背景色
	bg := color.RGBA{255, 255, 255, 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, bg)
		}
	}

	// 随机噪点
	for i := 0; i < 200; i++ {
		x := randInt(0, width)
		y := randInt(0, height)
		c := color.RGBA{uint8(randInt(150, 220)), uint8(randInt(150, 220)), uint8(randInt(150, 220)), 255}
		img.Set(x, y, c)
	}

	// 干扰线
	for i := 0; i < 3; i++ {
		x1 := randInt(0, width)
		y1 := randInt(0, height)
		x2 := randInt(0, width)
		y2 := randInt(0, height)
		lineColor := color.RGBA{uint8(randInt(100, 200)), uint8(randInt(100, 200)), uint8(randInt(100, 200)), 255}
		drawLine(img, x1, y1, x2, y2, lineColor)
	}

	// 绘制文字
	fontColors := []color.RGBA{
		{0, 0, 0, 255},
		{0, 0, 139, 255},
		{0, 100, 0, 255},
		{139, 0, 0, 255},
	}
	for i, c := range code {
		x := 20 + i*25
		y := 12 + randInt(0, 8) // 往上移动文字位置
		fontColor := fontColors[randInt(0, len(fontColors))]
		drawChar(img, x, y, string(c), fontColor)
	}

	return img
}

// 简单的画线函数
func drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.RGBA) {
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	sx := -1
	if x1 < x2 {
		sx = 1
	}
	sy := -1
	if y1 < y2 {
		sy = 1
	}
	err := dx - dy

	for {
		if x1 >= 0 && x1 < img.Bounds().Dx() && y1 >= 0 && y1 < img.Bounds().Dy() {
			img.Set(x1, y1, c)
		}
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

// 简单的画字符（用点模拟）
func drawChar(img *image.RGBA, x, y int, char string, c color.RGBA) {
	// 简化的字符绘制，用小方块模拟
	fontSize := 20
	for dy := 0; dy < fontSize; dy++ {
		for dx := 0; dx < fontSize-5; dx++ {
			if x+dx < img.Bounds().Dx() && y+dy < img.Bounds().Dy() {
				// 简单的字符形状
				if isCharPixel(char, dx, dy) {
					img.Set(x+dx, y+dy, c)
				}
			}
		}
	}
}

func isCharPixel(char string, x, y int) bool {
	// 简化的字符检测
	switch char {
	case "0":
		return (x < 5 || x > 9) && (y < 5 || y > 14) || (y >= 5 && y <= 14 && (x == 2 || x == 12))
	case "1":
		return x > 5 && x < 10 && y < 15 || y == 15
	case "2":
		return y < 5 || y > 10 || x > 8 && y < 10 || x < 5 && y > 10
	case "3":
		return y < 5 || y > 10 || x > 8 || y > 5 && y < 10 && x > 8
	case "4":
		return x > 8 || y < 10 && x < 5 || y > 5 && y < 10 && x > 8
	case "5":
		return y < 5 || y > 10 && x < 5 || y > 5 && y < 10 && x < 5 || y > 10 && x > 8
	case "6":
		return y < 5 || x > 8 || y > 10 && x < 5 || y > 5 && y < 10 && x < 5
	case "7":
		return y < 5 || x > 8
	case "8":
		return (x < 5 || x > 9) && y > 4 && y < 15 || (y < 5 || y > 10) && x > 4 && x < 10
	case "9":
		return y < 5 || x < 5 || y > 10 && x > 9 || y > 5 && y < 10 && x > 9
	default:
		return true
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func randInt(min, max int) int {
	b := make([]byte, 1)
	rand.Read(b)
	return min + int(b[0])%(max-min)
}

func imageToBase64(img image.Image) (string, error) {
	// 使用 PNG 编码
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
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

# 安全与注册风控

## 一、认证与会话

### 1.1 JWT 认证

```go
type Claims struct {
    UserID   int64  `json:"user_id"`
    Username string `json:"username"`
    Role     string `json:"role"`
    ExpiresAt time.Time `json:"exp"`
    jti string `json:"jti"` // Token ID
}

// 生成 Token
func GenerateToken(user *User) (string, error) {
    claims := &Claims{
        UserID:    user.ID,
        Username:  user.Username,
        Role:      user.Role,
        ExpiresAt: time.Now().Add(24 * time.Hour),
        jti:       uuid.New().String(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}
```

### 1.2 Token 存储

- **Header**: `Authorization: Bearer <token>`
- **Redis**: `session:{jti}` 存储 Token 黑名单/主动注销

### 1.3 中间件

```go
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
        if token == "" {
            c.JSON(401, gin.H{"code": 401, "message": "未登录"})
            c.Abort()
            return
        }

        claims, err := ValidateToken(token)
        if err != nil {
            c.JSON(401, gin.H{"code": 401, "message": "Token 无效"})
            c.Abort()
            return
        }

        // 检查 Redis 黑名单
        if redis.Exists("blacklist:" + claims.jti).Val() > 0 {
            c.JSON(401, gin.H{"code": 401, "message": "Token 已注销"})
            c.Abort()
            return
        }

        c.Set("user_id", claims.UserID)
        c.Set("username", claims.Username)
        c.Set("role", claims.Role)
        c.Next()
    }
}
```

---

## 二、RBAC 权限

### 2.1 角色定义

| 角色 | 权限 |
|------|------|
| `user` | 做题、比赛、提交 |
| `admin` | 题目管理、用户管理、比赛管理、系统配置 |

### 2.2 权限检查

```go
func RequireRole(roles ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userRole, _ := c.Get("role")

        for _, role := range roles {
            if userRole == role {
                c.Next()
                return
            }
        }

        c.JSON(403, gin.H{"code": 403, "message": "权限不足"})
        c.Abort()
    }
}

// 使用
r.POST("/problem", middleware.Auth(), middleware.RequireRole("admin"), problemHandler.Create)
```

---

## 三、注册风控

### 3.1 验证码

```go
// 生成验证码
func GenerateCaptcha() (key, imageBase64 string) {
    key = uuid.New().String()

    // 生成图片
    img := captcha.New()
    img.SetSize(120, 40)
    img.SetDisturbance(2)
    img.SetFrontColor(color.RGBA{0, 0, 0, 255})
    img.SetBkgColor(color.RGBA{240, 240, 240, 255})

    code := img.RandomDigit(4)
    img.Write(&buf, code)

    // 存储 Redis
    redis.SetEX("captcha:"+key, strings.ToUpper(code), 5*time.Minute)

    return key, buf.String()
}
```

### 3.2 注册限流

```go
// 同一 IP 1小时内最多注册 3 个账号
func CheckRegisterLimit(ip string) (bool, error) {
    key := fmt.Sprintf("register:ip:%s", ip)
    count, err := redis.Incr(key).Result()
    if err != nil {
        return false, err
    }

    if count == 1 {
        redis.Expire(key, 1*time.Hour)
    }

    return count <= 3, nil
}
```

### 3.3 密码强度

```go
// 密码要求: 8位以上, 包含字母和数字
func ValidatePassword(password string) bool {
    if len(password) < 8 {
        return false
    }

    hasLetter := false
    hasDigit := false
    for _, c := range password {
        if unicode.IsLetter(c) {
            hasLetter = true
        }
        if unicode.IsDigit(c) {
            hasDigit = true
        }
    }

    return hasLetter && hasDigit
}
```

---

## 四、接口限流

### 4.1 限流配置

```go
type RateLimitConfig struct {
    SubmitPerMinute  int `yaml:"submit_per_minute"`
    LoginPerMinute   int `yaml:"login_per_minute"`
    RegisterPerHour  int `yaml:"register_per_hour"`
}

// 限流中间件
func RateLimiter(limit int, window time.Duration) gin.HandlerFunc {
    return func(c *gin.Context) {
        ip := c.ClientIP()
        key := fmt.Sprintf("limit:%s:%s", ip, c.FullPath())

        allowed, err := limiter.Allow(key, limit, window)
        if err != nil {
            c.JSON(500, gin.H{"code": 500, "message": "服务器错误"})
            c.Abort()
            return
        }

        if !allowed {
            c.JSON(429, gin.H{"code": 429, "message": "请求过于频繁，请稍后再试"})
            c.Abort()
            return
        }

        c.Next()
    }
}
```

### 4.2 提交限流

```go
// 每用户每分钟最多 10 次提交
func (s *SubmitService) CheckSubmitLimit(userID int64) (bool, error) {
    key := fmt.Sprintf("limit:user:%d:submit", userID)
    count, err := redis.Incr(key).Result()
    if err != nil {
        return false, err
    }

    if count == 1 {
        redis.Expire(key, 1*time.Minute)
    }

    return count <= 10, nil
}
```

---

## 五、输入安全

### 5.1 SQL 注入防护

- 使用 ORM (GORM/SQLC) 参数化查询
- 禁止拼接 SQL

### 5.2 XSS 防护

```go
// 题目内容使用 sanitize-html 过滤
import "github.com/microcosm-cc/bluemonday"

func Sanitize(input string) string {
    policy := bluemonday.UGCPolicy()
    return policy.Sanitize(input)
}
```

### 5.3 文件上传

```go
// 限制上传类型和大小
func ValidateUpload(filename string, size int64) error {
    // 检查扩展名
    ext := strings.ToLower(filepath.Ext(filename))
    allowed := []string{".zip", ".jpg", ".png", ".pdf"}

    for _, e := range allowed {
        if ext == e {
            // 检查大小
            if size > 10*1024*1024 { // 10MB
                return errors.New("文件过大")
            }
            return nil
        }
    }

    return errors.New("不支持的文件类型")
}
```

---

## 六、API 安全

### 6.1 请求体大小

```go
r.Use(gin.BodyLimit("10M"))
```

### 6.2 CORS

```go
config := cors.DefaultConfig()
config.AllowOrigins = []string{"https://your-domain.com"}
config.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
r.Use(cors.New(config))
```

### 6.3 请求日志

```go
// 记录请求日志
func LoggerMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        path := c.Request.URL.Path

        c.Next()

        latency := time.Since(start)
        status := c.Writer.Status()

        log.Printf("[%d] %s %s %v", status, c.Request.Method, path, latency)
    }
}
```

---

## 七、敏感操作

### 7.1 操作日志

```go
func LogOperation(userID int64, action, targetType string, targetID int64, detail map[string]interface{}) {
    log := OperationLog{
        UserID:     userID,
        Action:     action,
        TargetType: targetType,
        TargetID:   targetID,
        Detail:     detail,
        IP:         getClientIP(),
        UserAgent:  getUserAgent(),
        CreatedAt:  time.Now(),
    }

    db.Create(&log)
}
```

### 7.2 管理员操作需二次确认

```go
// 删除题目、禁用用户等敏感操作需要额外确认
r.POST("/admin/problem/:id/delete",
    middleware.Auth(),
    middleware.RequireRole("admin"),
    middleware.CSRFToken(), // CSRF 防护
    adminHandler.DeleteProblem)
```

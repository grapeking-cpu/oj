# 认证模块使用说明

## 概述

本系统使用 JWT + HttpOnly Cookie 实现用户认证，具有以下安全特性：
- 密码使用 bcrypt 强哈希存储
- 登录防爆破（IP + 用户名维度限流）
- 注册防刷（IP 维度限流）
- 图形验证码

## 接口列表

### 1. 获取验证码
```
GET /api/v1/user/captcha
```
返回：
```json
{
  "code": 0,
  "data": {
    "captcha_key": "uuid-string",
    "captcha_image": "data:image/png;base64,..."
  }
}
```

### 2. 注册用户
```
POST /api/v1/user/register
Content-Type: application/json

{
  "username": "string",      // 3-50位，字母数字下划线
  "email": "string",         // 有效邮箱
  "password": "string",      // 8-50位，含字母+数字
  "captcha_key": "string",   // 验证码 Key
  "captcha_code": "string"   // 验证码
}
```

成功响应（设置 HttpOnly Cookie）：
```json
{
  "code": 0,
  "data": {
    "user_id": 123,
    "username": "testuser",
    "nickname": "testuser",
    "role": "user"
  }
}
```

### 3. 用户登录
```
POST /api/v1/user/login
Content-Type: application/json

{
  "username": "string",   // 用户名或邮箱
  "password": "string"
}
```

成功响应（设置 HttpOnly Cookie）：
```json
{
  "code": 0,
  "data": {
    "user_id": 123,
    "username": "testuser",
    "nickname": "测试用户",
    "role": "user",
    "rating": 1500
  }
}
```

### 4. 获取当前用户信息
```
GET /api/v1/user/info
Authorization: Bearer <token>  // 可选，Cookie 优先
```

### 5. 退出登录
```
POST /api/v1/user/logout
```

## 错误码

| code | message | 说明 |
|------|---------|------|
| 400 | invalid parameters | 参数校验失败 |
| 400 | invalid captcha | 验证码错误 |
| 400 | registration failed | 注册失败 |
| 401 | invalid credentials | 用户名或密码错误 |
| 401 | authorization required | 未登录 |
| 429 | too many attempts | 登录尝试过多 |

## 限流配置

- 登录失败：同一 IP/用户名 5 分钟内 5 次失败后锁定 15 分钟
- 注册：同一 IP 每小时最多注册 3 次

## 环境变量

详见 `backend/.env.example`

```bash
# JWT 密钥（生产环境请使用强随机字符串）
JWT_SECRET=your-jwt-secret-change-in-production

# CORS 允许的 origins
ALLOW_ORIGINS=http://localhost:5173,http://localhost:3000
```

## 运行测试

```bash
cd backend
go test -v ./internal/handler/...
```

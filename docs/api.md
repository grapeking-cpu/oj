# API 接口清单

## 通用约定

| 项目 | 说明 |
|------|------|
| 基础路径 | `/api/v1` |
| 认证 | JWT Bearer Token (Header: `Authorization: Bearer <token>`) |
| 错误码 | `code: 0=成功, 非0=失败, message=描述` |
| 分页 | `page, page_size` |
| 时间 | ISO8601, 如 `2024-01-01T00:00:00Z` |

---

## 一、用户模块 `user`

### 1.1 注册
```
POST /user/register
Body: {
    "username": "alice",
    "email": "alice@example.com",
    "password": "123456",
    "captcha_key": "xxx",
    "captcha_code": "abcd"
}
Response: {
    "code": 0,
    "data": { "user_id": 1, "token": "..." }
}
```

### 1.2 登录
```
POST /user/login
Body: {
    "username": "alice",
    "password": "123456"
}
Response: {
    "code": 0,
    "data": { "user_id": 1, "token": "..." }
}
```

### 1.3 验证码
```
GET /user/captcha
Response: { "captcha_key": "uuid", "captcha_image": "base64" }
```

### 1.4 获取用户信息
```
GET /user/info
Auth: Required
Response: { "code": 0, "data": { "id":1, "username":"alice", "rating":1500, ... } }
```

### 1.5 更新个人信息
```
PUT /user/profile
Auth: Required
Body: { "nickname": "Alice", "avatar": "url" }
```

### 1.6 修改密码
```
PUT /user/password
Auth: Required
Body: { "old_password": "x", "new_password": "y" }
```

### 1.7 用户列表 (Admin)
```
GET /user/list?page=1&page_size=20&role=admin&status=banned
Auth: Admin
```

### 1.8 禁用/启用用户 (Admin)
```
POST /user/ban
Auth: Admin
Body: { "user_id": 1, "ban": true }
```

---

## 二、语言模块 `language`

### 2.1 语言列表
```
GET /language/list
Response: {
    "code": 0,
    "data": [
        {
            "id": 1,
            "name": "C++17",
            "slug": "cpp17",
            "source_filename": "main.cpp",
            "docker_image": "oj-runner-cpp17",
            "time_factor": 1.0,
            "enabled": true
        }
    ]
}
```

---

## 三、题库模块 `problem`

### 3.1 题目列表
```
GET /problem/list?page=1&page_size=20&difficulty=3&tags=a,b&search=title
Response: {
    "code": 0,
    "data": {
        "list": [
            { "id":1, "title":"A+B", "difficulty":1, "tags":["入门"], "accept_rate":0.85 }
        ],
        "total": 100
    }
}
```

### 3.2 题目详情
```
GET /problem/:id
Response: {
    "code": 0,
    "data": {
        "id": 1,
        "title": "A+B",
        "description": "## 题目描述\n...",
        "input_format": "...",
        "sample_io": [{"input":"1 2","output":"3"}],
        "time_limit": 1000,
        "memory_limit": 256,
        ...
    }
}
```

### 3.3 创建题目 (Admin)
```
POST /problem
Auth: Admin
Body: {
    "title": "A+B",
    "description": "...",
    "difficulty": 1,
    "tags": ["入门"],
    "time_limit": 1000,
    "memory_limit": 256,
    "sample_io": [{"input":"1 2","output":"3"}],
    "test_cases": [...],
    "is_spj": false
}
```

### 3.4 更新题目 (Admin)
```
PUT /problem/:id
Auth: Admin
Body: { ... }
```

### 3.5 删除题目 (Admin)
```
DELETE /problem/:id
Auth: Admin
```

### 3.6 上传测试数据
```
POST /problem/:id/testdata
Auth: Admin
Body: FormData { file: .zip }
Response: { "code": 0, "data": { "url": "minio://...", "hash": "sha256" } }
```

---

## 四、提交模块 `submit` (核心)

### 4.1 提交代码
```
POST /submit
Auth: Required
Body: {
    "problem_id": 1,
    "language_id": 1,
    "code": "#include ...",
    "idempotency_key": "uuid"  // 幂等key
}
Response: {
    "code": 0,
    "data": {
        "submit_id": "uuid",
        "status": "PENDING",
        "estimated_time": 5  // 秒
    }
}
```

### 4.2 提交详情
```
GET /submit/:submit_id
Response: {
    "code": 0,
    "data": {
        "submit_id": "uuid",
        "status": "FINISHED",
        "score": 100,
        "result": {
            "status": "FINISHED",
            "score": 100,
            "accepted_test": 10,
            "total_test": 10,
            "time_ms": 125,
            "memory_kb": 35840,
            "cases": [...]
        }
    }
}
```

### 4.3 我的提交列表
```
GET /submit/list?problem_id=1&status=FINISHED&page=1
Auth: Required
```

### 4.4 代码查重 (Admin)
```
GET /submit/duplicate?problem_id=1
Auth: Admin
Response: { "code": 0, "data": [[submit_id1, submit_id2, similarity:0.95], ...] }
```

---

## 五、比赛模块 `contest`

### 5.1 比赛列表
```
GET /contest/list?type=ACM&status=running
Response: {
    "code": 0,
    "data": [
        { "id":1, "title":"周赛#1", "type":"ACM", "status":"running", "start_time":"..." }
    ]
}
```

### 5.2 比赛详情
```
GET /contest/:id
Response: {
    "code": 0,
    "data": {
        "id": 1,
        "title": "周赛#1",
        "type": "ACM",
        "start_time": "...",
        "end_time": "...",
        "problems": [{"letter":"A","id":1,"title":"A+B"}],
        "rule_type": "ACM",
        "status": "running"
    }
}
```

### 5.3 参加比赛
```
POST /contest/:id/join
Auth: Required
Body: { "password": "xxx" }  // 如果需要
Response: { "code": 0 }
```

### 5.4 比赛榜单
```
GET /contest/:id/rank?page=1&force=0  // force=1 强制刷新缓存
Response: {
    "code": 0,
    "data": {
        "frozen": true,
        "official": [...],  // 正式榜
        "unofficial": [...] //  unofficial (if frozen)
    }
}
```

### 5.5 比赛内提交
```
POST /contest/:id/submit
Auth: Required
Body: { "problem_letter": "A", "language_id": 1, "code": "..." }
Response: { "code": 0, "data": { "submit_id": "uuid" } }
```

### 5.6 创建比赛 (Admin)
```
POST /contest
Auth: Admin
Body: {
    "title": "周赛#1",
    "type": "ACM",
    "start_time": "2024-01-01T10:00:00Z",
    "end_time": "2024-01-01T14:00:00Z",
    "frozen_minutes": 30,
    "problems": [1,2,3],
    "password": ""
}
```

### 5.7 同步 Codeforces
```
POST /contest/sync/cf
Auth: Admin
Body: { "cf_contest_id": 1900 }
Response: { "code": 0, "data": { "contest_id": 10 } }
```

---

## 六、排行榜模块 `rank`

### 6.1 全局排名
```
GET /rank/global?page=1&page_size=50
Response: {
    "code": 0,
    "data": [
        { "rank":1, "user_id":1, "username":"alice", "rating":2000, "submit_count":500 }
    ]
}
```

### 6.2 用户Rating历史
```
GET /rank/history/:user_id
Response: { "code": 0, "data": [{ "contest_id":1, "rating_before":1500, "rating_after":1550, "delta":50, "created_at":"..." }] }
```

---

## 七、WebSocket 推送

### 7.1 连接
```
WS /ws?token=<jwt>
```

### 7.2 订阅主题
```json
{
    "action": "subscribe",
    "topic": "contest:1:rank"  // 比赛榜单
}
```

### 7.3 推送消息示例
```json
// 提交状态更新
{
    "type": "submit_status",
    "data": {
        "submit_id": "uuid",
        "status": "FINISHED",
        "score": 100
    }
}

// 比赛榜单更新
{
    "type": "contest_rank",
    "contest_id": 1,
    "data": [...]
}
```

---

## 八、系统配置 `system`

### 8.1 获取配置
```
GET /system/config
```

### 8.2 更新配置 (Admin)
```
PUT /system/config
Auth: Admin
Body: { "key": "judge.max_parallel", "value": "10" }
```

---

## 九、Admin 后台

| 接口 | 说明 |
|------|------|
| `GET /admin/dashboard` | 统计数据面板 |
| `GET /admin/judge/queue` | 查看待评测队列 |
| `POST /admin/judge/rejudge` | 重新评测 |
| `POST /admin/judge/stop` | 停止某提交 |
| `GET /admin/logs` | 操作日志 |

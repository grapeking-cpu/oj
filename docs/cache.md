# 缓存与排行榜策略

## 一、Redis 缓存设计

### 1.1 Key 命名规范

```
{prefix}:{entity}:{id}[:{field}]
```

| 前缀 | 实体 | 示例 |
|------|------|------|
| `user` | 用户 | `user:1` |
| `problem` | 题目 | `problem:1` |
| `contest` | 比赛 | `contest:1` |
| `rank` | 排名 | `rank:global` |
| `captcha` | 验证码 | `captcha:abc123` |
| `limit` | 限流 | `limit:ip:10.0.0.1:submit` |

### 1.2 缓存策略

| 缓存项 | 过期时间 | 淘汰策略 | 说明 |
|--------|----------|----------|------|
| `user:{id}` | 30min | LRU | 用户信息 |
| `problem:{id}` | 1h | LRU | 题目详情 |
| `problem:list:*` | 5min | LRU | 题目列表 |
| `contest:{id}` | 1min | LRU | 比赛信息 |
| `rank:global` | 1min | LRU | 全局排名 |
| `rank:contest:{id}` | 30s | LRU | 比赛排名 |
| `captcha:{key}` | 5min | TTL | 验证码 |
| `session:{token}` | 24h | TTL | 会话 |
| `limit:*` | 1min | TTL | 限流计数器 |

---

## 二、用户 Session

### 2.1 存储结构

```go
type Session struct {
    UserID    int64     `json:"user_id"`
    Username  string    `json:"username"`
    Role     string    `json:"role"`
    ExpiresAt time.Time `json:"expires_at"`
}
```

```bash
# Key: session:{token}
# Value: JSON
# TTL: 24h
```

### 2.2 登录流程

```
POST /user/login
    |
    --> 验证密码
    --> 生成 UUID token
    --> Redis SET session:{token} {user_info}
    --> 返回 token
```

---

## 三、限流设计

### 3.1 滑动窗口限流

```go
type RateLimiter struct {
    redis *redis.Client
}

func (r *RateLimiter) Allow(key string, limit int, window time.Duration) (bool, error) {
    now := time.Now().Unix()
    windowStart := now - int64(window.Seconds())

    pipe := r.redis.Pipeline()

    // 删除窗口外的数据
    pipe.ZRemRangeByScore(key, "0", fmt.Sprintf("%d", windowStart))

    // 添加当前请求
    pipe.ZAdd(key, redis.Z{Score: float64(now), Member: now})

    // 统计窗口内请求数
    pipe.ZCard(key)

    // 设置过期
    pipe.Expire(key, window)

    _, err := pipe.Exec()
    if err != nil {
        return false, err
    }

    count, _ := pipe.ZCard(key).Result()
    return count <= int64(limit), nil
}
```

### 3.2 限流规则

| 操作 | 限制 | 窗口 |
|------|------|------|
| 提交代码 | 10次 | 1分钟 |
| 注册 | 3次 | 1小时/IP |
| 登录 | 20次 | 1分钟/IP |
| 获取验证码 | 5次 | 1小时/IP |

---

## 四、排行榜缓存

### 4.1 Global Rank (全局排名)

```go
// 使用 Redis Sorted Set
// Key: rank:global
// Member: user_id
// Score: rating (或 accept_count)

func GetGlobalRank(redis *redis.Client, page, pageSize int) ([]UserRank, error) {
    start := (page - 1) * pageSize
    end := start + pageSize - 1

    // 获取排名用户ID
    ids, err := redis.ZRevRange("rank:global", int64(start), int64(end)).Result()
    if err != nil {
        return nil, err
    }

    // 批量获取用户信息 (Pipeline)
    pipe := redis.Pipeline()
    cmds := make([]*redis.StringCmd, len(ids))
    for i, id := range ids {
        cmds[i] = pipe.Get(fmt.Sprintf("user:%s", id))
    }
    pipe.Exec()

    // 解析结果
    results := make([]UserRank, len(ids))
    for i, id := range ids {
        rank, _ := redis.ZRevRank("rank:global", id).Result()
        score, _ := redis.ZScore("rank:global", id).Result()
        results[i] = UserRank{
            Rank:     int(rank) + 1,
            UserID:   id,
            Rating:   int(score),
        }
    }

    return results, nil
}
```

### 4.2 比赛排名 (ACM/IOI)

```python
# 比赛榜单 Redis 结构
# Key: contest:rank:{contest_id}
# Key Frozen: contest:rank:{contest_id}:frozen (封榜后)
# Sorted Set: user_id -> score ( penalty * 10000 + -solve_time)

# 实时更新 (Watch + Transaction)
def update_contest_rank(contest_id, user_id, solved, penalty):
    redis.zincrby(f"contest:rank:{contest_id}", 1, user_id)
    # 同时维护 solved 计数
    redis.hincrby(f"contest:solved:{contest_id}", user_id, 1)
    redis.hincrby(f"contest:penalty:{contest_id}", user_id, penalty)
```

### 4.3 榜单推送

```go
// WebSocket 推送榜单变化
func (h *Hub) broadcastRankUpdate(contestID int, rankData []byte) {
    h.mu.RLock()
    room, ok := h.rooms[fmt.Sprintf("contest:%d", contestID)]
    h.mu.RUnlock()

    if !ok {
        return
    }

    room.broadcast <- []byte(`{"type":"rank_update","data":` + string(rankData) + `}`)
}
```

### 4.4 封榜机制

```go
// 封榜时切换到 frozen key
func (c *ContestService) FreezeBoard(contestID int) error {
    // 复制当前排名到 frozen key
    redis.Rename(f"contest:rank:{contestID}", f"contest:rank:{contestID}:frozen")

    // 封榜后新提交存入 shadow key
    redis.Delete(f"contest:rank:{contestID}:shadow")
}

// 解封后合并
func (c *ContestService) UnfreezeBoard(contestID int) error {
    // 合并 shadow 到正式榜
    redis.ZUnionStore(f"contest:rank:{contestID}",
        redis.ZStore{},
        f"contest:rank:{contestID}",
        f"contest:rank:{contestID}:shadow",
    )
}
```

---

## 五、热点数据缓存

### 5.1 题目缓存

```go
// GetProblem 缓存逻辑
func (s *ProblemService) GetProblem(id int64) (*Problem, error) {
    // 1. 查缓存
    cacheKey := fmt.Sprintf("problem:%d", id)
    cached, err := s.redis.Get(cacheKey).Result()
    if err == nil {
        var problem Problem
        json.Unmarshal([]byte(cached), &problem)
        return &problem, nil
    }

    // 2. 查 DB
    problem, err := s.db.GetProblem(id)
    if err != nil {
        return nil, err
    }

    // 3. 写入缓存 (30min)
    data, _ := json.Marshal(problem)
    s.redis.Set(cacheKey, data, 30*time.Minute)

    return problem, nil
}

// UpdateProblem 时删除缓存
func (s *ProblemService) UpdateProblem(id int64, data map[string]interface{}) error {
    // DB 更新
    s.db.UpdateProblem(id, data)

    // 删除缓存
    s.redis.Delete(fmt.Sprintf("problem:%d", id))

    return nil
}
```

### 5.2 列表缓存与分页

```go
// 问题列表缓存
func (s *ProblemService) ListProblems(params ListParams) ([]Problem, int64, error) {
    cacheKey := fmt.Sprintf("problem:list:%d:%d:%s", params.Page, params.PageSize, params.Tags)

    // 尝试从缓存获取
    cached, err := s.redis.Get(cacheKey).Result()
    if err == nil {
        var result struct {
            List  []Problem `json:"list"`
            Total int64     `json:"total"`
        }
        json.Unmarshal([]byte(cached), &result)
        return result.List, result.Total, nil
    }

    // 查 DB
    list, total, err := s.db.ListProblems(params)
    if err != nil {
        return nil, 0, err
    }

    // 写入缓存 (5min)
    data, _ := json.Marshal(map[string]interface{}{"list": list, "total": total})
    s.redis.Set(cacheKey, data, 5*time.Minute)

    return list, total, nil
}
```

---

## 六、缓存失效策略

### 6.1 主动失效

| 场景 | 失效方式 |
|------|----------|
| 更新题目 | 删除 `problem:{id}` |
| 新增提交 | 删除 `rank:*` 相关缓存 |
| 比赛结束 | 重建比赛榜单 |

### 6.2 TTL 兜底

- 所有缓存设置 TTL，防止永久残留
- 推荐: 5min ~ 30min

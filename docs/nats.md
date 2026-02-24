# NATS JetStream 队列设计

## 一、Streams 配置

| Stream | 存储 | 副本 | 最大内存 | 保留策略 |
|--------|------|------|----------|----------|
| `JUDGE_TASKS` | File | 1 | 100MB | 24h 或 10万条 |
| `JUDGE_EVENTS` | File | 1 | 50MB | 1h |
| `JUDGE_DLQ` | File | 1 | 100MB | 7d |

```bash
# Stream: JUDGE_TASKS (评测任务)
jetstream stream add JUDGE_TASKS \
  --subjects "judge.tasks.*" \
  --storage file \
  --replicas 1 \
  --max-bytes 100MB \
  --max-age 24h \
  --max-msg-size 1MB

# Stream: JUDGE_EVENTS (事件推送)
jetstream stream add JUDGE_EVENTS \
  --subjects "judge.events.*" \
  --storage file \
  --replicas 1 \
  --max-bytes 50MB \
  --max-age 1h

# Stream: JUDGE_DLQ (死信队列)
jetstream stream add JUDGE_DLQ \
  --subjects "judge.dlq.*" \
  --storage file \
  --replicas 1 \
  --max-bytes 100MB \
  --max-age 7d
```

---

## 二、Subjects 分流设计

### 2.1 任务Subjects

| Subject | 用途 | 优先级 | 并发限制 |
|---------|------|--------|----------|
| `judge.tasks.light` | Python/Go/解释型 | 高 | 每Worker 2并发 |
| `judge.tasks.heavy` | C++/Java/编译型 | 中 | 每Worker 1并发 |

```go
// 分流规则
func getTaskSubject(langSlug string) string {
    heavyLangs := map[string]bool{
        "cpp17": true, "c11": true, "java17": true,
    }
    if heavyLangs[langSlug] {
        return "judge.tasks.heavy"
    }
    return "judge.tasks.light"
}
```

### 2.2 事件Subjects

| Subject | 用途 |
|---------|------|
| `judge.events.status` | 提交状态变更 |
| `judge.events.contest` | 比赛榜单更新 |
| `judge.events.system` | 系统通知 |

---

## 三、Consumers 设计

### 3.1 Light Worker Consumer

```bash
jetstream consumer add JUDGE_TASKS judge-light \
  --subject judge.tasks.light \
  --deliver new \
  --max-ack-pending 10 \
  --ack-timeout 30s \
  --max-deliver 3 \           # 最多投递3次
  --backoff "5s,10s,30s"     # 指数退避
```

### 3.2 Heavy Worker Consumer

```bash
jetstream consumer add JUDGE_TASKS judge-heavy \
  --subject judge.tasks.heavy \
  --deliver new \
  --max-ack-pending 5 \
  --ack-timeout 120s \
  --max-deliver 3 \
  --backoff "10s,30s,60s"
```

### 3.3 DLQ Consumer (死信)

```bash
jetstream consumer add JUDGE_DLQ judge-dlq-monitor \
  --subject "judge.dlq.*" \
  --deliver all \
  --ack-timeout 60s
```

---

## 四、消息格式设计

### 4.1 评测任务消息 (NATS Message)

```json
{
    "submit_id": "uuid-xxx",
    "idempotency_key": "client-provided-key",
    "problem": {
        "id": 1,
        "title": "A+B",
        "time_limit": 1000,
        "memory_limit": 256,
        "stack_limit": 64,
        "is_spj": false,
        "test_data_zip": "minio://bucket/testdata/1.zip",
        "test_data_hash": "sha256..."
    },
    "language": {
        "id": 1,
        "slug": "cpp17",
        "source_filename": "main.cpp",
        "compile_cmd": ["g++", "-o", "main", "main.cpp"],
        "run_cmd": ["./main"],
        "docker_image": "oj-runner-cpp17",
        "time_factor": 1.0,
        "memory_factor": 1.0,
        "output_limit": 65536,
        "pids_limit": 64
    },
    "code": "#include <iostream>...",
    "contest": {
        "id": 1,
        "type": "ACM",
        "penalty_minutes": 20,
        "frozen_minutes": 30,
        "is_virtual": false
    },
    "user": {
        "id": 1,
        "username": "alice"
    },
    "retry_count": 0,
    "created_at": "2024-01-01T10:00:00Z"
}
```

### 4.2 状态事件消息

```json
{
    "type": "status_change",
    "submit_id": "uuid-xxx",
    "status": "RUNNING",
    "worker_id": "judge-worker-1",
    "timestamp": "2024-01-01T10:00:05Z"
}
```

```json
{
    "type": "result",
    "submit_id": "uuid-xxx",
    "status": "FINISHED",
    "score": 100,
    "result": {
        "accepted_test": 10,
        "total_test": 10,
        "time_ms": 125,
        "memory_kb": 35840,
        "cases": [...]
    },
    "timestamp": "2024-01-01T10:00:30Z"
}
```

---

## 五、ACK 与重试策略

### 5.1 成功流程

```
API Server                  NATS                    Judge Worker
    |                          |                          |
    |--publish submit-------->|                          |
    |<--ack success-----------|                          |
    |                          |                          |
    |                          |---pull task (light)----->|
    |                          |<--ack (30s timeout)------|
    |                          |                          |
    |                          |<--publish result---------|
    |<---subscribe events-----|                          |
    |                          |                          |
```

### 5.2 失败重试流程

```
Worker 处理失败 (编译错误/系统异常)
    |
    |--重试1 (5s)--> 重新消费
    |--重试2 (10s)--> 重新消费
    |--重试3 (30s)--> 重新消费
    |
    |--超过max_deliver--> 进入DLQ
```

### 5.3 超时回收

```go
// Worker 侧超时回收策略
const (
    DefaultTimeout   = 30 * time.Second  // Light
    HeavyTimeout     = 120 * time.Second // Heavy
    GlobalMaxTimeout = 5 * time.Minute   // 全局超时
)

// 任务超时后，状态回滚为 PENDING，重新入队
// 并增加 retry_count
```

---

## 六、资源隔离与容量规划

### 6.1 Worker 并发配置

| Worker 类型 | 并发数 | 适用语言 | 内存预估 |
|-------------|--------|----------|----------|
| Light-1 | 2 | Python/Go | 2GB |
| Light-2 | 2 | Python/Go | 2GB |
| Heavy-1 | 1 | C++/Java | 4GB |

**单机部署建议**: Light×2 + Heavy×1 = 总并发 4~6

### 6.2 队列积压告警

```bash
# 监控队列长度
nats stream info JUDGE_TASKS
# 或设置监控告警: 队列 > 1000 时报警
```

---

## 七、幂等设计

### 7.1 消息去重

```go
// 消费者侧
type JudgeTask struct {
    SubmitID         string `json:"submit_id"`
    IdempotencyKey   string `json:"idempotency_key"` // 客户端提供
}

// 消费前检查 DB 中是否已存在处理结果
// 如果存在，直接跳过
func (j *JudgeWorker) isAlreadyProcessed(key string) bool {
    var count int64
    j.db.Model(&Submission{}).Where("idempotency_key = ?", key).Count(&count)
    return count > 0
}
```

### 7.2 状态机转换

```
PENDING --> RUNNING (worker_id, start_time)
       --> SYSTEM_ERROR (retry_count++, requeue)
       --> DLQ (retry exhausted)

RUNNING --> FINISHED (score, result)
        --> SYSTEM_ERROR (timeout, requeue)
        --> DLQ (retry exhausted)
```

### 7.3 更新幂等

```go
// DB 更新使用条件更新，防止覆盖
UPDATE submissions
SET judge_result = $1, worker_id = $2, finish_time = $3
WHERE submit_id = $4
  AND status != 'FINISHED'  -- 防止重复写入
```

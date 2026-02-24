# Judge Worker 并发模型与幂等策略

## 一、Worker 架构

```
                    ┌─────────────────────────────────────┐
                    │         Judge Worker                │
                    │  ┌─────────────────────────────┐   │
                    │  │     NATS Consumer            │   │
                    │  │  (light/heavy 消费者)         │   │
                    │  └──────────────┬────────────────┘   │
                    │                 │                    │
                    │  ┌──────────────▼────────────────┐   │
                    │  │     Task Dispatcher           │   │
                    │  │  (goroutine pool)             │   │
                    │  └──────────────┬────────────────┘   │
                    │                 │                    │
         ┌──────────┼─────────────────┼────────────────────┼──────────┐
         │          │                 │                    │          │
    ┌────▼────┐ ┌───▼────┐      ┌────▼────┐         ┌─────▼─────┐    │
    │Worker 1 │ │Worker 2│      │Worker N │         │  MinIO    │    │
    │(Light)  │ │(Light) │      │(Heavy)  │         │(日志上传)  │    │
    └────┬────┘ └───┬────┘      └────┬────┘         └───────────┘    │
         │          │                 │                                  │
         └──────────┼─────────────────┼──────────────────────────────────┘
                    │                 │
            ┌───────▼─────────────────▼───────┐
            │       Docker Runner            │
            │  ┌────┐ ┌────┐ ┌────┐ ┌────┐  │
            │  │cpp │ │py  │ │go  │ │java│  │
            │  │box │ │box │ │box │ │box │  │
            │  └────┘ └────┘ └────┘ └────┘  │
            └───────────────────────────────┘
```

---

## 二、并发模型

### 2.1 Worker 启动参数

```bash
./judge-worker \
  --worker-id=judge-light-1 \
  --consumer=judge-light \
  --concurrency=2 \
  --nats-server=nats://nats:4222 \
  --log-level=debug
```

### 2.2 Goroutine Pool 实现

```go
// 使用 worker pool 控制并发
type WorkerPool struct {
    queue      chan *JudgeTask
    workers    int
    wg         sync.WaitGroup
}

func NewWorkerPool(workers int) *WorkerPool {
    return &WorkerPool{
        queue:   make(chan *JudgeTask, workers*2),
        workers: workers,
    }
}

func (p *WorkerPool) Start(handler func(*JudgeTask)) {
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go func() {
            defer p.wg.Done()
            for task := range p.queue {
                handler(task)
            }
        }()
    }
}

func (p *WorkerPool) Submit(task *JudgeTask) {
    p.queue <- task
}
```

### 2.3 状态机流转

```
┌─────────────────────────────────────────────────────────────────┐
│                        评测状态机                                │
│                                                                 │
│   ┌─────────┐     ┌─────────┐     ┌───────────┐     ┌────────┐ │
│   │ PENDING │────▶│RUNNING  │────▶│  FINISHED │     │  DLQ   │ │
│   └─────────┘     └─────────┘     └───────────┘     └────────┘ │
│       ▲               │                  │                    │
│       │               │                  │                    │
│       │          ┌────▼─────┐            │                    │
│       │          │SYSTEM_ERR│────────────┘                    │
│       │          │(重试)    │                                 │
│       │          └──────────┘                                 │
│       │               │                                       │
│       │               ▼                                       │
│       │          ┌─────────┐                                  │
│       └──────────│  DLQ    │                                  │
│                  └─────────┘                                  │
└──────────────────────────────────────────────────────────────┘
```

---

## 三、评测流程

### 3.1 主流程

```go
func (w *JudgeWorker) ProcessTask(task *JudgeTask) error {
    // 1. 预检查：幂等验证
    if w.isAlreadyFinished(task.SubmitID) {
        log.Printf("Task %s already processed, skip", task.SubmitID)
        return nil
    }

    // 2. 更新状态为 RUNNING
    w.updateStatus(task.SubmitID, "RUNNING", w.workerID)

    // 3. 创建工作目录
    workspace, err := w.prepareWorkspace(task)
    if err != nil {
        return w.handleError(task, err)
    }
    defer w.cleanup(workspace)

    // 4. 编译阶段
    compileResult, err := w.compile(task, workspace)
    if err != nil {
        w.saveCompileResult(task, compileResult)
        return w.handleError(task, err)
    }

    // 5. 运行测试
    results, err := w.runTestCases(task, workspace)
    if err != nil {
        return w.handleError(task, err)
    }

    // 6. 聚合结果
    finalResult := w.aggregateResults(results)

    // 7. 保存结果
    w.saveResult(task, finalResult)

    // 8. 发布事件
    w.publishEvent(task, finalResult)

    return nil
}
```

### 3.2 编译阶段

```go
func (w *JudgeWorker) compile(task *JudgeTask, workspace string) (*CompileResult, error) {
    // 语言配置
    lang := task.Language

    // 空编译命令 = 解释型语言
    if len(lang.CompileCmd) == 0 {
        return &CompileResult{Success: true}, nil
    }

    // 构建编译命令
    cmd := exec.CommandContext(w.ctx, lang.CompileCmd[0], lang.CompileCmd[1:]...)
    cmd.Dir = workspace
    cmd.Env = []string{
        "PATH=/usr/local/bin:/usr/bin:/bin",
        "HOME=/tmp",
    }

    // 执行编译
    out, err := cmd.CombinedOutput()

    // 记录日志
    w.uploadLog(task.SubmitID, "compile.log", out)

    if err != nil {
        return &CompileResult{
            Success:   false,
            Error:     string(out),
            LogURL:    w.getLogURL(task.SubmitID, "compile.log"),
        }, err
    }

    return &CompileResult{Success: true}, nil
}
```

### 3.3 运行测试

```go
func (w *JudgeWorker) runTestCases(task *JudgeTask, workspace string) ([]TestResult, error) {
    results := make([]TestResult, 0, len(task.Problem.TestCases))

    for i, tc := range task.Problem.TestCases {
        // 读取输入
        input := tc.Input
        if task.Problem.TestDataZip != "" {
            input = w.extractTestCase(workspace, i)
        }

        // 构建运行命令
        runCmd := w.buildRunCmd(task.Language, workspace)

        // 计算超时 (基础超时 × 语言因子)
        timeout := time.Duration(task.Problem.TimeLimit) * time.Millisecond
        timeout = timeout * time.Duration(task.Language.TimeFactor*100)/100

        // 运行
        start := time.Now()
        output, err := w.runWithLimits(runCmd, input, timeout, task.Problem.MemoryLimit)
        elapsed := time.Since(start)

        // 比对
        status := w.compare(tc.ExpectedOutput, output, task.Problem.IsSPJ)

        results = append(results, TestResult{
            ID:        i + 1,
            Status:    status,
            TimeMs:    int(elapsed.Milliseconds()),
            MemoryKB:  0, // 从 cgroup 获取
            Score:     tc.Score,
        })
    }

    return results, nil
}
```

---

## 四、幂等更新策略

### 4.1 双重检查

```go
func (w *JudgeWorker) isAlreadyFinished(submitID string) bool {
    var sub Submission
    err := w.db.Where("submit_id = ?", submitID).First(&sub).Error
    if err == gorm.ErrRecordNotFound {
        return false
    }

    // 已完成或正在运行中
    result := sub.JudgeResult
    if result["status"] == "FINISHED" || result["status"] == "RUNNING" {
        return true
    }

    // RUNNING 但超过5分钟，视为超时，可以重新处理
    if result["status"] == "RUNNING" {
        if startTime, ok := result["start_time"].(time.Time); ok {
            if time.Since(startTime) > 5*time.Minute {
                return false // 允许重新处理
            }
        }
        return true
    }

    return false
}
```

### 4.2 原子更新

```go
func (w *JudgeWorker) updateStatus(submitID, status, workerID string) error {
    // 使用条件更新，防止覆盖
    result := map[string]interface{}{
        "status":     status,
        "worker_id":  workerID,
        "start_time": time.Now(),
    }

    err := w.db.Model(&Submission{}).
        Where("submit_id = ? AND status IN (?, ?)", submitID, "PENDING", "RUNNING").
        Updates(result).Error

    if err == gorm.ErrRecordNotFound {
        return fmt.Errorf("submit %s not in PENDING/RUNNING state", submitID)
    }

    return err
}
```

### 4.3 乐观锁

```go
type Submission struct {
    // ... fields
    Version int `gorm:"version"` // 乐观锁
}

// 更新时自动检查 version
err := w.db.Model(&sub).Updates(map[string]interface{}{
    "judge_result": resultJSON,
    "version":      sub.Version + 1,
})
// GORM 会自动 WHERE version = oldVersion
```

---

## 五、超时回收

### 5.1 Worker 侧超时

```go
func (w *JudgeWorker) runWithLimits(cmd *exec.Cmd, input []byte, timeout time.Duration, memoryLimit int) ([]byte, error) {
    // 使用 ctx 超时
    ctx, cancel := context.WithTimeout(w.ctx, timeout)
    defer cancel()

    cmd.Stdin = bytes.NewReader(input)

    // 使用 runc/Docker 限制资源
    // ... cgroup 设置

    output, err := cmd.Output()
    if ctx.Err() == context.DeadlineExceeded {
        return output, fmt.Errorf("time limit exceeded")
    }
    return output, err
}
```

### 5.2 外部超时回收 (Watchdog)

```go
// 独立 goroutine 定期检查超时任务
func (w *JudgeWorker) startWatchdog() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        // 查找 RUNNING 超过 5 分钟的任务
        var subs []Submission
        w.db.Where("status = ? AND start_time < ?", "RUNNING", time.Now().Add(-5*time.Minute)).
            Find(&subs)

        for _, sub := range subs {
            // 强制标记为 PENDING，重新入队
            w.db.Model(&sub).Updates(map[string]interface{}{
                "status":      "PENDING",
                "worker_id":   nil,
                "retry_count": sub.RetryCount + 1,
            })

            // 重新发布到队列
            w.requeue(sub.SubmitID)
        }
    }
}
```

---

## 六、重试与 DLQ

### 6.1 指数退避重试

```go
func (w *JudgeWorker) handleError(task *JudgeTask, err error) error {
    task.RetryCount++

    if task.RetryCount >= 3 {
        // 进入 DLQ
        w.sendToDLQ(task, err)
        w.updateStatus(task.SubmitID, "DLQ", "")
        return nil // 不返回 error，避免 NATS 重试
    }

    // 指数退避延迟重试
    delay := []time.Duration{5, 10, 30}[task.RetryCount-1] * time.Second
    time.Sleep(delay)

    // 重新入队
    w.requeue(task)
    return nil
}
```

### 6.2 DLQ 处理

```go
func (w *JudgeWorker) sendToDLQ(task *JudgeTask, err error) {
    dlqMsg := DLQMessage{
        OriginalSubject: "judge.tasks." + task.Type,
        SubmitID:        task.SubmitID,
        Error:           err.Error(),
        RetryCount:      task.RetryCount,
        FailedAt:        time.Now(),
        Payload:         task,
    }

    js.Publish("judge.dlq."+task.Type, dlqMsg)
}
```

---

## 七、日志上传

### 7.1 MinIO 存储结构

```
oj-logs/
├── submissions/
│   ├── {submit_id}/
│   │   ├── compile.log
│   │   ├── case_1.stdout
│   │   ├── case_1.stderr
│   │   ├── case_1_resource.json
│   │   └── result.json
│   └── ...
```

### 7.2 上传代码

```go
func (w *JudgeWorker) uploadLog(submitID, filename string, content []byte) error {
    path := fmt.Sprintf("submissions/%s/%s", submitID, filename)
    _, err := w.minio.PutObject("oj-logs", path, bytes.NewReader(content), int64(len(content)))
    return err
}
```

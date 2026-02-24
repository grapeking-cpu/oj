# 远程拉比赛 (Codeforces 镜像赛)

## 一、架构设计

```
┌─────────────────────────────────────────────────────────────────┐
│                      Codeforces 同步架构                         │
│                                                                  │
│  ┌─────────────┐    ┌─────────────┐    ┌──────────────────┐    │
│  │ Codeforces  │    │  Sync Job   │    │  本地 OJ         │    │
│  │    API      │───▶│ (定时/手动) │───▶│  比赛/榜单       │    │
│  └─────────────┘    └─────────────┘    └──────────────────┘    │
│         │                   │                     │              │
│         │           ┌───────▼───────┐            │              │
│         │           │ 题目同步      │            │              │
│         │           │ (可选)        │            │              │
│         │           └───────────────┘            │              │
└─────────────────────────────────────────────────────────────────┘
```

---

## 二、Codeforces API

### 2.1 常用接口

| 接口 | 说明 |
|------|------|
| `contest.list` | 获取比赛列表 |
| `contest.standings` | 获取比赛榜单 |
| `problemset.problems` | 获取题目列表 |
| `problemset.problems` | 获取题目详情 |
| `user.rating` | 获取用户 Rating 变化 |

### 2.2 API 封装

```go
type CFClient struct {
    APIKey    string
    APISecret string
    HTTPClient *http.Client
}

func (c *CFClient) Call(method string, params map[string]string) (json.RawMessage, error) {
    // 1. 构造参数
    params["apiKey"] = c.APIKey
    params["time"] = strconv.FormatInt(time.Now().Unix(), 10)

    // 2. 生成签名
    randStr := fmt.Sprintf("%6d", rand.Intn(1000000))
    paramStr := ""
    for k, v := range params {
        paramStr += fmt.Sprintf("%s=%s&", k, url.QueryEscape(v))
    }
    paramStr += randStr

    hash := sha1.Hash(paramStr)
    params["apiSig"] = randStr + hash

    // 3. 发起请求
    url := "https://codeforces.com/api/" + method + "?" + paramStr
    resp, err := c.HTTPClient.Get(url)
    // ...
}
```

---

## 三、比赛同步

### 3.1 同步流程

```go
func (s *CFSyncService) SyncContest(cfContestID int) (*Contest, error) {
    // 1. 获取 CF 比赛信息
    cfContest, err := s.cfClient.GetContest(cfContestID)
    if err != nil {
        return nil, err
    }

    // 2. 创建本地比赛
    contest := &Contest{
        Title:        cfContest.Name,
        Type:         "ACM",
        Source:       "codeforces",
        CFContestID:  cfContestID,
        CFContestSlug: cfContest.Slug,
        StartTime:    time.Unix(int64(cfContest.StartTimeSeconds), 0),
        EndTime:      time.Unix(int64(cfContest.StartTimeSeconds+7200), 0), // 2小时
        FrozenMinutes: 30,
        IsPublic:     true,
        Status:       "upcoming",
    }

    // 3. 获取 CF 题目列表
    problems, err := s.cfClient.GetContestProblems(cfContestID)
    // 映射到本地题目

    // 4. 保存
    s.db.Create(contest)

    return contest, nil
}
```

### 3.2 定时同步

```go
// 每5分钟同步正在进行的CF比赛榜单
func (s *CFSyncService) StartRankSyncJob() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        // 获取正在进行的CF比赛
        contests, _ := s.cfClient.GetRunningContests()

        for _, cf := range contests {
            // 检查是否已在本地
            local, _ := s.db.GetContestByCFID(cf.ID)
            if local == nil {
                continue
            }

            // 同步榜单
            s.SyncStandings(local.ID, cf.ID)
        }
    }
}
```

---

## 四、榜单同步

### 4.1 获取 CF 榜单

```go
func (s *CFSyncService) SyncStandings(contestID, cfContestID int) error {
    // 分页获取榜单
    page := 1
    for {
        standings, err := s.cfClient.GetContestStandings(cfContestID, page, 100)
        if err != nil {
            return err
        }

        if len(standings.Rows) == 0 {
            break
        }

        // 处理每行数据
        for _, row := range standings.Rows {
            s.updateParticipant(contestID, row)
        }

        page++
    }

    // 更新本地榜单缓存
    s.cache.Delete(fmt.Sprintf("contest:rank:%d", contestID))

    return nil
}
```

### 4.2 选手映射

```go
func (s *CFSyncService) updateParticipant(contestID int, row CFStandingsRow) {
    // 查找本地用户 (通过 CF handle)
    user, _ := s.db.GetUserByCFHandle(row.Handle)

    if user == nil {
        // 提示用户绑定 CF 账号
        return
    }

    // 计算 ACM 分数
    solved := 0
    penalty := 0
    for i, prob := range row.ProblemResults {
        if prob.Points > 0 {
            solved++
            penalty += prob.Penalty
        }
    }

    // 更新参赛记录
    participant := &ContestParticipant{
        ContestID: int64(contestID),
        UserID:    user.ID,
        Rank:      row.Rank,
        Score:     solved,
        Penalty:   penalty,
    }

    s.db.UpsertContestParticipant(contestID, user.ID, participant)
}
```

---

## 五、虚拟参赛

### 5.1 虚拟比赛创建

```go
// 用户可选择某个历史CF比赛进行虚拟参赛
func (s *ContestService) CreateVirtualContest(cfContestID int, userID int64) (*Contest, error) {
    // 获取CF比赛信息
    cfContest, _ := s.cfClient.GetContest(cfContestID)

    contest := &Contest{
        Title:        cfContest.Name + " (Virtual)",
        Type:         "ACM",
        Source:       "codeforces",
        CFContestID:  cfContestID,
        IsVirtual:    true,
        VirtualUserID: userID,
        // 虚拟比赛从用户点击开始计时，2小时
        StartTime:    time.Now(),
        EndTime:      time.Now().Add(2*time.Hour),
    }

    s.db.Create(contest)
    return contest, nil
}
```

### 5.2 虚拟提交

- 虚拟比赛中提交不进入公共榜单
- 用户可查看自己提交
- 虚拟比赛结束后可转为练习模式

---

## 六、练习模式

### 6.1 赛后练习

```go
// 比赛结束后自动转为练习
func (s *ContestService) OnContestEnd(contestID int64) {
    contest, _ := s.db.GetContest(contestID)
    if contest.Source == "codeforces" {
        // 标记为练习模式
        s.db.Model(contest).Update("status", "practice")
    }
}
```

### 6.2 练习提交

- 练习模式下不参与排名
- 提交后立即评测并显示结果
- 与正式比赛提交隔离

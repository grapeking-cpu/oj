# OJ 评测平台目录结构

```
E:\bishe_oj_go\
├── backend/                          # 后端 Go 项目
│   ├── cmd/                         # 入口
│   │   ├── api/main.go             # API 主入口
│   │   ├── worker/main.go         # Worker 主入口
│   │   └── migrate/main.go         # 数据库迁移
│   │
│   ├── internal/                    # 内部包 (不可导出)
│   │   ├── config/                 # 配置加载
│   │   │   └── config.go
│   │   ├── handler/                # HTTP Handler
│   │   │   ├── user.go
│   │   │   ├── problem.go
│   │   │   ├── submit.go
│   │   │   ├── contest.go
│   │   │   └── websocket.go
│   │   ├── middleware/             # 中间件
│   │   │   ├── auth.go
│   │   │   ├── cors.go
│   │   │   └── ratelimit.go
│   │   ├── service/                # 业务逻辑
│   │   │   ├── user.go
│   │   │   ├── problem.go
│   │   │   ├── submit.go
│   │   │   ├── contest.go
│   │   │   ├── judge.go
│   │   │   ├── language.go
│   │   │   └── cf_sync.go
│   │   ├── repository/             # 数据访问层
│   │   │   ├── user.go
│   │   │   ├── problem.go
│   │   │   ├── submit.go
│   │   │   ├── contest.go
│   │   │   └── language.go
│   │   ├── model/                  # 数据模型
│   │   │   ├── user.go
│   │   │   ├── problem.go
│   │   │   ├── submission.go
│   │   │   ├── contest.go
│   │   │   └── language.go
│   │   ├── queue/                  # NATS 队列
│   │   │   ├── client.go
│   │   │   ├── publisher.go
│   │   │   └── consumer.go
│   │   ├── cache/                  # Redis 缓存
│   │   │   ├── redis.go
│   │   │   ├── session.go
│   │   │   └── rank.go
│   │   ├── storage/                # MinIO 存储
│   │   │   └── minio.go
│   │   ├── docker/                 # Docker Runner
│   │   │   └── runner.go
│   │   └── types/                   # 通用类型
│   │       ├── request.go
│   │       └── response.go
│   │
│   ├── pkg/                         # 可导出包
│   │   ├── utils/                  # 工具函数
│   │   │   ├── jwt.go
│   │   │   ├── hash.go
│   │   │   └── captcha.go
│   │   └── errors/                 # 错误定义
│   │       └── errors.go
│   │
│   ├── migrations/                 # 数据库迁移
│   │   └── 001_init.sql
│   │
│   ├── Dockerfile                   # API 镜像
│   ├── Dockerfile.worker            # Worker 镜像
│   ├── go.mod
│   └── go.sum
│
├── frontend/                       # 前端 React 项目
│   ├── public/
│   ├── src/
│   │   ├── api/                    # API 请求
│   │   │   ├── index.ts
│   │   │   └── types.ts
│   │   ├── components/             # 公共组件
│   │   ├── pages/                  # 页面
│   │   │   ├── Login/
│   │   │   ├── ProblemList/
│   │   │   ├── ProblemDetail/
│   │   │   ├── Submit/
│   │   │   ├── ContestList/
│   │   │   ├── ContestDetail/
│   │   │   └── User/
│   │   ├── hooks/                  # 自定义 Hooks
│   │   ├── context/                # React Context
│   │   ├── styles/                 # 样式
│   │   ├── utils/                  # 工具
│   │   ├── App.tsx
│   │   └── main.tsx
│   ├── package.json
│   ├── vite.config.ts
│   ├── tsconfig.json
│   └── Dockerfile
│
├── docs/                          # 文档
│   ├── database.sql
│   ├── api.md
│   ├── nats.md
│   ├── judge.md
│   ├── docker.md
│   ├── cache.md
│   ├── security.md
│   ├── cf_sync.md
│   ├── docker-compose.md
│   └── mvp.md
│
├── docker-compose.yml             # 部署配置
├── nginx.conf                     # Nginx 配置
└── README.md
```

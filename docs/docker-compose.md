# Docker Compose 部署方案

## 一、单机部署架构

```yaml
# docker-compose.yml
version: '3.8'

services:
  # =========================================
  # 数据库层
  # =========================================

  postgres:
    image: postgres:15-alpine
    container_name: oj-postgres
    environment:
      POSTGRES_USER: oj
      POSTGRES_PASSWORD: oj_password
      POSTGRES_DB: oj
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U oj"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: oj-redis
    command: redis-server --appendonly yes --maxmemory 512mb --maxmemory-policy allkeys-lru
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  # =========================================
  # 消息队列
  # =========================================

  nats:
    image: nats:2.10-alpine
    container_name: oj-nats
    command:
      - "--jetstream"
      - "--mem-store=100MB"  # 内存存储限制
      - "--max-store=200MB"
      - "--max-payload=1MB"
      - "--max-conections=1000"
    ports:
      - "4222:4222"
      - "8222:8222"  # 监控
    volumes:
      - nats_data:/data
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8222/healthz"]
      interval: 10s
      timeout: 5s
      retries: 5

  # =========================================
  # 对象存储
  # =========================================

  minio:
    image: minio/minio:latest
    container_name: oj-minio
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    command: server /data --console-address ":9001"
    volumes:
      - minio_data:/data
    ports:
      - "9000:9000"
      - "9001:9001"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 30s
      timeout: 20s
      retries: 3

  # =========================================
  # 后端 API
  # =========================================

  api:
    build:
      context: ./backend
      dockerfile: Dockerfile
    container_name: oj-api
    environment:
      - DATABASE_URL=postgres://oj:oj_password@postgres:5432/oj?sslmode=disable
      - REDIS_URL=redis://redis:6379
      - NATS_URL=nats://nats:4222
      - MINIO_ENDPOINT=minio:9000
      - MINIO_ACCESS_KEY=minioadmin
      - MINIO_SECRET_KEY=minioadmin
      - JWT_SECRET=your-jwt-secret-key
      - GIN_MODE=release
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      nats:
        condition: service_healthy
      minio:
        condition: service_healthy
    ports:
      - "8080:8080"
    volumes:
      - ./backend:/app
    restart: unless-stopped

  # =========================================
  # 评测 Worker (Light x2 + Heavy x1)
  # =========================================

  judge-light-1:
    build:
      context: ./backend
      dockerfile: Dockerfile.worker
    container_name: oj-judge-light-1
    environment:
      - WORKER_ID=judge-light-1
      - CONSUMER=judge.tasks.light
      - CONCURRENCY=2
      - NATS_URL=nats://nats:4222
      - DATABASE_URL=postgres://oj:oj_password@postgres:5432/oj?sslmode=disable
      - MINIO_ENDPOINT=minio:9000
      - MINIO_ACCESS_KEY=minioadmin
      - MINIO_SECRET_KEY=minioadmin
    depends_on:
      nats:
        condition: service_healthy
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    restart: unless-stopped

  judge-light-2:
    build:
      context: ./backend
      dockerfile: Dockerfile.worker
    container_name: oj-judge-light-2
    environment:
      - WORKER_ID=judge-light-2
      - CONSUMER=judge.tasks.light
      - CONCURRENCY=2
      - NATS_URL=nats://nats:4222
      - DATABASE_URL=postgres://oj:oj_password@postgres:5432/oj?sslmode=disable
      - MINIO_ENDPOINT=minio:9000
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    restart: unless-stopped

  judge-heavy-1:
    build:
      context: ./backend
      dockerfile: Dockerfile.worker
    container_name: oj-judge-heavy-1
    environment:
      - WORKER_ID=judge-heavy-1
      - CONSUMER=judge.tasks.heavy
      - CONCURRENCY=1
      - NATS_URL=nats://nats:4222
      - DATABASE_URL=postgres://oj:oj_password@postgres:5432/oj?sslmode=disable
      - MINIO_ENDPOINT=minio:9000
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    restart: unless-stopped

  # =========================================
  # 前端
  # =========================================

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    container_name: oj-frontend
    ports:
      - "80:80"
    depends_on:
      - api
    restart: unless-stopped

  # =========================================
  # Nginx (可选，反向代理)
  # =========================================

  nginx:
    image: nginx:alpine
    container_name: oj-nginx
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
    ports:
      - "80:80"
      - "443:443"
    depends_on:
      - api
      - frontend
    restart: unless-stopped

# =========================================
# 数据卷
# =========================================

volumes:
  postgres_data:
  redis_data:
  nats_data:
  minio_data:
```

---

## 二、Nginx 配置

```nginx
# nginx.conf
events {
    worker_connections 1024;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    upstream api {
        server api:8080;
    }

    server {
        listen 80;
        server_name localhost;

        # 前端静态文件
        location / {
            root /usr/share/nginx/html;
            index index.html;
            try_files $uri $uri/ /index.html;
        }

        # API 代理
        location /api {
            proxy_pass http://api;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        }

        # WebSocket 代理
        location /ws {
            proxy_pass http://api;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_read_timeout 86400;
        }

        # MinIO 代理 (可选)
        location /minio {
            proxy_pass http://minio:9000;
            proxy_set_header Host $host;
        }
    }
}
```

---

## 三、初始化脚本

```bash
#!/bin/bash
# init.sh

set -e

echo "Waiting for services..."

# 等待 PostgreSQL
until pg_isready -h postgres -U oj; do
    echo "Waiting for PostgreSQL..."
    sleep 2
done

echo "PostgreSQL ready!"

# 等待 Redis
until redis-cli -h redis ping; do
    echo "Waiting for Redis..."
    sleep 2
done

echo "Redis ready!"

# 运行数据库迁移
echo "Running migrations..."
cd /app
go run cmd/migrate/main.go

echo "Migration done!"

# 初始化 NATS Streams
echo "Setting up NATS..."
/usr/local/bin/nats -s nats://nats:4222 stream add JUDGE_TASKS \
    --subjects "judge.tasks.*" \
    --storage file \
    --max-bytes 100MB \
    --max-age 24h

/usr/local/bin/nats -s nats://nats:4222 stream add JUDGE_EVENTS \
    --subjects "judge.events.*" \
    --storage file \
    --max-bytes 50MB \
    --max-age 1h

/usr/local/bin/nats -s nats://nats:4222 stream add JUDGE_DLQ \
    --subjects "judge.dlq.*" \
    --storage file \
    --max-bytes 100MB \
    --max-age 7d

# 创建 Consumers
echo "Creating Consumers..."
/usr/local/bin/nats -s nats://nats:4222 consumer add JUDGE_TASKS judge-light \
    --subject judge.tasks.light \
    --deliver new \
    --max-ack-pending 10 \
    --ack-timeout 30s \
    --max-deliver 3

/usr/local/bin/nats -s nats://nats:4222 consumer add JUDGE_TASKS judge-heavy \
    --subject judge.tasks.heavy \
    --deliver new \
    --max-ack-pending 5 \
    --ack-timeout 120s \
    --max-deliver 3

echo "Done!"
```

---

## 四、启动命令

```bash
# 启动全部服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 查看特定服务
docker-compose logs -f judge-worker

# 扩缩容
docker-compose up -d --scale judge-light-1=2

# 停止
docker-compose down
```

---

## 五、性能调优

### 5.1 资源规划 (8核8G)

| 服务 | CPU | 内存 |
|------|-----|------|
| PostgreSQL | 2核 | 2GB |
| Redis | 1核 | 512MB |
| NATS | 1核 | 256MB |
| MinIO | 1核 | 512MB |
| API | 1核 | 512MB |
| Worker x3 | 2核 | 2GB |

### 5.2 Worker 调优

```yaml
# 初始配置 (最稳)
judge-light-1: CONCURRENCY=2
judge-light-2: CONCURRENCY=2
judge-heavy-1: CONCURRENCY=1
# 总并发: 5

# 想要更快 (需更多资源)
judge-light-1: CONCURRENCY=3
judge-light-2: CONCURRENCY=3
judge-heavy-1: CONCURRENCY=2
# 总并发: 8
```

### 5.3 分布式扩容

```yaml
# docker-compose.swarm.yml (Swarm 模式)
deploy:
  mode: replicated
  replicas: 2
  resources:
    limits:
      cpus: '2'
      memory: 2G
```

---

## 六、健康检查

```yaml
# docker-compose healthcheck 示例
api:
    healthcheck:
        test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/api/v1/health"]
        interval: 30s
        timeout: 10s
        retries: 3
        start_period: 40s
```

```go
// Backend Health Endpoint
func HealthCheck(c *gin.Context) {
    // 检查 DB
    db.SQL.Ping()

    // 检查 Redis
    redis.Ping()

    // 检查 NATS
    nc, _ := js.ConsumerInfo("JUDGE_TASKS", "judge-light")

    c.JSON(200, gin.H{
        "status": "ok",
        "db": "ok",
        "redis": "ok",
        "nats": nc != nil,
    })
}
```

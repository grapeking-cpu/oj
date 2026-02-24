-- =====================================================
-- OJ 评测平台数据库表结构 (PostgreSQL)
-- =====================================================

-- -----------------------------------------------------
-- 1. 用户相关表
-- -----------------------------------------------------

-- 用户表
CREATE TABLE users (
    id              BIGSERIAL PRIMARY KEY,
    username        VARCHAR(50) NOT NULL UNIQUE,
    email           VARCHAR(255) NOT NULL UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    nickname        VARCHAR(100),
    avatar          VARCHAR(500),           -- MinIO URL
    rating          INTEGER DEFAULT 0,      -- OJ rating
    submit_count    INTEGER DEFAULT 0,
    accept_count    INTEGER DEFAULT 0,
    role            VARCHAR(20) DEFAULT 'user', -- user/admin
    status          VARCHAR(20) DEFAULT 'active', -- active/baned
    ip_register     INET,
    ip_last_login   INET,
    last_login_at   TIMESTAMP,
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_rating ON users(rating DESC);

-- 用户 Session (Redis 也可做，DB 做持久化)
CREATE TABLE sessions (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       VARCHAR(255) NOT NULL UNIQUE,
    ip          INET,
    user_agent  VARCHAR(500),
    expires_at  TIMESTAMP NOT NULL,
    created_at  TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_sessions_token ON sessions(token);
CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);

-- -----------------------------------------------------
-- 2. 语言注册表 (配置化，不写死代码)
-- -----------------------------------------------------

CREATE TABLE languages (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(50) NOT NULL,          -- C++17, Python3, Go1.22...
    slug            VARCHAR(30) NOT NULL UNIQUE,   -- cpp17, py311, go122...
    display_order   INTEGER DEFAULT 0,

    -- 文件配置
    source_filename VARCHAR(100) NOT NULL,          -- main.cpp, main.py...

    -- 编译命令 (空=解释型)
    compile_cmd     TEXT,                           -- ["g++", "-o", "main", "main.cpp"]
    compile_timeout INTEGER DEFAULT 10,             -- 秒

    -- 运行命令
    run_cmd         TEXT NOT NULL,                  -- ["./main"]
    run_timeout     INTEGER DEFAULT 2,               -- 秒 (基础超时)

    -- Docker 配置
    docker_image    VARCHAR(200) NOT NULL,          -- oj-runner-cpp17

    -- 资源因子 (Java/Python 需放大)
    time_factor     DECIMAL(3,2) DEFAULT 1.0,        -- 运行时 × time_factor
    memory_factor   DECIMAL(3,2) DEFAULT 1.0,        -- 内存 MB × memory_factor

    -- 限制
    output_limit    INTEGER DEFAULT 65536,          -- bytes
    pids_limit      INTEGER DEFAULT 64,

    -- 状态
    enabled         BOOLEAN DEFAULT TRUE,
    is_spj         BOOLEAN DEFAULT FALSE,           -- 是否支持 Special Judge

    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_languages_slug ON languages(slug);
CREATE INDEX idx_languages_enabled ON languages(enabled);

-- -----------------------------------------------------
-- 3. 题库表
-- -----------------------------------------------------

-- 题目表
CREATE TABLE problems (
    id              BIGSERIAL PRIMARY KEY,
    title           VARCHAR(200) NOT NULL,
    slug            VARCHAR(100) UNIQUE,            -- URL友好

    -- 难度与分类
    difficulty      INTEGER DEFAULT 3,               -- 1-5
    tags            VARCHAR(500)[],                  -- JSON array
    source          VARCHAR(200),                   -- 来源

    -- 内容 (Markdown)
    description     TEXT,                           -- 题面
    input_format    TEXT,
    output_format   TEXT,
    sample_io       JSONB,                          -- [{"input": "...", "output": "..."}]
    hint            TEXT,

    -- 评测配置
    time_limit      INTEGER DEFAULT 1000,            -- ms
    memory_limit    INTEGER DEFAULT 256,             -- MB
    stack_limit     INTEGER DEFAULT 64,              -- MB
    is_spj          BOOLEAN DEFAULT FALSE,
    spj_lang        VARCHAR(30),                    -- 支持SPJ的语言
    spj_code        TEXT,                           -- SPJ代码 (MinIO存也可)
    spj_compile_out TEXT,                           -- 编译产物URL

    -- 测试数据
    test_cases      JSONB,                          -- [{"input": "...", "output": "...", "score": 10}]
    test_data_zip   VARCHAR(500),                   -- MinIO URL
    test_data_hash  VARCHAR(64),                    -- SHA256 校验

    -- 统计
    submit_count    INTEGER DEFAULT 0,
    accept_count    INTEGER DEFAULT 0,
    accept_rate     DECIMAL(5,4) DEFAULT 0,

    -- 开放设置
    is_public       BOOLEAN DEFAULT FALSE,
    visible         BOOLEAN DEFAULT TRUE,

    -- 关联
    created_by      BIGINT REFERENCES users(id),
    updated_by      BIGINT REFERENCES users(id),

    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_problems_slug ON problems(slug);
CREATE INDEX idx_problems_difficulty ON problems(difficulty);
CREATE INDEX idx_problems_is_public ON problems(is_public);
CREATE INDEX idx_problems_created_by ON problems(created_by);

-- 题目标签
CREATE TABLE problem_tags (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(50) NOT NULL UNIQUE,
    color       VARCHAR(20),
    created_at  TIMESTAMP DEFAULT NOW()
);

-- -----------------------------------------------------
-- 4. 提交记录表 (核心高频)
-- -----------------------------------------------------

CREATE TABLE submissions (
    id              BIGSERIAL PRIMARY KEY,

    -- 唯一约束 (幂等)
    submit_id       VARCHAR(36) NOT NULL UNIQUE,    -- UUID，客户端可生成

    -- 关联
    user_id         BIGINT NOT NULL REFERENCES users(id),
    problem_id      BIGINT NOT NULL REFERENCES problems(id),
    contest_id      BIGINT,                         -- NULL=自由练习
    language_id     BIGINT NOT NULL REFERENCES languages(id),

    -- 代码
    code            TEXT NOT NULL,
    code_length     INTEGER NOT NULL,
    code_hash       VARCHAR(64),                    -- 查重用

    -- 编译 (SPJ/多文件)
    compile_info    TEXT,                           -- 编译错误信息
    compile_log_url VARCHAR(500),                   -- MinIO

    -- 评测结果 (JSONB 存储多测试点)
    judge_result    JSONB,                          -- 见下文状态机
    -- {
    --   "status": "PENDING|RUNNING|FINISHED|system_error|dlq",
    --   "score": 100,
    --   "total_test": 10,
    --   "accepted_test": 10,
    --   "time_ms": 125,
    --   "memory_kb": 35840,
    --   "cases": [
    --     {"id": 1, "status": "AC", "time_ms": 10, "memory_kb": 1024, "score": 10}
    --   ],
    --   "error": "...",
    --   "retry_count": 0
    -- }

    -- 运行时信息
    worker_id       VARCHAR(50),                    -- 评测机ID
    start_time      TIMESTAMP,
    finish_time     TIMESTAMP,

    -- 比赛相关
    is_contest      BOOLEAN DEFAULT FALSE,
    contest_rank    INTEGER,                        -- 比赛内排名
    frozen_score    JSONB,                          -- 封榜时的分数

    -- 重试幂等
    idempotency_key VARCHAR(100) UNIQUE,            -- 去重

    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);

-- 核心索引 (查询高频)
CREATE INDEX idx_submissions_user ON submissions(user_id);
CREATE INDEX idx_submissions_problem ON submissions(problem_id);
CREATE INDEX idx_submissions_contest ON submissions(contest_id);
CREATE INDEX idx_submissions_language ON submissions(language_id);
CREATE INDEX idx_submissions_status ON submissions((judge_result->>'status'));
CREATE INDEX idx_submissions_created ON submissions(created_at DESC);
CREATE UNIQUE INDEX idx_submissions_submit_id ON submissions(submit_id);

-- -----------------------------------------------------
-- 5. 比赛系统
-- -----------------------------------------------------

CREATE TABLE contests (
    id              BIGSERIAL PRIMARY KEY,
    title           VARCHAR(200) NOT NULL,
    slug            VARCHAR(100) UNIQUE,

    -- 比赛类型
    type            VARCHAR(20) NOT NULL,            -- ACM/IOI/OI
    source          VARCHAR(50),                    -- local/codeforces

    -- 时间
    start_time      TIMESTAMP NOT NULL,
    end_time        TIMESTAMP NOT NULL,
    frozen_minutes  INTEGER DEFAULT 0,              -- 封榜时间(分钟)
    freeze_board    BOOLEAN DEFAULT TRUE,            -- 是否启用封榜

    -- 赛制配置
    rule_type       VARCHAR(20) DEFAULT 'ACM',       -- ACM/IOI
    penalty_minutes INTEGER DEFAULT 20,              -- ACM 罚时

    -- 虚拟比赛
    is_virtual      BOOLEAN DEFAULT FALSE,
    virtual_user_id BIGINT,                         -- 虚拟参赛者ID

    -- Codeforces 同步
    cf_contest_id   INTEGER,                        -- CF比赛ID
    cf_contest_slug VARCHAR(100),

    -- 题目列表
    problems        JSONB,                          -- [{"pid": 1, "letter": "A"}, ...]

    -- 权限
    password        VARCHAR(100),                   -- 访问密码
    is_public       BOOLEAN DEFAULT TRUE,

    -- 状态
    status          VARCHAR(20) DEFAULT 'upcoming', -- upcoming/running/ended

    -- 管理
    created_by      BIGINT REFERENCES users(id),

    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_contests_slug ON contests(slug);
CREATE INDEX idx_contests_status ON contests(status);
CREATE INDEX idx_contests_start_time ON contests(start_time);

-- 比赛报名
CREATE TABLE contest_participants (
    id              BIGSERIAL PRIMARY KEY,
    contest_id      BIGINT NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- 比赛内信息
    rank            INTEGER,                        -- 当前排名
    score           INTEGER DEFAULT 0,             -- 总分
    penalty         INTEGER DEFAULT 0,             -- 总罚时

    -- 虚拟参赛
    virtual_start   TIMESTAMP,                      -- 虚拟开始时间

    -- 虚拟参赛独立记录
    is_virtual      BOOLEAN DEFAULT FALSE,

    UNIQUE(contest_id, user_id, COALESCE(virtual_start, '1970-01-01'))
);

CREATE INDEX idx_contest_participants_contest ON contest_participants(contest_id);
CREATE INDEX idx_contest_participants_user ON contest_participants(user_id);
CREATE INDEX idx_contest_participants_rank ON contest_participants(contest_id, rank);

-- -----------------------------------------------------
-- 6. 排行榜缓存 (Redis 为主，DB 做持久化)
-- -----------------------------------------------------

CREATE TABLE rating_history (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id),
    contest_id      BIGINT REFERENCES contests(id),

    rating_before   INTEGER,
    rating_after    INTEGER,
    delta           INTEGER,

    created_at      TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_rating_history_user ON rating_history(user_id);
CREATE INDEX idx_rating_history_contest ON rating_history(contest_id);

-- -----------------------------------------------------
-- 7. 系统配置 (Key-Value)
-- -----------------------------------------------------

CREATE TABLE system_configs (
    id              BIGSERIAL PRIMARY KEY,
    key             VARCHAR(100) NOT NULL UNIQUE,
    value           TEXT,
    description     VARCHAR(500),
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);

-- 常用配置: site.title, site.icp, judge.max_parallel, judge.timeout_seconds

-- -----------------------------------------------------
-- 8. 操作日志 (审计)
-- -----------------------------------------------------

CREATE TABLE operation_logs (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT REFERENCES users(id),
    action          VARCHAR(100) NOT NULL,
    target_type     VARCHAR(50),
    target_id       BIGINT,
    detail          JSONB,
    ip              INET,
    user_agent      VARCHAR(500),
    created_at      TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_operation_logs_user ON operation_logs(user_id);
CREATE INDEX idx_operation_logs_action ON operation_logs(action);
CREATE INDEX idx_operation_logs_created ON operation_logs(created_at DESC);

-- -----------------------------------------------------
-- 9. 验证码/限流
-- -----------------------------------------------------

CREATE TABLE captcha (
    id              BIGSERIAL PRIMARY KEY,
    key             VARCHAR(100) NOT NULL UNIQUE,
    code            VARCHAR(10) NOT NULL,
    expires_at      TIMESTAMP NOT NULL,
    created_at      TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_captcha_key ON captcha(key);
CREATE INDEX idx_captcha_expires ON captcha(expires_at);

-- =====================================================
-- 扩展: 题目收藏、题单、举报等功能按需扩展
-- =====================================================

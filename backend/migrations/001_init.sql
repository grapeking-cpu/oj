-- 初始化数据库表结构和默认数据

-- 启用 UUID 扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    nickname VARCHAR(100),
    avatar VARCHAR(500),
    rating INTEGER DEFAULT 0,
    submit_count INTEGER DEFAULT 0,
    accept_count INTEGER DEFAULT 0,
    role VARCHAR(20) DEFAULT 'user',
    status VARCHAR(20) DEFAULT 'active',
    ip_register INET,
    ip_last_login INET,
    last_login_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_rating ON users(rating DESC);

-- 语言表 (Language Registry)
CREATE TABLE IF NOT EXISTS languages (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    slug VARCHAR(30) NOT NULL UNIQUE,
    display_order INTEGER DEFAULT 0,
    source_filename VARCHAR(100) NOT NULL,
    compile_cmd TEXT,
    compile_timeout INTEGER DEFAULT 10,
    run_cmd TEXT NOT NULL,
    run_timeout INTEGER DEFAULT 2,
    docker_image VARCHAR(200) NOT NULL,
    time_factor DECIMAL(3,2) DEFAULT 1.0,
    memory_factor DECIMAL(3,2) DEFAULT 1.0,
    output_limit INTEGER DEFAULT 65536,
    pids_limit INTEGER DEFAULT 64,
    enabled BOOLEAN DEFAULT TRUE,
    is_spj BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_languages_slug ON languages(slug);
CREATE INDEX IF NOT EXISTS idx_languages_enabled ON languages(enabled);

-- 题目表
CREATE TABLE IF NOT EXISTS problems (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    slug VARCHAR(100) UNIQUE,
    difficulty INTEGER DEFAULT 3,
    tags TEXT[],
    source VARCHAR(200),
    description TEXT,
    input_format TEXT,
    output_format TEXT,
    sample_io JSONB,
    hint TEXT,
    time_limit INTEGER DEFAULT 1000,
    memory_limit INTEGER DEFAULT 256,
    stack_limit INTEGER DEFAULT 64,
    is_spj BOOLEAN DEFAULT FALSE,
    spj_lang VARCHAR(30),
    spj_code TEXT,
    spj_compile_out TEXT,
    test_cases JSONB,
    test_data_zip VARCHAR(500),
    test_data_hash VARCHAR(64),
    submit_count INTEGER DEFAULT 0,
    accept_count INTEGER DEFAULT 0,
    accept_rate DECIMAL(5,4) DEFAULT 0,
    is_public BOOLEAN DEFAULT FALSE,
    visible BOOLEAN DEFAULT TRUE,
    created_by BIGINT REFERENCES users(id),
    updated_by BIGINT REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_problems_slug ON problems(slug);
CREATE INDEX IF NOT EXISTS idx_problems_difficulty ON problems(difficulty);
CREATE INDEX IF NOT EXISTS idx_problems_is_public ON problems(is_public);
CREATE INDEX IF NOT EXISTS idx_problems_created_by ON problems(created_by);

-- 提交记录表
CREATE TABLE IF NOT EXISTS submissions (
    id BIGSERIAL PRIMARY KEY,
    submit_id VARCHAR(36) NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES users(id),
    problem_id BIGINT NOT NULL REFERENCES problems(id),
    contest_id BIGINT,
    language_id BIGINT NOT NULL REFERENCES languages(id),
    code TEXT NOT NULL,
    code_length INTEGER NOT NULL,
    code_hash VARCHAR(64),
    compile_info TEXT,
    compile_log_url VARCHAR(500),
    judge_result JSONB,
    worker_id VARCHAR(50),
    start_time TIMESTAMP,
    finish_time TIMESTAMP,
    is_contest BOOLEAN DEFAULT FALSE,
    contest_rank INTEGER,
    frozen_score JSONB,
    idempotency_key VARCHAR(100) UNIQUE,
    retry_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_submissions_user ON submissions(user_id);
CREATE INDEX IF NOT EXISTS idx_submissions_problem ON submissions(problem_id);
CREATE INDEX IF NOT EXISTS idx_submissions_contest ON submissions(contest_id);
CREATE INDEX IF NOT EXISTS idx_submissions_language ON submissions(language_id);
CREATE INDEX IF NOT EXISTS idx_submissions_created ON submissions(created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_submissions_submit_id ON submissions(submit_id);

-- 比赛表
CREATE TABLE IF NOT EXISTS contests (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    slug VARCHAR(100) UNIQUE,
    type VARCHAR(20) NOT NULL,
    source VARCHAR(50),
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    frozen_minutes INTEGER DEFAULT 0,
    freeze_board BOOLEAN DEFAULT TRUE,
    rule_type VARCHAR(20) DEFAULT 'ACM',
    penalty_minutes INTEGER DEFAULT 20,
    is_virtual BOOLEAN DEFAULT FALSE,
    virtual_user_id BIGINT,
    cf_contest_id INTEGER,
    cf_contest_slug VARCHAR(100),
    problems JSONB,
    password VARCHAR(100),
    is_public BOOLEAN DEFAULT TRUE,
    status VARCHAR(20) DEFAULT 'upcoming',
    created_by BIGINT REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_contests_slug ON contests(slug);
CREATE INDEX IF NOT EXISTS idx_contests_status ON contests(status);
CREATE INDEX IF NOT EXISTS idx_contests_start_time ON contests(start_time);

-- 比赛参与者表
CREATE TABLE IF NOT EXISTS contest_participants (
    id BIGSERIAL PRIMARY KEY,
    contest_id BIGINT NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rank INTEGER,
    score INTEGER DEFAULT 0,
    penalty INTEGER DEFAULT 0,
    virtual_start TIMESTAMP,
    is_virtual BOOLEAN DEFAULT FALSE,
    UNIQUE(contest_id, user_id, COALESCE(virtual_start, '1970-01-01'))
);

CREATE INDEX IF NOT EXISTS idx_contest_participants_contest ON contest_participants(contest_id);
CREATE INDEX IF NOT EXISTS idx_contest_participants_user ON contest_participants(user_id);
CREATE INDEX IF NOT EXISTS idx_contest_participants_rank ON contest_participants(contest_id, rank);

-- 验证码表
CREATE TABLE IF NOT EXISTS captchas (
    id BIGSERIAL PRIMARY KEY,
    key VARCHAR(100) NOT NULL UNIQUE,
    code VARCHAR(10) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_captchas_key ON captchas(key);
CREATE INDEX IF NOT EXISTS idx_captchas_expires ON captchas(expires_at);

-- 操作日志表
CREATE TABLE IF NOT EXISTS operation_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id),
    action VARCHAR(100) NOT NULL,
    target_type VARCHAR(50),
    target_id BIGINT,
    detail JSONB,
    ip INET,
    user_agent VARCHAR(500),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_operation_logs_user ON operation_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_operation_logs_action ON operation_logs(action);
CREATE INDEX IF NOT EXISTS idx_operation_logs_created ON operation_logs(created_at DESC);

-- 插入默认语言
INSERT INTO languages (name, slug, display_order, source_filename, compile_cmd, compile_timeout, run_cmd, run_timeout, docker_image, time_factor, memory_factor, output_limit, pids_limit, enabled) VALUES
('C++17', 'cpp17', 1, 'main.cpp', '["g++", "-o", "main", "main.cpp", "-std=c++17", "-O2", "-Wall"]', 10, '["/app/main"]', 10, 'oj-runner-cpp17', 1.0, 1.0, 65536, 64, true),
('C11', 'c11', 2, 'main.c', '["gcc", "-o", "main", "main.c", "-std=c11", "-O2", "-Wall"]', 10, '["/app/main"]', 10, 'oj-runner-c11', 1.0, 1.0, 65536, 64, true),
('Go 1.22', 'go122', 3, 'main.go', NULL, 0, '["go", "run", "main.go"]', 10, 'oj-runner-go122', 1.5, 1.0, 65536, 32, true),
('Python 3.11', 'py311', 4, 'main.py', NULL, 0, '["python3", "main.py"]', 15, 'oj-runner-py311', 3.0, 2.0, 65536, 32, true),
('Java 17', 'java17', 5, 'Main.java', '["javac", "Main.java"]', 30, '["java", "-cp", ".", "-Xmx256m", "Main"]', 30, 'oj-runner-java17', 2.0, 2.0, 65536, 64, true)
ON CONFLICT (slug) DO NOTHING;

-- 插入管理员账号 (密码: admin123)
INSERT INTO users (username, email, password_hash, nickname, role) VALUES
('admin', 'admin@oj.local', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'Administrator', 'admin')
ON CONFLICT (username) DO NOTHING;

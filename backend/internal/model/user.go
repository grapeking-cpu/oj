package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID           int64          `gorm:"primaryKey" json:"id"`
	Username     string         `gorm:"uniqueIndex;size:50" json:"username"`
	Email        string         `gorm:"uniqueIndex;size:255" json:"email"`
	PasswordHash string         `gorm:"size:255" json:"-"`
	Nickname     string         `gorm:"size:100" json:"nickname"`
	Avatar       string         `gorm:"size:500" json:"avatar"`
	Rating       int            `gorm:"default:0" json:"rating"`
	SubmitCount  int            `gorm:"default:0" json:"submit_count"`
	AcceptCount  int            `gorm:"default:0" json:"accept_count"`
	Role         string         `gorm:"size:20;default:user" json:"role"`
	Status       string         `gorm:"size:20;default:active" json:"status"`
	IPRegister   string         `gorm:"size:50" json:"ip_register"`
	IPLastLogin  string         `gorm:"size:50" json:"ip_last_login"`
	LastLoginAt  *time.Time     `json:"last_login_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// Session 会话模型
type Session struct {
	ID        int64          `gorm:"primaryKey"`
	UserID    int64          `gorm:"index" json:"user_id"`
	Token     string         `gorm:"uniqueIndex;size:255" json:"token"`
	IP        string         `gorm:"size:50" json:"ip"`
	UserAgent string         `gorm:"size:500" json:"user_agent"`
	ExpiresAt time.Time      `json:"expires_at"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Language 语言模型 (Language Registry)
type Language struct {
	ID             int64          `gorm:"primaryKey" json:"id"`
	Name           string         `gorm:"size:50" json:"name"`
	Slug           string         `gorm:"uniqueIndex;size:30" json:"slug"`
	DisplayOrder   int            `gorm:"default:0" json:"display_order"`
	SourceFilename string         `gorm:"size:100" json:"source_filename"`
	CompileCmd     string         `gorm:"type:text" json:"compile_cmd"` // JSON array
	CompileTimeout int            `gorm:"default:10" json:"compile_timeout"`
	RunCmd         string         `gorm:"type:text;not null" json:"run_cmd"`
	RunTimeout     int            `gorm:"default:2" json:"run_timeout"`
	DockerImage    string         `gorm:"size:200" json:"docker_image"`
	TimeFactor     float64        `gorm:"default:1.0" json:"time_factor"`
	MemoryFactor   float64        `gorm:"default:1.0" json:"memory_factor"`
	OutputLimit    int            `gorm:"default:65536" json:"output_limit"`
	PidsLimit      int            `gorm:"default:64" json:"pids_limit"`
	Enabled        bool           `gorm:"default:true" json:"enabled"`
	IsSPJ          bool           `gorm:"default:false" json:"is_spj"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// Problem 题目模型
type Problem struct {
	ID            int64          `gorm:"primaryKey" json:"id"`
	Title         string         `gorm:"size:200" json:"title"`
	Slug          string         `gorm:"uniqueIndex;size:100" json:"slug"`
	Difficulty    int            `gorm:"default:3" json:"difficulty"`
	Tags          StringArray    `gorm:"type:text" json:"tags"` // PostgreSQL array
	Source        string         `gorm:"size:200" json:"source"`
	Description   string         `gorm:"type:text" json:"description"`
	InputFormat   string         `gorm:"type:text" json:"input_format"`
	OutputFormat  string         `gorm:"type:text" json:"output_format"`
	SampleIO      string         `gorm:"type:jsonb" json:"sample_io"`
	Hint          string         `gorm:"type:text" json:"hint"`
	TimeLimit     int            `gorm:"default:1000" json:"time_limit"`
	MemoryLimit   int            `gorm:"default:256" json:"memory_limit"`
	StackLimit    int            `gorm:"default:64" json:"stack_limit"`
	IsSPJ         bool           `gorm:"default:false" json:"is_spj"`
	SpjLang       string         `gorm:"size:30" json:"spj_lang"`
	SpjCode       string         `gorm:"type:text" json:"spj_code"`
	SpjCompileOut string         `gorm:"size:500" json:"spj_compile_out"`
	TestCases     string         `gorm:"type:jsonb" json:"test_cases"`
	TestDataZip   string         `gorm:"size:500" json:"test_data_zip"`
	TestDataHash  string         `gorm:"size:64" json:"test_data_hash"`
	SubmitCount   int            `gorm:"default:0" json:"submit_count"`
	AcceptCount   int            `gorm:"default:0" json:"accept_count"`
	AcceptRate    float64        `gorm:"default:0" json:"accept_rate"`
	IsPublic      bool           `gorm:"default:false" json:"is_public"`
	Visible       bool           `gorm:"default:true" json:"visible"`
	CreatedBy     *int64         `json:"created_by"`
	UpdatedBy     *int64         `json:"updated_by"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// Submission 提交记录模型
type Submission struct {
	ID             int64          `gorm:"primaryKey" json:"id"`
	SubmitID       string         `gorm:"uniqueIndex;size:36" json:"submit_id"`
	UserID         int64          `gorm:"index" json:"user_id"`
	ProblemID      int64          `gorm:"index" json:"problem_id"`
	ContestID      *int64         `gorm:"index" json:"contest_id"`
	LanguageID     int64          `gorm:"index" json:"language_id"`
	Code           string         `gorm:"type:text" json:"code"`
	CodeLength     int            `json:"code_length"`
	CodeHash       string         `gorm:"size:64" json:"code_hash"`
	CompileInfo    string         `gorm:"type:text" json:"compile_info"`
	CompileLogURL  string         `gorm:"size:500" json:"compile_log_url"`
	JudgeResult    string         `gorm:"type:jsonb" json:"judge_result"`
	WorkerID       string         `gorm:"size:50" json:"worker_id"`
	StartTime      *time.Time     `json:"start_time"`
	FinishTime     *time.Time     `json:"finish_time"`
	IsContest      bool           `gorm:"default:false" json:"is_contest"`
	ContestRank    *int           `json:"contest_rank"`
	FrozenScore    string         `gorm:"type:jsonb" json:"frozen_score"`
	IdempotencyKey string         `gorm:"uniqueIndex;size:100" json:"idempotency_key"`
	RetryCount     int            `gorm:"default:0" json:"retry_count"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// Contest 比赛模型
type Contest struct {
	ID             int64          `gorm:"primaryKey" json:"id"`
	Title          string         `gorm:"size:200" json:"title"`
	Slug           string         `gorm:"uniqueIndex;size:100" json:"slug"`
	Type           string         `gorm:"size:20;not null" json:"type"` // ACM/IOI
	Source         string         `gorm:"size:50" json:"source"`        // local/codeforces
	StartTime      time.Time      `json:"start_time"`
	EndTime        time.Time      `json:"end_time"`
	FrozenMinutes  int            `gorm:"default:0" json:"frozen_minutes"`
	FreezeBoard    bool           `gorm:"default:true" json:"freeze_board"`
	RuleType       string         `gorm:"size:20;default:ACM" json:"rule_type"`
	PenaltyMinutes int            `gorm:"default:20" json:"penalty_minutes"`
	IsVirtual      bool           `gorm:"default:false" json:"is_virtual"`
	VirtualUserID  *int64         `json:"virtual_user_id"`
	CfContestID    *int           `gorm:"uniqueIndex" json:"cf_contest_id"`
	CfContestSlug  string         `gorm:"size:100" json:"cf_contest_slug"`
	Problems       string         `gorm:"type:jsonb" json:"problems"`
	Password       string         `gorm:"size:100" json:"password"`
	IsPublic       bool           `gorm:"default:true" json:"is_public"`
	Status         string         `gorm:"size:20;default:upcoming" json:"status"`
	CreatedBy      *int64         `json:"created_by"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// ContestParticipant 比赛参赛者
type ContestParticipant struct {
	ID           int64          `gorm:"primaryKey" json:"id"`
	ContestID    int64          `gorm:"index" json:"contest_id"`
	UserID       int64          `gorm:"index" json:"user_id"`
	Rank         int            `json:"rank"`
	Score        int            `gorm:"default:0" json:"score"`
	Penalty      int            `gorm:"default:0" json:"penalty"`
	VirtualStart *time.Time     `json:"virtual_start"`
	IsVirtual    bool           `gorm:"default:false" json:"is_virtual"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// StringArray PostgreSQL text[] 兼容类型
type StringArray []string

// Scan 实现 sql.Scanner 接口
func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, a)
	case string:
		return json.Unmarshal([]byte(v), a)
	default:
		return fmt.Errorf("cannot scan type %T into StringArray", value)
	}
}

// Value 实现 driver.Valuer 接口
func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	return json.Marshal(a)
}

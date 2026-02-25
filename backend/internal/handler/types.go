package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/oj/oj-backend/internal/model"
)

type ProblemInput struct {
	Title        string   `json:"title"`
	Slug         string   `json:"slug"`
	Difficulty   int      `json:"difficulty"`
	Tags         []string `json:"tags"`
	Source       string   `json:"source"`
	Description  string   `json:"description"`
	InputFormat  string   `json:"input_format"`
	OutputFormat string   `json:"output_format"`
	SampleIO     string   `json:"sample_io"`
	Hint         string   `json:"hint"`
	TimeLimit    int      `json:"time_limit"`
	MemoryLimit  int      `json:"memory_limit"`
	IsSPJ        bool     `json:"is_spj"`
	IsPublic     bool     `json:"is_public"`
}

func (p *ProblemInput) ToModel() *model.Problem {
	return &model.Problem{
		Title:        p.Title,
		Slug:         p.Slug,
		Difficulty:   p.Difficulty,
		Tags:         p.Tags,
		Source:       p.Source,
		Description:  p.Description,
		InputFormat:  p.InputFormat,
		OutputFormat: p.OutputFormat,
		SampleIO:     p.SampleIO,
		Hint:         p.Hint,
		TimeLimit:    p.TimeLimit,
		MemoryLimit:  p.MemoryLimit,
		IsSPJ:        p.IsSPJ,
		IsPublic:     p.IsPublic,
		Visible:      true,
	}
}

// Helper functions
func getInt(c *gin.Context, key string, defaultValue int) int {
	if v := c.Query(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultValue
}

func getInt64(c *gin.Context, key string, defaultValue int64) int64 {
	if v := c.Query(key); v != "" {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return defaultValue
}

func getInt64Ptr(c *gin.Context, key string) *int64 {
	if v := c.Query(key); v != "" {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return &i
		}
	}
	return nil
}

func getStringPtr(c *gin.Context, key string) *string {
	if v := c.Query(key); v != "" {
		return &v
	}
	return nil
}

func getInt64Param(c *gin.Context, key string) int64 {
	if v := c.Param(key); v != "" {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return 0
}

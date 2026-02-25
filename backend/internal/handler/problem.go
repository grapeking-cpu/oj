package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/oj/oj-backend/internal/repository"
	"github.com/oj/oj-backend/internal/service"
)

type ProblemHandler struct {
	service *service.ProblemService
}

func NewProblemHandler(s *service.ProblemService) *ProblemHandler {
	return &ProblemHandler{service: s}
}

func (h *ProblemHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid id"})
		return
	}

	problem, err := h.service.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": problem,
	})
}

func (h *ProblemHandler) List(c *gin.Context) {
	page := getInt(c, "page", 1)
	pageSize := getInt(c, "page_size", 20)
	difficulty := getInt(c, "difficulty", 0)
	tags := c.Query("tags")
	search := c.Query("search")

	params := repository.ListProblemParams{
		Page:       page,
		PageSize:   pageSize,
		Difficulty: difficulty,
		Tags:       tags,
		Search:     search,
	}

	problems, total, err := h.service.List(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  problems,
			"total": total,
		},
	})
}

func (h *ProblemHandler) Create(c *gin.Context) {
	userID := c.GetInt64("user_id")

	var problem ProblemInput
	if err := c.ShouldBindJSON(&problem); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if err := h.service.Create(problem.ToModel(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0})
}

func (h *ProblemHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid id"})
		return
	}

	userID := c.GetInt64("user_id")

	var problem ProblemInput
	if err := c.ShouldBindJSON(&problem); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if err := h.service.Update(id, problem.ToModel(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0})
}

func (h *ProblemHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid id"})
		return
	}

	if err := h.service.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0})
}

func (h *ProblemHandler) UploadTestData(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "not implemented"})
}

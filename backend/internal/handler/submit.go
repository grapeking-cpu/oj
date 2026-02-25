package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/oj/oj-backend/internal/service"
)

type SubmitHandler struct {
	service *service.SubmitService
}

func NewSubmitHandler(s *service.SubmitService) *SubmitHandler {
	return &SubmitHandler{service: s}
}

type SubmitRequest struct {
	ProblemID      int64  `json:"problem_id" binding:"required"`
	LanguageID     int64  `json:"language_id" binding:"required"`
	Code           string `json:"code" binding:"required"`
	IdempotencyKey string `json:"idempotency_key"`
}

func (h *SubmitHandler) Create(c *gin.Context) {
	userID := c.GetInt64("user_id")

	var req SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	submission, err := h.service.Create(userID, service.SubmitParams{
		ProblemID:      req.ProblemID,
		LanguageID:    req.LanguageID,
		Code:          req.Code,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"submit_id": submission.SubmitID,
			"status":    "PENDING",
		},
	})
}

func (h *SubmitHandler) Get(c *gin.Context) {
	submitID := c.Param("submit_id")

	submission, err := h.service.GetBySubmitID(submitID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": submission,
	})
}

func (h *SubmitHandler) List(c *gin.Context) {
	userID := c.GetInt64("user_id")
	problemID := getInt64Ptr(c, "problem_id")
	status := getStringPtr(c, "status")
	page := getInt(c, "page", 1)
	pageSize := getInt(c, "page_size", 20)

	submissions, total, err := h.service.ListByUser(userID, problemID, status, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  submissions,
			"total": total,
		},
	})
}

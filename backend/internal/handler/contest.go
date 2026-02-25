package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/oj/oj-backend/internal/repository"
	"github.com/oj/oj-backend/internal/service"
)

type ContestHandler struct {
	service *service.ContestService
}

func NewContestHandler(s *service.ContestService) *ContestHandler {
	return &ContestHandler{service: s}
}

func (h *ContestHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid id"})
		return
	}

	contest, err := h.service.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": contest,
	})
}

func (h *ContestHandler) List(c *gin.Context) {
	page := getInt(c, "page", 1)
	pageSize := getInt(c, "page_size", 20)
	contestType := c.Query("type")
	status := c.Query("status")

	params := repository.ListContestParams{
		Page:     page,
		PageSize: pageSize,
		Type:     contestType,
		Status:   status,
	}

	contests, total, err := h.service.List(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  contests,
			"total": total,
		},
	})
}

func (h *ContestHandler) GetRank(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid id"})
		return
	}

	rank, err := h.service.GetRank(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": rank,
	})
}

func (h *ContestHandler) Join(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid id"})
		return
	}

	userID := c.GetInt64("user_id")

	var req struct {
		Password string `json:"password"`
	}
	c.ShouldBindJSON(&req)

	if err := h.service.Join(id, userID, req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0})
}

func (h *ContestHandler) Create(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "not implemented"})
}

func (h *ContestHandler) Update(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "not implemented"})
}

func (h *ContestHandler) Delete(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "not implemented"})
}

package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/oj/oj-backend/internal/service"
)

type LanguageHandler struct {
	service *service.LanguageService
}

func NewLanguageHandler(s *service.LanguageService) *LanguageHandler {
	return &LanguageHandler{service: s}
}

func (h *LanguageHandler) List(c *gin.Context) {
	enabled := true

	languages, err := h.service.List(&enabled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": languages,
	})
}

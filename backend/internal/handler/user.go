package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/oj/oj-backend/internal/service"
)

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(s *service.UserService) *UserHandler {
	return &UserHandler{service: s}
}

type RegisterRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=50"`
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	CaptchaKey  string `json:"captcha_key" binding:"required"`
	CaptchaCode string `json:"captcha_code" binding:"required"`
}

func (h *UserHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	ip := c.ClientIP()

	user, token, err := h.service.Register(req.Username, req.Email, req.Password, req.CaptchaKey, req.CaptchaCode, ip)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"user_id": user.ID,
			"token":   token,
		},
	})
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *UserHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	ip := c.ClientIP()

	user, token, err := h.service.Login(req.Username, req.Password, ip)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"user_id": user.ID,
			"token":   token,
		},
	})
}

func (h *UserHandler) GetCaptcha(c *gin.Context) {
	key, image, err := h.service.GenerateCaptcha()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"captcha_key":   key,
			"captcha_image": image,
		},
	})
}

func (h *UserHandler) Info(c *gin.Context) {
	userID := c.GetInt64("user_id")

	user, err := h.service.GetUserInfo(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": user,
	})
}

type UpdateProfileRequest struct {
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetInt64("user_id")

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if err := h.service.UpdateProfile(userID, req.Nickname, req.Avatar); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0})
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID := c.GetInt64("user_id")

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if err := h.service.ChangePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0})
}

func (h *UserHandler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0})
}

func (h *UserHandler) List(c *gin.Context) {
	page := getInt(c, "page", 1)
	pageSize := getInt(c, "page_size", 20)
	role := c.Query("role")
	status := c.Query("status")

	users, total, err := h.service.ListUsers(page, pageSize, role, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  users,
			"total": total,
		},
	})
}

type BanRequest struct {
	Ban bool `json:"ban"`
}

func (h *UserHandler) Ban(c *gin.Context) {
	userID := getInt64Param(c, "id")

	var req BanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if err := h.service.BanUser(userID, req.Ban); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0})
}

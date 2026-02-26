package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/oj/oj-backend/internal/service"
)

// setAuthCookie 设置 HttpOnly Cookie
func (h *UserHandler) setAuthCookie(c *gin.Context, token string) {
	// 使用 Gin 的 SetCookie 方法
	// 参数: name, value, maxAge, path, domain, secure, httpOnly
	c.SetCookie(CookieTokenName, token, 86400, "/", "", false, true)
	// 覆盖使用正确的 SameSite 设置（不设置 SameSite，兼容开发环境）
	c.Header("Set-Cookie", CookieTokenName+"="+token+"; Path=/; HttpOnly; Max-Age=86400")
}

// clearAuthCookie 清除认证 Cookie
func (h *UserHandler) clearAuthCookie(c *gin.Context) {
	// 使用 Gin 的 SetCookie 方法清除 Cookie
	c.SetCookie(CookieTokenName, "", -1, "/", "", false, true)
	// 手动清除 SameSite 设置
	c.Header("Set-Cookie", CookieTokenName+"=; Path=/; HttpOnly; Max-Age=0")
}

const CookieTokenName = "token"

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(s *service.UserService) *UserHandler {
	return &UserHandler{service: s}
}

// 统一错误响应
func errorResponse(c *gin.Context, code int, message string) {
	// 不暴露具体错误细节
	c.JSON(code, gin.H{
		"code":    code,
		"message": message,
		"data":    nil,
	})
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
		errorResponse(c, http.StatusBadRequest, "invalid parameters")
		return
	}

	ip := c.ClientIP()

	user, token, err := h.service.Register(req.Username, req.Email, req.Password, req.CaptchaKey, req.CaptchaCode, ip)
	if err != nil {
		// 统一错误信息，不区分具体原因
		if strings.Contains(err.Error(), "invalid") {
			errorResponse(c, http.StatusBadRequest, "invalid parameters")
		} else if strings.Contains(err.Error(), "captcha") {
			errorResponse(c, http.StatusBadRequest, "invalid captcha")
		} else {
			errorResponse(c, http.StatusBadRequest, "registration failed")
		}
		return
	}

	// 设置 HttpOnly Cookie
	h.setAuthCookie(c, token)

	// 返回用户信息（不含敏感数据）
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"user_id":  user.ID,
			"username": user.Username,
			"nickname": user.Nickname,
			"role":     user.Role,
		},
		"message": "success",
	})
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *UserHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "invalid parameters")
		return
	}

	ip := c.ClientIP()

	user, token, err := h.service.Login(req.Username, req.Password, ip)
	if err != nil {
		// 统一错误信息
		if strings.Contains(err.Error(), "locked") || strings.Contains(err.Error(), "attempts") {
			errorResponse(c, http.StatusTooManyRequests, "too many attempts, please try again later")
		} else {
			errorResponse(c, http.StatusUnauthorized, "invalid credentials")
		}
		return
	}

	// 设置 HttpOnly Cookie
	h.setAuthCookie(c, token)

	// 返回用户信息
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"user_id":     user.ID,
			"username":    user.Username,
			"nickname":    user.Nickname,
			"role":        user.Role,
			"rating":      user.Rating,
			"submit_count": user.SubmitCount,
			"accept_count": user.AcceptCount,
		},
		"message": "success",
	})
}

func (h *UserHandler) GetCaptcha(c *gin.Context) {
	key, image, err := h.service.GenerateCaptcha()
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "internal error")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"captcha_key":   key,
			"captcha_image": image,
		},
		"message": "success",
	})
}

func (h *UserHandler) Info(c *gin.Context) {
	userID := c.GetInt64("user_id")

	user, err := h.service.GetUserInfo(userID)
	if err != nil {
		errorResponse(c, http.StatusNotFound, "user not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":           user.ID,
			"username":     user.Username,
			"email":        user.Email,
			"nickname":     user.Nickname,
			"avatar":       user.Avatar,
			"role":         user.Role,
			"rating":       user.Rating,
			"submit_count": user.SubmitCount,
			"accept_count": user.AcceptCount,
			"created_at":   user.CreatedAt,
			"last_login_at": user.LastLoginAt,
		},
		"message": "success",
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
	// 清除 Cookie
	h.clearAuthCookie(c)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
	})
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

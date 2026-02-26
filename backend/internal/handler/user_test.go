package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
		wantCode   int
	}{
		{
			name:       "missing username",
			body:       map[string]interface{}{"email": "test@example.com", "password": "password123"},
			wantStatus: http.StatusBadRequest,
			wantCode:   400,
		},
		{
			name:       "invalid email",
			body:       map[string]interface{}{"username": "testuser", "email": "invalid", "password": "password123"},
			wantStatus: http.StatusBadRequest,
			wantCode:   400,
		},
		{
			name:       "short password",
			body:       map[string]interface{}{"username": "testuser", "email": "test@example.com", "password": "123"},
			wantStatus: http.StatusBadRequest,
			wantCode:   400,
		},
		{
			name:       "missing captcha",
			body:       map[string]interface{}{"username": "testuser", "email": "test@example.com", "password": "password123", "captcha_key": ""},
			wantStatus: http.StatusBadRequest,
			wantCode:   400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			// 只测试绑定，不调用实际服务
			router.POST("/register", func(c *gin.Context) {
				var req RegisterRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid parameters"})
					return
				}
				c.JSON(http.StatusOK, gin.H{"code": 0})
			})

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestLoginValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name:       "missing username",
			body:       map[string]interface{}{"password": "password123"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing password",
			body:       map[string]interface{}{"username": "testuser"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty body",
			body:       map[string]interface{}{},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.POST("/login", func(c *gin.Context) {
				var req LoginRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid parameters"})
					return
				}
				c.JSON(http.StatusOK, gin.H{"code": 0})
			})

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestErrorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/error", func(c *gin.Context) {
		errorResponse(c, http.StatusUnauthorized, "invalid credentials")
	})

	req := httptest.NewRequest("GET", "/error", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want %d", w.Code, http.StatusUnauthorized)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	// code 可能是 float64
	code := int(resp["code"].(float64))
	if code != 401 {
		t.Errorf("got code %d, want 401", code)
	}
	if resp["message"] != "invalid credentials" {
		t.Errorf("got message %v, want 'invalid credentials'", resp["message"])
	}
}

package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Auth 从 Cookie 读取 JWT 进行鉴权（仅 HttpOnly Cookie）
func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 只从 Cookie 读取 token，不支持 Authorization header
		tokenString, err := c.Cookie("token")

		// 如果 Cookie 不存在，返回 unauthorized
		if err != nil || tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "unauthorized"})
			c.Abort()
			return
		}

		// 解析 token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		// token 解析失败，返回 invalid token
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid token claims"})
			c.Abort()
			return
		}

		// 提取用户信息
		userID, _ := claims["user_id"].(float64)
		username, _ := claims["username"].(string)
		role, _ := claims["role"].(string)

		c.Set("user_id", int64(userID))
		c.Set("username", username)
		c.Set("role", role)

		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "Role not found"})
			c.Abort()
			return
		}

		for _, r := range roles {
			if role == r {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "Insufficient permissions"})
		c.Abort()
	}
}

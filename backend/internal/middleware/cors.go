package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS 创建 CORS 中间件
// allowOrigins: 允许的 origins 列表，传入 nil 时使用默认值
func CORS(allowOrigins []string) gin.HandlerFunc {
	// 默认允许的 origins
	if len(allowOrigins) == 0 {
		allowOrigins = []string{"http://localhost:5173", "http://localhost:3000"}
	}

	return cors.New(cors.Config{
		AllowOrigins:     allowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}

package middleware

import (
	"github.com/gin-gonic/gin"
)

// CORS 创建 CORS 中间件
// allowOrigins: 允许的 origins 列表
// allowCredentials: 是否允许携带凭据（Cookie、Authorization header）
func CORS(allowOrigins []string, allowCredentials bool) gin.HandlerFunc {
	// 默认允许的 origins
	if len(allowOrigins) == 0 {
		allowOrigins = []string{
			"http://localhost:5173",
			"http://localhost:3000",
			"http://localhost",
			"http://127.0.0.1:5173",
			"http://127.0.0.1:3000",
			"http://127.0.0.1",
		}
	}

	// 返回自定义中间件，确保所有响应都有 CORS 头
	return func(c *gin.Context) {
		// 设置 CORS 响应头
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			allowed := false
			for _, allowedOrigin := range allowOrigins {
				if allowedOrigin == origin || allowedOrigin == "*" {
					allowed = true
					break
				}
			}
			if allowed {
				c.Header("Access-Control-Allow-Origin", origin)
				if allowCredentials {
					c.Header("Access-Control-Allow-Credentials", "true")
				}
				c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")
				c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type")
			}
		}

		// 处理 OPTIONS 预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

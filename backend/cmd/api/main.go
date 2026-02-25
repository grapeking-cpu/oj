package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/oj/oj-backend/internal/config"
	"github.com/oj/oj-backend/internal/handler"
	"github.com/oj/oj-backend/internal/middleware"
	"github.com/oj/oj-backend/internal/repository"
	"github.com/oj/oj-backend/internal/service"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化数据库
	db, err := config.InitDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}

	// 初始化 Redis
	rdb := config.InitRedis(cfg.RedisURL)

	// 初始化 NATS
	_, js := config.InitNATS(cfg.NATSURL)

	// 初始化 MinIO
	minioClient := config.InitMinIO(cfg.MinIOEndpoint, cfg.MinIOAccessKey, cfg.MinIOSecretKey)

	// 确保 bucket 存在
	if err := config.InitMinIOBucket(minioClient, cfg.MinIOBucket); err != nil {
		log.Printf("Warning: Failed to ensure MinIO bucket: %v", err)
	}

	// 初始化 Repository
	repos := repository.NewRepositories(db)

	// 初始化 Service
	services := service.NewServices(repos, rdb, js, minioClient, cfg.JWTSecret)

	// 初始化 Handler
	handlers := handler.NewHandlers(services)

	// 初始化 WebSocket Hub
	wsHub := handler.NewWSHub()

	// 启动 WebSocket 广播协程
	go wsHub.Run()

	// Gin 路由
	r := gin.Default()

	// 中间件
	r.Use(middleware.CORS(cfg.AllowOrigins))
	r.Use(middleware.Logger())

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API 路由
	api := r.Group("/api/v1")
	{
		// 公开接口
		api.GET("/languages", handlers.Lang.List)

		// 用户相关
		api.POST("/user/register", handlers.User.Register)
		api.POST("/user/login", handlers.User.Login)
		api.GET("/user/captcha", handlers.User.GetCaptcha)

		// 题目相关
		api.GET("/problems", handlers.Problem.List)
		api.GET("/problems/:id", handlers.Problem.Get)

		// 比赛相关
		api.GET("/contests", handlers.Contest.List)
		api.GET("/contests/:id", handlers.Contest.Get)
		api.GET("/contests/:id/rank", handlers.Contest.GetRank)

		// 需要认证的接口
		auth := api.Group("")
		auth.Use(middleware.Auth(cfg.JWTSecret))
		{
			// 用户
			auth.GET("/user/info", handlers.User.Info)
			auth.PUT("/user/profile", handlers.User.UpdateProfile)
			auth.PUT("/user/password", handlers.User.ChangePassword)
			auth.POST("/user/logout", handlers.User.Logout)

			// 提交
			auth.POST("/submit", handlers.Submit.Create)
			auth.GET("/submit/:submit_id", handlers.Submit.Get)
			auth.GET("/my/submits", handlers.Submit.List)

			// 比赛
			auth.POST("/contests/:id/join", handlers.Contest.Join)
			auth.POST("/contests/:id/submit", handlers.Submit.CreateContest)
		}

		// 管理员接口
		admin := auth.Group("")
		admin.Use(middleware.RequireRole("admin"))
		{
			// 题目管理
			admin.POST("/problems", handlers.Problem.Create)
			admin.PUT("/problems/:id", handlers.Problem.Update)
			admin.DELETE("/problems/:id", handlers.Problem.Delete)
			admin.POST("/problems/:id/testdata", handlers.Problem.UploadTestData)

			// 比赛管理
			admin.POST("/contests", handlers.Contest.Create)
			admin.PUT("/contests/:id", handlers.Contest.Update)
			admin.DELETE("/contests/:id", handlers.Contest.Delete)

			// 用户管理
			admin.GET("/admin/users", handlers.User.List)
			admin.POST("/admin/users/:id/ban", handlers.User.Ban)
		}

		// WebSocket
		api.GET("/ws", handler.HandleWebSocket(wsHub))
	}

	// 启动服务
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on port %s", port)
	r.Run(":" + port)
}

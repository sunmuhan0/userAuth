package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ttuser/config-server/internal/service"
)

const (
	defaultPort  = 7963
	defaultToken = "ttuser-config-token-2024"
)

// HTTPServer 配置中心HTTP服务
type HTTPServer struct {
	ConfigService *service.ConfigService `inject:"configService"`
	engine        *gin.Engine
	httpServer    *http.Server
}

// Start 实现 inji.Startable
func (s *HTTPServer) Start() error {
	s.engine = gin.Default()
	s.setupRoutes()

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", defaultPort),
		Handler:      s.engine,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Printf("[config-server] listening on :%d\n", defaultPort)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("[config-server] listen error: %v\n", err)
		}
	}()
	return nil
}

// Shutdown 优雅关闭HTTP服务
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

func (s *HTTPServer) setupRoutes() {
	s.engine.Use(s.authMiddleware())

	// GET /config/files?env=prod&service=auth-server — 获取该环境下该服务的所有配置文件
	s.engine.GET("/config/files", s.listFiles)
}

// authMiddleware 静态token认证
func (s *HTTPServer) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token != "Bearer "+defaultToken {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// listFiles GET /config/files?env=prod&service=auth-server
func (s *HTTPServer) listFiles(c *gin.Context) {
	env := c.Query("env")
	svc := c.Query("service")
	if env == "" || svc == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "env and service are required"})
		return
	}

	files, err := s.ConfigService.ListFiles(env, svc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"files": files}})
}

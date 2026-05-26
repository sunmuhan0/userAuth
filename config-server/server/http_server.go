package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"ttuser/config-server/internal/service"
)

const (
	defaultPort  = 7963
	defaultToken = "ttuser-config-token-2024" // 静态认证token，后期可改为更安全的方式
)

// HTTPServer 配置中心HTTP服务
type HTTPServer struct {
	ConfigService *service.ConfigService `inject:"configService"`
	engine        *gin.Engine
}

// Start 实现 inji.Startable
func (s *HTTPServer) Start() error {
	s.engine = gin.Default()
	s.setupRoutes()
	fmt.Printf("[config-server] listening on :%d\n", defaultPort)
	go s.engine.Run(fmt.Sprintf(":%d", defaultPort))
	return nil
}

func (s *HTTPServer) setupRoutes() {
	s.engine.Use(s.authMiddleware())

	// GET /config?service=xxx&key=xxx — 获取单个配置
	s.engine.GET("/config", s.getConfig)

	// GET /configs?service=xxx — 获取服务所有配置
	s.engine.GET("/configs", s.listConfigs)

	// POST /config — 设置配置（明文）
	s.engine.POST("/config", s.setConfig)

	// POST /config/encrypted — 设置配置（加密存储）
	s.engine.POST("/config/encrypted", s.setEncryptedConfig)
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

// getConfig GET /config?service=xxx&key=xxx
func (s *HTTPServer) getConfig(c *gin.Context) {
	svc := c.Query("service")
	key := c.Query("key")
	if svc == "" || key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "service and key are required"})
		return
	}

	config, err := s.ConfigService.Get(svc, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if config == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "config not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": config})
}

// listConfigs GET /configs?service=xxx
func (s *HTTPServer) listConfigs(c *gin.Context) {
	svc := c.Query("service")
	if svc == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "service is required"})
		return
	}

	configs, err := s.ConfigService.List(svc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": configs})
}

// setConfig POST /config — 明文存储
func (s *HTTPServer) setConfig(c *gin.Context) {
	var req struct {
		Service string `json:"service" binding:"required"`
		Key     string `json:"key" binding:"required"`
		Value   string `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if err := s.ConfigService.Set(req.Service, req.Key, req.Value); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok"})
}

// setEncryptedConfig POST /config/encrypted — 加密存储
func (s *HTTPServer) setEncryptedConfig(c *gin.Context) {
	var req struct {
		Service string `json:"service" binding:"required"`
		Key     string `json:"key" binding:"required"`
		Value   string `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if err := s.ConfigService.SetEncrypted(req.Service, req.Key, req.Value); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok (encrypted)"})
}

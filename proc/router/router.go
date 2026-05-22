package router

import (
	"github.com/gin-gonic/gin"

	"ttuser/proc/filter"
	"ttuser/proc/internal/handler"
)

// Router HTTP路由管理器
// 借鉴hlthproc模式：不同路由组使用不同filter链
type Router struct {
	Engine *gin.Engine
}

// Setup 配置路由并返回gin.Engine
func (r *Router) Setup() *gin.Engine {
	r.Engine = gin.Default()

	authHandler := &handler.AuthHandler{}
	authFilter := &filter.AuthFilter{}

	// ========== 公开路由组（无需鉴权） ==========
	publicGroup := r.Engine.Group("/api/v1")
	{
		publicGroup.POST("/register", authHandler.Register)
		publicGroup.POST("/login", authHandler.Login)
		publicGroup.POST("/refresh", authHandler.Refresh)
	}

	// ========== 需要Bearer Token鉴权的路由组 ==========
	authGroup := r.Engine.Group("/api/v1")
	authGroup.Use(authFilter.Filter)
	{
		authGroup.POST("/logout", authHandler.Logout)
		authGroup.GET("/user/info", authHandler.GetUserInfo)
		authGroup.PUT("/user/info", authHandler.UpdateUserInfo)
	}

	// ========== 未来扩展 ==========
	// 内部服务路由组（使用不同的filter）
	// innerGroup := r.Engine.Group("/inner/v1")
	// innerGroup.Use(innerAuthFilter.Filter)

	// 开放平台路由组（OAuth鉴权）
	// openGroup := r.Engine.Group("/open/v1")
	// openGroup.Use(oauthFilter.Filter)

	return r.Engine
}

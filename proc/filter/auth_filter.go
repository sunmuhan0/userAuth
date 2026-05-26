package filter

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ttuser/proc/sp"
)

const authGRPCTimeout = 10 * time.Second

// AuthFilter Bearer Token鉴权过滤器
// 用于需要用户登录态的路由组
type AuthFilter struct{}

// Filter 鉴权过滤器入口
func (f *AuthFilter) Filter(c *gin.Context) {
	// 1. 从Header中提取token
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "authorization header is required",
		})
		c.Abort()
		return
	}

	// 2. 解析Bearer格式
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "invalid authorization format, expected: Bearer <token>",
		})
		c.Abort()
		return
	}

	tokenStr := parts[1]

	// 3. 通过ServiceProvider获取AuthManager，调gRPC验证token
	ctx, cancel := context.WithTimeout(c.Request.Context(), authGRPCTimeout)
	defer cancel()
	resp, err := sp.Get().AuthManager.ValidateToken(ctx, tokenStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "token validation failed: " + err.Error(),
		})
		c.Abort()
		return
	}

	if !resp.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "invalid or expired token",
		})
		c.Abort()
		return
	}

	// 4. 将用户信息写入gin context
	c.Set("user_id", resp.UserId)
	c.Set("username", resp.Username)
	c.Set("token", tokenStr)

	c.Next()
}

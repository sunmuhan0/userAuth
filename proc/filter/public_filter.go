package filter

import (
	"github.com/gin-gonic/gin"
)

// PublicFilter 公开路由过滤器（无需鉴权）
// 可在此处做通用的请求日志、限流等
type PublicFilter struct{}

// Filter 公开路由过滤器入口
func (f *PublicFilter) Filter(c *gin.Context) {
	// 公开路由不做鉴权，直接放行
	// 可扩展：添加IP限流、请求日志等
	c.Next()
}

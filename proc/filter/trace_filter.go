package filter

import (
	"github.com/gin-gonic/gin"

	"ttuser/pkg/trace"
)

// TraceFilter 链路追踪中间件
// 从请求头 X-Trace-Id 提取trace_id，没有则自动生成
// 写入gin.Context和Go context，后续链路自动传递
func TraceFilter() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(trace.HeaderKey)
		if traceID == "" {
			traceID = trace.NewTraceID()
		}

		// 写入Go context（供gRPC调用等使用）
		ctx := trace.WithTraceID(c.Request.Context(), traceID)
		c.Request = c.Request.WithContext(ctx)

		// 同时设置到gin context（方便handler直接获取）
		c.Set("trace_id", traceID)

		// 响应头也带上trace_id（方便客户端/前端排查）
		c.Header(trace.HeaderKey, traceID)

		c.Next()
	}
}

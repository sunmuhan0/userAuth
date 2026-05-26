package filter

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"ttuser/pkg/metrics"
)

// MetricsFilter Prometheus HTTP指标采集中间件
func MetricsFilter() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		metrics.HTTPServerMetrics.ActiveRequests.Inc()
		defer metrics.HTTPServerMetrics.ActiveRequests.Dec()

		c.Next()

		status := c.Writer.Status()
		method := c.Request.Method
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}
		duration := time.Since(start).Seconds()
		metrics.HTTPServerMetrics.RequestCount.WithLabelValues(method, path, fmt.Sprintf("%d", status)).Inc()
		metrics.HTTPServerMetrics.RequestDuration.WithLabelValues(method, path).Observe(duration)
	}
}

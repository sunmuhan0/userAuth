package http

import (
	"bytes"
	"fmt"
	"io"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"ttuser/pkg/log"
	"ttuser/pkg/metrics"
	"ttuser/pkg/trace"
)

const maxBodySize = 4 * 1024

func traceFilter() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(trace.HeaderKey)
		if traceID == "" {
			traceID = trace.NewTraceID()
		}
		ctx := trace.WithTraceID(c.Request.Context(), traceID)
		c.Request = c.Request.WithContext(ctx)
		c.Set("trace_id", traceID)
		c.Header(trace.HeaderKey, traceID)
		c.Next()
	}
}

func accessLogFilter() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		var reqBody string
		if c.Request.Body != nil && c.Request.ContentLength > 0 {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err != nil {
				log.Warn(c.Request.Context(), "failed to read request body", "error", err)
			} else {
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				reqBody = truncate(string(bodyBytes))
			}
		}

		rw := &responseWriter{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
		c.Writer = rw

		c.Next()

		cost := fmt.Sprintf("%dms", time.Since(start).Milliseconds())

		uid, _ := c.Get("user_id")
		uidStr, _ := uid.(string)

		retBody := truncate(rw.body.String())
		httpStatus := c.Writer.Status()

		data := map[string]interface{}{
			"path":       c.Request.URL.Path,
			"method":     c.Request.Method,
			"UA":         c.GetHeader("User-Agent"),
			"cost":       cost,
			"uid":        uidStr,
			"client_ip":  c.ClientIP(),
			"httpStatus": httpStatus,
			"req":        tryParseJSON([]byte(reqBody)),
			"ret":        tryParseJSON([]byte(retBody)),
		}

		ctx := c.Request.Context()

		switch {
		case httpStatus >= 500:
			log.Error(ctx, "access_log", "data", data)
		case httpStatus >= 400:
			log.Warn(ctx, "access_log", "data", data)
		default:
			log.Info(ctx, "access_log", "data", data)
		}
	}
}

func metricsFilter() gin.HandlerFunc {
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

type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func truncate(s string) string {
	if len(s) > maxBodySize {
		b := []byte(s)
		b = b[:maxBodySize]
		for !utf8.Valid(b) {
			b = b[:len(b)-1]
		}
		return string(b) + "...(truncated)"
	}
	return s
}

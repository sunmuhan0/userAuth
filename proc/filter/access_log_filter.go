package filter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"

	"ttuser/pkg/log"
)

const maxBodySize = 4 * 1024 // 4KB

// AccessLogFilter 请求/响应日志中间件
// 记录：trace_id, method, path, cost, uid, client_ip, header, req, ret, http_status
func AccessLogFilter() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 读取请求body（需要缓存，Body只能读一次）
		var reqBody string
		if c.Request.Body != nil {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // 写回去让后续handler读
			reqBody = truncate(string(bodyBytes))
		}

		// 包装ResponseWriter以拦截响应体
		rw := &responseWriter{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
		c.Writer = rw

		// 执行后续handler
		c.Next()

		// 计算耗时
		cost := fmt.Sprintf("%dms", time.Since(start).Milliseconds())

		// 获取uid（鉴权后由auth_filter设置）
		uid, _ := c.Get("user_id")
		uidStr, _ := uid.(string)

		// 获取响应体
		retBody := truncate(rw.body.String())

		// 获取HTTP状态码
		httpStatus := c.Writer.Status()

		// 构建关键header（脱敏）
		header := buildSanitizedHeader(c)

		// 脱敏请求body
		reqBody = sanitizeBody(reqBody)

		// 获取context（带trace_id）
		ctx := c.Request.Context()

		// 根据状态码选择日志级别
		switch {
		case httpStatus >= 500:
			log.Error(ctx, "access",
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				"cost", cost,
				"uid", uidStr,
				"client_ip", c.ClientIP(),
				"header", header,
				"req", reqBody,
				"ret", retBody,
				"http_status", httpStatus,
			)
		case httpStatus >= 400:
			log.Warn(ctx, "access",
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				"cost", cost,
				"uid", uidStr,
				"client_ip", c.ClientIP(),
				"header", header,
				"req", reqBody,
				"ret", retBody,
				"http_status", httpStatus,
			)
		default:
			log.Info(ctx, "access",
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				"cost", cost,
				"uid", uidStr,
				"client_ip", c.ClientIP(),
				"header", header,
				"req", reqBody,
				"ret", retBody,
				"http_status", httpStatus,
			)
		}
	}
}

// responseWriter 包装gin.ResponseWriter，拦截写入以捕获响应体
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b) // 同时写入缓冲
	return w.ResponseWriter.Write(b)
}

// buildSanitizedHeader 构建关键header JSON（脱敏Authorization）
func buildSanitizedHeader(c *gin.Context) string {
	headers := make(map[string]string)

	if v := c.GetHeader("Content-Type"); v != "" {
		headers["Content-Type"] = v
	}
	if v := c.GetHeader("User-Agent"); v != "" {
		headers["User-Agent"] = v
	}
	if v := c.GetHeader("Authorization"); v != "" {
		headers["Authorization"] = "Bearer ***"
	}

	b, _ := json.Marshal(headers)
	return string(b)
}

// sanitizeBody 脱敏请求body中的password字段
func sanitizeBody(body string) string {
	if body == "" {
		return ""
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		// 不是JSON格式，原样返回
		return body
	}

	if _, ok := data["password"]; ok {
		data["password"] = "***"
	}

	b, _ := json.Marshal(data)
	return string(b)
}

// truncate 截断超过4KB的字符串
func truncate(s string) string {
	if len(s) > maxBodySize {
		return s[:maxBodySize] + "...(truncated)"
	}
	return s
}

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
// 日志格式参照hlthproc：msg=path, 业务数据打包为JSON data字段
func AccessLogFilter() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 读取请求body（需要缓存，Body只能读一次）
		var reqBody string
		if c.Request.Body != nil {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
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

		// 解析ret为JSON对象（如果可以的话）
		var retObj interface{}
		if err := json.Unmarshal([]byte(retBody), &retObj); err != nil {
			retObj = retBody
		}

		// 解析req为JSON对象
		var reqObj interface{}
		if err := json.Unmarshal([]byte(reqBody), &reqObj); err != nil {
			reqObj = reqBody
		}

		// 组装data字段（所有业务数据打包为一个JSON对象）
		data := map[string]interface{}{
			"UA":         c.GetHeader("User-Agent"),
			"cost":       cost,
			"uid":        uidStr,
			"client_ip":  c.ClientIP(),
			"header":     header,
			"req":        reqObj,
			"ret":        retObj,
			"httpStatus": httpStatus,
		}

		// path放入data
		data["path"] = c.Request.URL.Path
		data["method"] = c.Request.Method

		// 获取context（带trace_id）
		ctx := c.Request.Context()

		// 根据状态码选择日志级别
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

// responseWriter 包装gin.ResponseWriter，拦截写入以捕获响应体
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// buildSanitizedHeader 构建关键header（脱敏Authorization）
func buildSanitizedHeader(c *gin.Context) map[string]string {
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

	return headers
}

// sanitizeBody 脱敏请求body中的password字段
func sanitizeBody(body string) string {
	if body == "" {
		return ""
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(body), &data); err != nil {
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

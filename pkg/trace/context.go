package trace

import "context"

type traceKey struct{}

// WithTraceID 将trace_id写入context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceKey{}, traceID)
}

// GetTraceID 从context取出trace_id，不存在返回空字符串
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	val := ctx.Value(traceKey{})
	if val == nil {
		return ""
	}
	return val.(string)
}

// MetadataKey gRPC metadata中trace_id的key名
const MetadataKey = "x-trace-id"

// HeaderKey HTTP header中trace_id的key名
const HeaderKey = "X-Trace-Id"

// PropertyKey RocketMQ message property中trace_id的key名（备用）
const PropertyKey = "trace_id"

package interceptor

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"ttuser/pkg/metrics"
	"ttuser/pkg/trace"
)

// UnaryServerTraceInterceptor gRPC服务端追踪拦截器
// 从incoming metadata提取trace_id，写入ctx
func UnaryServerTraceInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 从gRPC metadata提取trace_id
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			values := md.Get(trace.MetadataKey)
			if len(values) > 0 && values[0] != "" {
				ctx = trace.WithTraceID(ctx, values[0])
			}
		}

		// 如果没有trace_id，生成一个（兜底）
		if trace.GetTraceID(ctx) == "" {
			ctx = trace.WithTraceID(ctx, trace.NewTraceID())
		}

		// 记录gRPC调用指标（带service标签）
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		st := status.Code(err).String()
		metrics.RecordGRPCCall(ctx, info.FullMethod, st, duration)

		return resp, err
	}
}

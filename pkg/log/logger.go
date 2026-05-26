package log

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"ttuser/pkg/trace"
)

var logger *zap.Logger

func init() {
	config := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		MessageKey:     "msg",
		CallerKey:      "caller",
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(config),
		zapcore.AddSync(os.Stdout),
		zapcore.InfoLevel,
	)

	logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
}

// GetLogger 获取底层zap.Logger（用于需要直接操作的场景）
func GetLogger() *zap.Logger {
	return logger
}

// Info 带trace_id的Info日志
func Info(ctx context.Context, msg string, fields ...zap.Field) {
	fields = appendTraceID(ctx, fields)
	logger.Info(msg, fields...)
}

// Error 带trace_id的Error日志
func Error(ctx context.Context, msg string, fields ...zap.Field) {
	fields = appendTraceID(ctx, fields)
	logger.Error(msg, fields...)
}

// Warn 带trace_id的Warn日志
func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	fields = appendTraceID(ctx, fields)
	logger.Warn(msg, fields...)
}

// Debug 带trace_id的Debug日志
func Debug(ctx context.Context, msg string, fields ...zap.Field) {
	fields = appendTraceID(ctx, fields)
	logger.Debug(msg, fields...)
}

// appendTraceID 从ctx取trace_id，追加到fields
func appendTraceID(ctx context.Context, fields []zap.Field) []zap.Field {
	if ctx == nil {
		return fields
	}
	traceID := trace.GetTraceID(ctx)
	if traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}
	return fields
}

// Sync 刷新日志缓冲（程序退出前调用）
func Sync() {
	_ = logger.Sync()
}

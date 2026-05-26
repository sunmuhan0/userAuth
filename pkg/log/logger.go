package log

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"ttuser/pkg/trace"
)

var logger *zap.Logger
var sugar *zap.SugaredLogger

func init() {
	// 默认初始化（仅stdout），如需Loki推送调用 InitWithLoki()
	initLogger(nil)
}

// initLogger 初始化zap logger
// 如果lokiWriter不为nil，日志同时输出到stdout和Loki
func initLogger(lokiWriter zapcore.WriteSyncer) {
	config := zapcore.EncoderConfig{
		TimeKey:      "ts",
		LevelKey:     "level",
		MessageKey:   "msg",
		CallerKey:    "caller",
		EncodeTime:   zapcore.ISO8601TimeEncoder,
		EncodeLevel:  zapcore.LowercaseLevelEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}

	encoder := zapcore.NewJSONEncoder(config)

	var core zapcore.Core
	if lokiWriter != nil {
		// 同时输出到stdout和Loki
		core = zapcore.NewCore(
			encoder,
			zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), lokiWriter),
			zapcore.InfoLevel,
		)
	} else {
		core = zapcore.NewCore(
			encoder,
			zapcore.AddSync(os.Stdout),
			zapcore.InfoLevel,
		)
	}

	logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	sugar = logger.Sugar()
}

// InitWithLoki 初始化日志并启用Loki推送
// 在应用启动时调用：log.InitWithLoki(log.DefaultLokiConfig())
func InitWithLoki(config *LokiConfig) {
	InitLoki(config)
	if config != nil && config.Enable {
		initLogger(&lokiWriteSyncer{})
	}
}

// lokiWriteSyncer 实现 zapcore.WriteSyncer，将日志推送到Loki
type lokiWriteSyncer struct{}

func (w *lokiWriteSyncer) Write(p []byte) (n int, err error) {
	pushToLoki(string(p))
	return len(p), nil
}

func (w *lokiWriteSyncer) Sync() error {
	if loki != nil {
		loki.flush()
	}
	return nil
}

// ==================== printf 风格（简单场景，推荐） ====================

// Infof printf风格的Info日志，自动带trace_id
func Infof(ctx context.Context, format string, args ...interface{}) {
	sugar.Infow(fmt.Sprintf(format, args...), traceFields(ctx)...)
}

// Errorf printf风格的Error日志，自动带trace_id
func Errorf(ctx context.Context, format string, args ...interface{}) {
	sugar.Errorw(fmt.Sprintf(format, args...), traceFields(ctx)...)
}

// Warnf printf风格的Warn日志，自动带trace_id
func Warnf(ctx context.Context, format string, args ...interface{}) {
	sugar.Warnw(fmt.Sprintf(format, args...), traceFields(ctx)...)
}

// Debugf printf风格的Debug日志，自动带trace_id
func Debugf(ctx context.Context, format string, args ...interface{}) {
	sugar.Debugw(fmt.Sprintf(format, args...), traceFields(ctx)...)
}

// ==================== kv 风格（结构化场景） ====================
// 用法：log.Info(ctx, "user registered", "user_id", "xxx", "username", "test")
// keysAndValues 必须是偶数个参数，key-value交替

// Info 结构化Info日志，自动带trace_id
func Info(ctx context.Context, msg string, keysAndValues ...interface{}) {
	kvs := appendTraceKV(ctx, keysAndValues)
	sugar.Infow(msg, kvs...)
}

// Error 结构化Error日志，自动带trace_id
func Error(ctx context.Context, msg string, keysAndValues ...interface{}) {
	kvs := appendTraceKV(ctx, keysAndValues)
	sugar.Errorw(msg, kvs...)
}

// Warn 结构化Warn日志，自动带trace_id
func Warn(ctx context.Context, msg string, keysAndValues ...interface{}) {
	kvs := appendTraceKV(ctx, keysAndValues)
	sugar.Warnw(msg, kvs...)
}

// Debug 结构化Debug日志，自动带trace_id
func Debug(ctx context.Context, msg string, keysAndValues ...interface{}) {
	kvs := appendTraceKV(ctx, keysAndValues)
	sugar.Debugw(msg, kvs...)
}

// ==================== 工具函数 ====================

// traceFields 返回trace_id的kv对（用于printf风格）
func traceFields(ctx context.Context) []interface{} {
	if ctx == nil {
		return nil
	}
	traceID := trace.GetTraceID(ctx)
	if traceID != "" {
		return []interface{}{"trace_id", traceID}
	}
	return nil
}

// appendTraceKV 将trace_id追加到kv列表
func appendTraceKV(ctx context.Context, keysAndValues []interface{}) []interface{} {
	if ctx == nil {
		return keysAndValues
	}
	traceID := trace.GetTraceID(ctx)
	if traceID != "" {
		keysAndValues = append(keysAndValues, "trace_id", traceID)
	}
	return keysAndValues
}

// Sync 刷新日志缓冲（程序退出前调用）
func Sync() {
	_ = logger.Sync()
	StopLoki()
}

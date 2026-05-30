package log

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/teou/inji"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"ttuser/pkg/trace"
)

var logger *zap.Logger
var sugar *zap.SugaredLogger

func init() {
	// 默认初始化（仅stdout），应用启动时 main 中调用 Init 重新初始化
	initLogger(nil, nil)
}

// LogConfig 日志配置
type LogConfig struct {
	ServiceName string // 服务名，如 "auth-server"
	Port        int    // 服务端口，如 9090
	LogDir      string // 日志根目录，默认 ./log
}

// DefaultLogConfig 默认日志配置，从 inji 容器获取服务名和端口
// main 中先注册：
//
//	inji.Reg("serverName", "auth-server")
//	inji.Reg("serverPort", "9090")
func DefaultLogConfig() *LogConfig {
	svc := "auth-server"
	if v, ok := inji.Find("serverName"); ok {
		if s, ok := v.(string); ok && s != "" {
			svc = s
		}
	}
	port := 9090
	if v, ok := inji.Find("serverPort"); ok {
		switch val := v.(type) {
		case string:
			if p, err := fmt.Sscanf(val, "%d", &port); err != nil || p != 1 {
				port = 9090
			}
		case int:
			port = val
		}
	}
	return &LogConfig{
		ServiceName: svc,
		Port:        port,
		LogDir:      "./log",
	}
}

// lumberjackLogger 默认日志轮转配置
func newLumberjackWriter(logDir, serviceName string, port int) zapcore.WriteSyncer {
	logFile := filepath.Join(logDir, fmt.Sprintf("%s_%d.log", serviceName, port))
	dir := filepath.Dir(logFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("[log] failed to create log dir %s: %v\n", dir, err)
		return nil
	}
	lw := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    100, // MB
		MaxAge:     30,  // days
		MaxBackups: 10,
		LocalTime:  true,
		Compress:   true,
	}
	fmt.Printf("[log] writing to file: %s (maxSize=%dMB, maxAge=%dd, maxBackups=%d, compress=%t)\n",
		logFile, 100, 30, 10, true)
	return zapcore.AddSync(lw)
}

// Init 初始化日志（输出到stdout + 文件）
// 日志文件路径：{logDir}/{serviceName}_{port}/app.log
func Init(config *LogConfig) {
	if config == nil {
		config = DefaultLogConfig()
	}

	var fileWriter zapcore.WriteSyncer
	if config.ServiceName != "" && config.Port > 0 {
		fileWriter = newLumberjackWriter(config.LogDir, config.ServiceName, config.Port)
	}

	initLogger(fileWriter, nil)
}

// initLogger 初始化zap logger
func initLogger(fileWriter, lokiWriter zapcore.WriteSyncer) {
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

	// 构建写入目标列表
	writers := []zapcore.WriteSyncer{zapcore.AddSync(os.Stdout)}
	if fileWriter != nil {
		writers = append(writers, fileWriter)
	}
	if lokiWriter != nil {
		writers = append(writers, lokiWriter)
	}

	core := zapcore.NewCore(
		encoder,
		zapcore.NewMultiWriteSyncer(writers...),
		zapcore.InfoLevel,
	)

	logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	sugar = logger.Sugar()
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

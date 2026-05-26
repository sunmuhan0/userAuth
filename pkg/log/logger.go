package log

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"ttuser/pkg/trace"
)

var logger *zap.Logger
var sugar *zap.SugaredLogger

// LogConfig 日志配置
type LogConfig struct {
	ServiceName string // 服务名，如 "auth-server"
	Port        int    // 服务端口，如 9090
	LogDir      string // 日志根目录，默认 /home/work/log
}

// DefaultLogConfig 默认日志配置
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		LogDir: "/home/work/log",
	}
}

func init() {
	// 默认初始化（仅stdout），应用启动时应调用 Init() 重新初始化
	initLogger(nil, nil)
}

// Init 初始化日志（输出到stdout + 文件）
// 日志文件路径：{logDir}/{serviceName}_{port}/20260526.log
func Init(config *LogConfig) {
	if config == nil {
		config = DefaultLogConfig()
	}

	var fileWriter zapcore.WriteSyncer
	if config.ServiceName != "" && config.Port > 0 {
		dir := filepath.Join(config.LogDir, fmt.Sprintf("%s_%d", config.ServiceName, config.Port))
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("[log] failed to create log dir %s: %v\n", dir, err)
		} else {
			logFile := filepath.Join(dir, time.Now().Format("20060102")+".log")
			f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Printf("[log] failed to open log file %s: %v\n", logFile, err)
			} else {
				fileWriter = zapcore.AddSync(f)
				fmt.Printf("[log] writing to file: %s\n", logFile)
			}
		}
	}

	initLogger(fileWriter, nil)
}

// InitWithLoki 初始化日志并启用Loki推送（输出到stdout + 文件 + Loki）
// 在应用启动时调用
func InitWithLoki(config *LogConfig, lokiConfig *LokiConfig) {
	if config == nil {
		config = DefaultLogConfig()
	}

	var fileWriter zapcore.WriteSyncer
	if config.ServiceName != "" && config.Port > 0 {
		dir := filepath.Join(config.LogDir, fmt.Sprintf("%s_%d", config.ServiceName, config.Port))
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("[log] failed to create log dir %s: %v\n", dir, err)
		} else {
			logFile := filepath.Join(dir, time.Now().Format("20060102")+".log")
			f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Printf("[log] failed to open log file %s: %v\n", logFile, err)
			} else {
				fileWriter = zapcore.AddSync(f)
				fmt.Printf("[log] writing to file: %s\n", logFile)
			}
		}
	}

	InitLoki(lokiConfig)
	var lokiWriter zapcore.WriteSyncer
	if lokiConfig != nil && lokiConfig.Enable {
		lokiWriter = &lokiWriteSyncer{}
	}

	initLogger(fileWriter, lokiWriter)
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

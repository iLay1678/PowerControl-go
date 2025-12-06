package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	loggers    = make(map[string]*zap.SugaredLogger)
	globalLog  *zap.Logger
	dataDir    = "/app/data"
	logLevel   zapcore.Level
	keepDays   int
)

// Init 初始化日志系统
func Init(level string, days int) error {
	if dir := os.Getenv("DATA_DIR"); dir != "" {
		dataDir = dir
	}

	keepDays = days

	// 解析日志级别
	switch level {
	case "DEBUG":
		logLevel = zapcore.DebugLevel
	case "INFO":
		logLevel = zapcore.InfoLevel
	case "WARN", "WARNING":
		logLevel = zapcore.WarnLevel
	case "ERROR":
		logLevel = zapcore.ErrorLevel
	default:
		logLevel = zapcore.InfoLevel
	}

	// 创建日志目录
	logDir := filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	// 配置日志输出
	logFile := filepath.Join(logDir, "powercontrol.log")

	// 文件输出配置（带日志轮转）
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    100, // MB
		MaxBackups: keepDays,
		MaxAge:     keepDays, // days
		Compress:   true,
	})

	// 控制台输出配置
	consoleWriter := zapcore.AddSync(os.Stdout)

	// 编码器配置
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05"))
	}
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	// 创建核心
	core := zapcore.NewTee(
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			consoleWriter,
			logLevel,
		),
		zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			fileWriter,
			logLevel,
		),
	)

	// 创建全局日志记录器
	globalLog = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	// 启动日志清理任务
	go cleanOldLogs()

	return nil
}

// GetLogger 获取指定名称的日志记录器
func GetLogger(name string) *zap.SugaredLogger {
	if logger, ok := loggers[name]; ok {
		return logger
	}

	logger := globalLog.Named(name).Sugar()
	loggers[name] = logger
	return logger
}

// Sync 同步日志
func Sync() {
	if globalLog != nil {
		_ = globalLog.Sync()
	}
}

// cleanOldLogs 清理过期日志文件
func cleanOldLogs() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		logDir := filepath.Join(dataDir, "logs")
		files, err := filepath.Glob(filepath.Join(logDir, "*.log*"))
		if err != nil {
			continue
		}

		cutoff := time.Now().AddDate(0, 0, -keepDays)
		for _, file := range files {
			info, err := os.Stat(file)
			if err != nil {
				continue
			}

			if info.ModTime().Before(cutoff) {
				_ = os.Remove(file)
			}
		}
	}
}

package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Log is the global logger instance.
	Log *zap.Logger
	// Sugar is the sugared logger for convenient logging.
	Sugar *zap.SugaredLogger
)

func init() {
	// Initialize with a no-op logger by default to prevent nil pointer panics in tests.
	Log = zap.NewNop()
	Sugar = Log.Sugar()
}

// Config holds logger configuration.
type Config struct {
	Level  string // debug, info, warn, error
	Format string // json or text
}

// Init initializes the global logger with the given configuration.
func Init(cfg Config) error {
	level := parseLevel(cfg.Level)

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	var consoleEncoder zapcore.Encoder
	if cfg.Format == "text" {
		consoleCfg := encoderConfig
		consoleCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		consoleEncoder = zapcore.NewConsoleEncoder(consoleCfg)
	} else {
		consoleEncoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	core := zapcore.NewCore(
		consoleEncoder,
		zapcore.AddSync(os.Stdout),
		level,
	)

	Log = zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(0),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	Sugar = Log.Sugar()

	return nil
}

func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// Sync flushes any buffered log entries.
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}

// Debug logs a debug message.
func Debug(msg string, fields ...zap.Field) { Log.Debug(msg, fields...) }

// Info logs an info message.
func Info(msg string, fields ...zap.Field) { Log.Info(msg, fields...) }

// Warn logs a warning message.
func Warn(msg string, fields ...zap.Field) { Log.Warn(msg, fields...) }

// Error logs an error message.
func Error(msg string, fields ...zap.Field) { Log.Error(msg, fields...) }

// Fatal logs a fatal message and exits.
func Fatal(msg string, fields ...zap.Field) { Log.Fatal(msg, fields...) }

// With creates a child logger with additional fields.
func With(fields ...zap.Field) *zap.Logger { return Log.With(fields...) }

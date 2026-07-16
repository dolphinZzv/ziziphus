package logger

import (
	"io"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config holds logger initialization parameters.
type Config struct {
	Level      string // debug, info, warn, error
	File       string // log file path (empty = stdout only)
	MaxSize    int    // megabytes before rotation
	MaxAge     int    // days to retain old logs
	MaxBackups int    // number of old log files to retain
	Compress   bool   // compress rotated files
}

var (
	sugar       *zap.SugaredLogger
	atomicLevel zap.AtomicLevel
)

// Init configures the global zap logger. Must be called once at startup.
func Init(cfg Config) {
	atomicLevel = zap.NewAtomicLevel()
	atomicLevel.SetLevel(parseLevel(cfg.Level))

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "time"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderCfg.EncodeCaller = zapcore.ShortCallerEncoder

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		newWriter(cfg),
		atomicLevel,
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	zap.ReplaceGlobals(logger)
	sugar = logger.Sugar()
}

// newWriter returns a teed writer: stdout + optional file with rotation.
func newWriter(cfg Config) zapcore.WriteSyncer {
	writers := []io.Writer{os.Stdout}

	if cfg.File != "" {
		writers = append(writers, &lumberjack.Logger{
			Filename:   cfg.File,
			MaxSize:    cfg.MaxSize,
			MaxAge:     cfg.MaxAge,
			MaxBackups: cfg.MaxBackups,
			Compress:   cfg.Compress,
		})
	}

	if len(writers) == 1 {
		return zapcore.AddSync(writers[0])
	}
	return zapcore.AddSync(io.MultiWriter(writers...))
}

func parseLevel(s string) zapcore.Level {
	switch strings.ToLower(s) {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// Sync flushes any buffered log entries. Call before shutdown.
func Sync() {
	if sugar != nil {
		_ = sugar.Sync()
	}
}

// SetLevel dynamically changes the log level at runtime.
// Must be called after Init. Valid values: "debug", "info", "warn", "error".
func SetLevel(level string) {
	if atomicLevel != (zap.AtomicLevel{}) {
		atomicLevel.SetLevel(parseLevel(level))
	}
}

func Debug(msg string, args ...any) {
	if sugar == nil {
		return
	}
	sugar.Debugw(msg, args...)
}

func Info(msg string, args ...any) {
	if sugar == nil {
		return
	}
	sugar.Infow(msg, args...)
}

func Warn(msg string, args ...any) {
	if sugar == nil {
		return
	}
	sugar.Warnw(msg, args...)
}

func Error(msg string, args ...any) {
	if sugar == nil {
		return
	}
	sugar.Errorw(msg, args...)
}

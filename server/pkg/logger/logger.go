package logger

import (
	"io"
	"os"
	"path/filepath"
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

	ws := newWriter(cfg)

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		ws,
		atomicLevel,
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	zap.ReplaceGlobals(logger)
	sugar = logger.Sugar()

	// Log startup diagnostics to stderr so they're visible even before
	// the file logger is fully initialized.
	os.Stderr.WriteString("ziziphus: logger initialized level=" + cfg.Level)
	if cfg.File != "" {
		absPath := cfg.File
		if !filepath.IsAbs(cfg.File) {
			if wd, err := os.Getwd(); err == nil {
				absPath = filepath.Join(wd, cfg.File)
			}
		}
		os.Stderr.WriteString(" file=" + absPath)

		// Detect common misconfiguration: path looks like a directory, not a file
		if fi, err := os.Stat(cfg.File); err == nil && fi.IsDir() {
			os.Stderr.WriteString(" (ERROR: path \"" + cfg.File + "\" is an existing DIRECTORY, not a file. Use a file path like \"./logs/ziziphus.log\")")
		} else if filepath.Ext(cfg.File) == "" && !strings.Contains(cfg.File, ".") {
			os.Stderr.WriteString(" (WARNING: no file extension — make sure \"" + cfg.File + "\" points to a file, not a directory. Try \"./logs/ziziphus.log\")")
		}

		// Verify write access by creating the file immediately
		f, err := os.OpenFile(cfg.File, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			os.Stderr.WriteString(" (WARNING: cannot write: " + err.Error() + ")")
		} else {
			f.Close()
		}
	}
	os.Stderr.WriteString("\n")
}

// newWriter returns a teed writer: stdout + optional file with rotation.
func newWriter(cfg Config) zapcore.WriteSyncer {
	writers := []io.Writer{os.Stdout}

	if cfg.File != "" {
		// Ensure parent directory exists
		if dir := filepath.Dir(cfg.File); dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				// If it fails because dir exists as a FILE (common migration issue
				// when config previously pointed "file" to a directory path), remove
				// the stale file and retry.
				if fi, statErr := os.Stat(dir); statErr == nil && !fi.IsDir() {
					os.Stderr.WriteString("ziziphus: removing stale file \"" + dir + "\", replacing with directory\n")
					os.Remove(dir)
					if mkErr := os.MkdirAll(dir, 0755); mkErr != nil {
						os.Stderr.WriteString("ziziphus: still cannot create directory \"" + dir + "\": " + mkErr.Error() + "\n")
					}
				} else {
					os.Stderr.WriteString("ziziphus: cannot create directory \"" + dir + "\": " + err.Error() + "\n")
				}
			}
		}
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

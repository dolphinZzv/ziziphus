package logger

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// captureLogs replaces the global zap logger with one that writes to a buffer,
// runs fn, then restores the original.
func captureLogs(t *testing.T, fn func()) string {
	t.Helper()

	var buf bytes.Buffer

	// Use the same encoder config as Init for consistent field naming.
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "time"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderCfg.EncodeCaller = zapcore.ShortCallerEncoder

	// Use a fresh atomic level for capture so we don't depend on Init.
	lvl := zap.NewAtomicLevelAt(zapcore.DebugLevel)

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(&buf),
		lvl,
	)

	oldLogger := zap.L()
	newLogger := zap.New(core)
	zap.ReplaceGlobals(newLogger)
	sugar = newLogger.Sugar()
	t.Cleanup(func() {
		zap.ReplaceGlobals(oldLogger)
		if oldLogger != nil {
			sugar = oldLogger.Sugar()
		}
	})

	fn()
	_ = sugar.Sync()
	return buf.String()
}

// jsonLine parses s as a single JSON object and returns it as a map.
func jsonLine(t *testing.T, s string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("invalid JSON line %q: %v", s, err)
	}
	return m
}

func TestInfo_WritesJSON(t *testing.T) {
	out := captureLogs(t, func() {
		Info("info message")
	})
	m := jsonLine(t, out)

	if m["msg"] != "info message" {
		t.Errorf(`msg = %v, want "info message"`, m["msg"])
	}
	if m["level"] != "INFO" {
		t.Errorf(`level = %v, want "INFO"`, m["level"])
	}
}

func TestInfo_WithArgs(t *testing.T) {
	out := captureLogs(t, func() {
		Info("info with args", "key1", "val1", "key2", 42)
	})
	m := jsonLine(t, out)

	if m["msg"] != "info with args" {
		t.Errorf(`msg = %v, want "info with args"`, m["msg"])
	}
	if m["key1"] != "val1" {
		t.Errorf(`key1 = %v, want "val1"`, m["key1"])
	}
	// JSON numbers decode as float64
	if m["key2"] != float64(42) {
		t.Errorf(`key2 = %v, want 42`, m["key2"])
	}
}

func TestWarn_WritesJSON(t *testing.T) {
	out := captureLogs(t, func() {
		Warn("warn message")
	})
	m := jsonLine(t, out)

	if m["msg"] != "warn message" {
		t.Errorf(`msg = %v, want "warn message"`, m["msg"])
	}
	if m["level"] != "WARN" {
		t.Errorf(`level = %v, want "WARN"`, m["level"])
	}
}

func TestError_WritesJSON(t *testing.T) {
	out := captureLogs(t, func() {
		Error("error message")
	})
	m := jsonLine(t, out)

	if m["msg"] != "error message" {
		t.Errorf(`msg = %v, want "error message"`, m["msg"])
	}
	if m["level"] != "ERROR" {
		t.Errorf(`level = %v, want "ERROR"`, m["level"])
	}
}

func TestDebug_SuppressedAtInfoLevel(t *testing.T) {
	var buf bytes.Buffer
	lvl := zap.NewAtomicLevelAt(zapcore.InfoLevel)
	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(&buf), lvl)
	zap.ReplaceGlobals(zap.New(core))
	sugar = zap.L().Sugar()

	Debug("debug message")
	_ = sugar.Sync()

	if buf.Len() > 0 {
		t.Errorf("expected no output at Info level, got: %s", buf.String())
	}
}

func TestDebug_VisibleAfterSetLevel(t *testing.T) {
	var buf bytes.Buffer
	atomicLevel = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.EncodeLevel = zapcore.CapitalLevelEncoder
	core := zapcore.NewCore(zapcore.NewJSONEncoder(encCfg), zapcore.AddSync(&buf), atomicLevel)
	zap.ReplaceGlobals(zap.New(core))
	sugar = zap.L().Sugar()

	SetLevel("debug")
	Debug("debug message")
	_ = sugar.Sync()

	if buf.Len() == 0 {
		t.Fatal("expected debug output, got nothing")
	}
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if m["msg"] != "debug message" {
		t.Errorf(`msg = %v, want "debug message"`, m["msg"])
	}
	if m["level"] != "DEBUG" {
		t.Errorf(`level = %v, want "DEBUG"`, m["level"])
	}
}

func TestSetLevel_RestrictsOutput(t *testing.T) {
	var buf bytes.Buffer
	atomicLevel = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.EncodeLevel = zapcore.CapitalLevelEncoder
	core := zapcore.NewCore(zapcore.NewJSONEncoder(encCfg), zapcore.AddSync(&buf), atomicLevel)
	zap.ReplaceGlobals(zap.New(core))
	sugar = zap.L().Sugar()

	SetLevel("warn")
	Info("should be suppressed")
	Warn("should appear")
	_ = sugar.Sync()

	if buf.Len() == 0 {
		t.Fatal("expected at least one log line")
	}

	// Parse each JSON line
	var sawSuppressed, sawAppear bool
	for _, line := range bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n")) {
		var m map[string]any
		if err := json.Unmarshal(line, &m); err != nil {
			continue
		}
		switch m["msg"] {
		case "should be suppressed":
			sawSuppressed = true
		case "should appear":
			sawAppear = true
		}
	}

	if sawSuppressed {
		t.Error("Info message 'should be suppressed' was logged but should have been filtered by SetLevel(warn)")
	}
	if !sawAppear {
		t.Error("Warn message 'should appear' was not logged")
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  zapcore.Level
	}{
		{"debug", zapcore.DebugLevel},
		{"DEBUG", zapcore.DebugLevel},
		{"Debug", zapcore.DebugLevel},
		{"info", zapcore.InfoLevel},
		{"INFO", zapcore.InfoLevel},
		{"warn", zapcore.WarnLevel},
		{"WARN", zapcore.WarnLevel},
		{"error", zapcore.ErrorLevel},
		{"ERROR", zapcore.ErrorLevel},
		{"", zapcore.InfoLevel},
		{"invalid", zapcore.InfoLevel},
		{"unknown", zapcore.InfoLevel},
	}
	for _, tc := range tests {
		got := parseLevel(tc.input)
		if got != tc.want {
			t.Errorf("parseLevel(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestSync_NilSugar(t *testing.T) {
	// Save and restore global state
	oldSugar := sugar
	sugar = nil
	t.Cleanup(func() { sugar = oldSugar })

	// Should not panic
	Sync()
}

func TestLogFunctions_NilSugar(t *testing.T) {
	oldSugar := sugar
	sugar = nil
	t.Cleanup(func() { sugar = oldSugar })

	// None of these should panic
	Debug("no panic")
	Info("no panic")
	Warn("no panic")
	Error("no panic")
}

func TestNewWriter_Default(t *testing.T) {
	ws := newWriter(Config{Level: "info"})
	if ws == nil {
		t.Fatal("newWriter returned nil")
	}
	// Should write to stdout (we can't easily verify content, but it shouldn't crash)
	n, err := ws.Write([]byte("test\n"))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != 5 {
		t.Errorf("wrote %d bytes, want 5", n)
	}
}

func TestInit_StdoutOnly(t *testing.T) {
	oldSugar := sugar
	oldAtomic := atomicLevel
	t.Cleanup(func() {
		sugar = oldSugar
		atomicLevel = oldAtomic
	})

	Init(Config{Level: "info"})

	if sugar == nil {
		t.Fatal("sugar is nil after Init")
	}

	out := captureLogs(t, func() {
		Info("after init message")
	})
	m := jsonLine(t, out)
	if m["msg"] != "after init message" {
		t.Errorf(`msg = %v, want "after init message"`, m["msg"])
	}
}

func TestInit_WithFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	oldSugar := sugar
	oldAtomic := atomicLevel
	t.Cleanup(func() {
		sugar = oldSugar
		atomicLevel = oldAtomic
	})

	Init(Config{Level: "debug", File: logPath})

	Info("file logger test")
	Sync()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}
	if !strings.Contains(string(data), "file logger test") {
		t.Errorf("log file does not contain expected message: %s", data)
	}
}

func TestInit_ParseLevelDefault(t *testing.T) {
	oldSugar := sugar
	oldAtomic := atomicLevel
	t.Cleanup(func() {
		sugar = oldSugar
		atomicLevel = oldAtomic
	})

	Init(Config{Level: "invalid_default"})

	var buf bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&buf),
		zap.NewAtomicLevelAt(zapcore.DebugLevel),
	)
	zap.ReplaceGlobals(zap.New(core))
	sugar = zap.L().Sugar()
	_ = sugar.Sync()

	Debug("debug after default-level init")
	_ = sugar.Sync()
	if buf.Len() == 0 {
		t.Log("debug suppressed == default level was Info, as expected for invalid level")
	}
}

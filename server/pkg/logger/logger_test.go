package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

// captureLogs swaps the global slog default for one that writes to a
// buffer, runs fn, restores the original default, and returns the captured
// output.
func captureLogs(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer

	old := slog.Default()
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: programLevel})
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(old) })

	fn()
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
	// Ensure level is Info (as init() sets it).
	programLevel.Set(slog.LevelInfo)

	out := captureLogs(t, func() {
		Debug("debug message")
	})
	if out != "" {
		t.Errorf("expected no output at Info level, got: %s", out)
	}
}

func TestDebug_VisibleAfterSetLevel(t *testing.T) {
	programLevel.Set(slog.LevelDebug)
	t.Cleanup(func() { programLevel.Set(slog.LevelInfo) })

	out := captureLogs(t, func() {
		Debug("debug message")
	})
	m := jsonLine(t, out)

	if m["msg"] != "debug message" {
		t.Errorf(`msg = %v, want "debug message"`, m["msg"])
	}
	if m["level"] != "DEBUG" {
		t.Errorf(`level = %v, want "DEBUG"`, m["level"])
	}
}

func TestSetLevel_RestrictsOutput(t *testing.T) {
	programLevel.Set(slog.LevelWarn)
	t.Cleanup(func() { programLevel.Set(slog.LevelInfo) })

	out := captureLogs(t, func() {
		Info("should be suppressed")
		Warn("should appear")
	})
	m := jsonLine(t, out)

	if m["msg"] != "should appear" {
		t.Errorf(`msg = %v, want "should appear"`, m["msg"])
	}
	if m["level"] != "WARN" {
		t.Errorf(`level = %v, want "WARN"`, m["level"])
	}
}

func TestWithContext_ReturnsLogger(t *testing.T) {
	logger := WithContext(nil, "req_id", "abc-123")
	if logger == nil {
		t.Fatal("WithContext returned nil")
	}
}

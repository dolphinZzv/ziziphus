package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"ziziphus/config"
)

func TestAnnouncement(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(`
jwt:
  secret: this-is-a-long-enough-test-secret-key-32+
announcement:
  enabled: true
  title: "维护通知"
  body: "系统将于今晚进行维护"
  url: "https://example.com/notice"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	mgr, err := config.NewManager(path)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	handler := Announcement(mgr)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/announcement", nil)
	handler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	resp := decodeResponse(t, w)
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Data is not a map")
	}
	if data["enabled"].(bool) != true {
		t.Error("enabled = false, want true")
	}
	if data["title"].(string) != "维护通知" {
		t.Errorf("title = %q, want %q", data["title"], "维护通知")
	}
	if data["body"].(string) != "系统将于今晚进行维护" {
		t.Errorf("body = %q, want %q", data["body"], "系统将于今晚进行维护")
	}
	if data["url"].(string) != "https://example.com/notice" {
		t.Errorf("url = %q, want %q", data["url"], "https://example.com/notice")
	}
}

func TestAnnouncement_Disabled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(`
jwt:
  secret: this-is-a-long-enough-test-secret-key-32+
announcement:
  enabled: false
`), 0o644); err != nil {
		t.Fatal(err)
	}

	mgr, err := config.NewManager(path)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	handler := Announcement(mgr)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/announcement", nil)
	handler(w, r)

	resp := decodeResponse(t, w)
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Data is not a map")
	}
	if data["enabled"].(bool) != false {
		t.Error("enabled = true, want false")
	}
}

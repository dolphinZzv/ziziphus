package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"ziziphus/config"
)

func TestAppInfo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(`
app:
  name: "测试IM"
  headline: "即时通讯"
  env: "testing"
jwt:
  secret: this-is-a-long-enough-test-secret-key-32+
`), 0o644); err != nil {
		t.Fatal(err)
	}

	mgr, err := config.NewManager(path)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	handler := AppInfo(mgr)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/app/info", nil)
	handler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	resp := decodeResponse(t, w)
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Data is not a map")
	}
	if data["name"].(string) != "测试IM" {
		t.Errorf("name = %q, want %q", data["name"], "测试IM")
	}
	if data["headline"].(string) != "即时通讯" {
		t.Errorf("headline = %q, want %q", data["headline"], "即时通讯")
	}
	if data["env"].(string) != "testing" {
		t.Errorf("env = %q, want %q", data["env"], "testing")
	}
}

func TestAppInfo_Defaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(`
jwt:
  secret: this-is-a-long-enough-test-secret-key-32+
`), 0o644); err != nil {
		t.Fatal(err)
	}

	mgr, err := config.NewManager(path)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	handler := AppInfo(mgr)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/app/info", nil)
	handler(w, r)

	resp := decodeResponse(t, w)
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Data is not a map")
	}
	// Default values from setDefaults
	if data["name"].(string) != "Ziziphus" {
		t.Errorf("name = %q, want %q", data["name"], "Ziziphus")
	}
	if data["env"].(string) != "development" {
		t.Errorf("env = %q, want %q", data["env"], "development")
	}
}

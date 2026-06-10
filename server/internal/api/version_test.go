package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"siciv.space/agent/panda_ai/pkg/version"
)

func TestGetVersion(t *testing.T) {
	h := &Handlers{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/version", nil)
	h.GetVersion(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Data versionResp `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Data.Version != version.ServerVersion {
		t.Errorf("Version = %q, want %q", resp.Data.Version, version.ServerVersion)
	}
	if resp.Data.GitCommit != version.GitCommit {
		t.Errorf("GitCommit = %q, want %q", resp.Data.GitCommit, version.GitCommit)
	}
}

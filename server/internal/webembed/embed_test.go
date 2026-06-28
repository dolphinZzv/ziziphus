package webembed

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandler_ServesIndexAtRoot(t *testing.T) {
	h := Handler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<!doctype html") && !strings.Contains(body, "<html") {
		t.Error("response should contain HTML for index page")
	}
	if rec.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("Cache-Control = %q, want %q", rec.Header().Get("Cache-Control"), "no-cache")
	}
}

func TestHandler_RedirectsIndexHtmlToRoot(t *testing.T) {
	h := Handler()
	req := httptest.NewRequest(http.MethodGet, "/index.html", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	// Go's http.FileServer redirects /index.html → /
	if rec.Code != http.StatusMovedPermanently {
		t.Errorf("status = %d, want 301 redirect", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if loc != "/" && loc != "./" {
		t.Errorf("Location = %q, want redirect to / or ./", loc)
	}
}

func TestHandler_ServesExistingAsset(t *testing.T) {
	h := Handler()
	req := httptest.NewRequest(http.MethodGet, "/favicon.svg", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Existing assets should not have no-cache
	cc := rec.Header().Get("Cache-Control")
	if cc == "no-cache" {
		t.Errorf("expected asset to be cacheable, got Cache-Control: %q", cc)
	}
}

func TestHandler_SpaFallback(t *testing.T) {
	h := Handler()
	req := httptest.NewRequest(http.MethodGet, "/some/unknown/path", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 for SPA fallback", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<!doctype html") && !strings.Contains(body, "<html") {
		t.Error("SPA fallback should return index.html")
	}
	if rec.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("Cache-Control = %q, want %q", rec.Header().Get("Cache-Control"), "no-cache")
	}
}

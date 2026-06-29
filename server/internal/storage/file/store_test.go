package file

import (
	"testing"

	"github.com/spf13/afero"
)

func TestFullPath_PathTraversalBlocked(t *testing.T) {
	s := NewStore(afero.NewMemMapFs(), "/data/files")

	tests := []struct {
		name     string
		input    string
		wantSafe bool
	}{
		{"normal file", "avatar.jpg", true},
		{"subdirectory", "images/photo.png", true},
		{"dot dot traversal", "../../../etc/passwd", false},
		{"dot dot nested", "a/../../etc/shadow", false},
		{"just dot dots", "../etc/hosts", false},
		{"leading slash stripped", "/home/user/.bashrc", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.fullPath(tt.input)
			isBlocked := got == "/data/files/blocked"
			if tt.wantSafe && isBlocked {
				t.Errorf("fullPath(%q) = blocked; expected safe path", tt.input)
			}
			if !tt.wantSafe && !isBlocked {
				t.Errorf("fullPath(%q) = %q; expected blocked", tt.input, got)
			}
		})
	}
}

func TestFullPath_RejectsAllTraversals(t *testing.T) {
	s := NewStore(afero.NewMemMapFs(), "/opt/uploads")
	attacks := []string{
		"../../etc/passwd",
		"../config.yaml",
		"a/b/../../../secret",
	}
	for _, atk := range attacks {
		if got := s.fullPath(atk); got != "/opt/uploads/blocked" {
			t.Errorf("fullPath(%q) = %q; should be blocked", atk, got)
		}
	}
}

func TestFullPath_AllowsNormalPaths(t *testing.T) {
	s := NewStore(afero.NewMemMapFs(), "/data/files")
	for _, p := range []string{"f1.jpg", "sub/doc.pdf"} {
		if got := s.fullPath(p); got == "/data/files/blocked" {
			t.Errorf("fullPath(%q) was incorrectly blocked", p)
		}
	}
}

func TestFullPath_BasePath(t *testing.T) {
	s := NewStore(afero.NewMemMapFs(), "/data/files")
	if s.BasePath() != "/data/files" {
		t.Errorf("BasePath() = %q; want /data/files", s.BasePath())
	}
}

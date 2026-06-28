package file

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

func newTestStore() *Store {
	return NewStore(afero.NewMemMapFs(), "/base")
}

func TestSave(t *testing.T) {
	s := newTestStore()
	ctx := context.Background()

	n, err := s.Save(ctx, "foo/bar.txt", strings.NewReader("hello world"))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if n != 11 {
		t.Errorf("bytes written = %d, want 11", n)
	}

	// Verify it exists
	ok, err := s.Exists(ctx, "foo/bar.txt")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !ok {
		t.Error("file should exist after Save")
	}
}

func TestOpen(t *testing.T) {
	s := newTestStore()
	ctx := context.Background()

	content := "test content"
	_, err := s.Save(ctx, "test.txt", strings.NewReader(content))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	rc, err := s.Open(ctx, "test.txt")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if string(data) != content {
		t.Errorf("content = %q, want %q", string(data), content)
	}
}

func TestOpen_NonExistent(t *testing.T) {
	s := newTestStore()
	ctx := context.Background()

	_, err := s.Open(ctx, "nonexistent.txt")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestDelete(t *testing.T) {
	s := newTestStore()
	ctx := context.Background()

	_, err := s.Save(ctx, "delete_me.txt", strings.NewReader("data"))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	err = s.Delete(ctx, "delete_me.txt")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	ok, err := s.Exists(ctx, "delete_me.txt")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if ok {
		t.Error("file should not exist after Delete")
	}
}

func TestDelete_NonExistent(t *testing.T) {
	s := newTestStore()
	ctx := context.Background()

	err := s.Delete(ctx, "nonexistent.txt")
	if err == nil {
		t.Fatal("expected error for deleting non-existent file")
	}
}

func TestExists(t *testing.T) {
	s := newTestStore()
	ctx := context.Background()

	// Non-existent
	ok, err := s.Exists(ctx, "missing.txt")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if ok {
		t.Error("missing file should not exist")
	}

	// After saving
	_, err = s.Save(ctx, "present.txt", strings.NewReader("data"))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	ok, err = s.Exists(ctx, "present.txt")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !ok {
		t.Error("saved file should exist")
	}
}

func TestSize(t *testing.T) {
	s := newTestStore()
	ctx := context.Background()

	_, err := s.Save(ctx, "size_test.txt", strings.NewReader("12345"))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	size, err := s.Size(ctx, "size_test.txt")
	if err != nil {
		t.Fatalf("Size failed: %v", err)
	}
	if size != 5 {
		t.Errorf("size = %d, want 5", size)
	}
}

func TestSize_NonExistent(t *testing.T) {
	s := newTestStore()
	ctx := context.Background()

	_, err := s.Size(ctx, "nonexistent.txt")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestBasePath(t *testing.T) {
	s := newTestStore()
	if got := s.BasePath(); got != "/base" {
		t.Errorf("BasePath = %q, want %q", got, "/base")
	}
}

func Test_fullPath(t *testing.T) {
	s := newTestStore()

	tests := []struct {
		relative string
		want     string
	}{
		{"foo.txt", "/base/foo.txt"},
		{"/leading/slash.txt", "/base/leading/slash.txt"},
		{"sub/deep/file.txt", "/base/sub/deep/file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.relative, func(t *testing.T) {
			got := s.fullPath(tt.relative)
			if got != tt.want {
				t.Errorf("fullPath(%q) = %q, want %q", tt.relative, got, tt.want)
			}
		})
	}
}

func TestBaseDirCreated(t *testing.T) {
	// Use a sub-path to ensure MkdirAll happens during Save
	s := NewStore(afero.NewMemMapFs(), "/deeply/nested/base/path")
	ctx := context.Background()

	_, err := s.Save(ctx, "file.txt", strings.NewReader("data"))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify we can open what we saved
	rc, err := s.Open(ctx, "file.txt")
	if err != nil {
		t.Fatalf("Open after Save with nested base path failed: %v", err)
	}
	rc.Close()
}

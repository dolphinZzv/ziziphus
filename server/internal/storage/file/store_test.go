package file

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

func newTestStore(t *testing.T) (*Store, afero.Fs) {
	t.Helper()
	fs := afero.NewMemMapFs()
	s := NewStore(fs, "/data/files")
	return s, fs
}

func TestNewStore(t *testing.T) {
	s := NewStore(afero.NewMemMapFs(), "/base")
	if s.BasePath() != "/base" {
		t.Errorf("BasePath() = %q, want /base", s.BasePath())
	}
}

func TestStore_SaveAndOpen(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	n, err := s.Save(ctx, "test/hello.txt", strings.NewReader("hello world"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if n != 11 {
		t.Errorf("Save wrote %d bytes, want 11", n)
	}

	rc, err := s.Open(ctx, "test/hello.txt")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer rc.Close()
	var buf bytes.Buffer
	buf.ReadFrom(rc)
	if buf.String() != "hello world" {
		t.Errorf("Open content = %q, want %q", buf.String(), "hello world")
	}
}

func TestStore_Save_PathTraversal(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	_, err := s.Save(ctx, "../../etc/passwd", strings.NewReader("data"))
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
}

func TestStore_Delete(t *testing.T) {
	s, fs := newTestStore(t)
	ctx := context.Background()

	afero.WriteFile(fs, "/data/files/test.txt", []byte("data"), 0644)

	err := s.Delete(ctx, "test.txt")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	exists, _ := afero.Exists(fs, "/data/files/test.txt")
	if exists {
		t.Error("file still exists after Delete")
	}
}

func TestStore_Delete_PathTraversal(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	err := s.Delete(ctx, "../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
}

func TestStore_Exists(t *testing.T) {
	s, fs := newTestStore(t)
	ctx := context.Background()

	afero.WriteFile(fs, "/data/files/exists.txt", []byte("data"), 0644)

	ok, err := s.Exists(ctx, "exists.txt")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !ok {
		t.Error("Exists returned false for existing file")
	}

	ok, err = s.Exists(ctx, "missing.txt")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if ok {
		t.Error("Exists returned true for missing file")
	}
}

func TestStore_Size(t *testing.T) {
	s, fs := newTestStore(t)
	ctx := context.Background()

	afero.WriteFile(fs, "/data/files/sized.dat", []byte("1234567890"), 0644)

	sz, err := s.Size(ctx, "sized.dat")
	if err != nil {
		t.Fatalf("Size: %v", err)
	}
	if sz != 10 {
		t.Errorf("Size = %d, want 10", sz)
	}
}

func TestStore_Size_Missing(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	_, err := s.Size(ctx, "nope.dat")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestStore_ConvFileSpace(t *testing.T) {
	s, _ := newTestStore(t)
	got := s.ConvFileSpace("conv123")
	want := "/data/files/conv123"
	if got != want {
		t.Errorf("ConvFileSpace = %q, want %q", got, want)
	}
}

func TestStore_EnsureConvSpace(t *testing.T) {
	s, fs := newTestStore(t)

	err := s.EnsureConvSpace("conv456")
	if err != nil {
		t.Fatalf("EnsureConvSpace: %v", err)
	}

	exists, _ := afero.DirExists(fs, "/data/files/conv456")
	if !exists {
		t.Error("directory not created by EnsureConvSpace")
	}

	// Should be idempotent
	err = s.EnsureConvSpace("conv456")
	if err != nil {
		t.Fatalf("EnsureConvSpace (2nd call): %v", err)
	}
}

func TestStore_CreateAndListFolders(t *testing.T) {
	s, fs := newTestStore(t)
	ctx := context.Background()

	// Ensure conv space first
	afero.WriteFile(fs, "/data/files/conv1/.placeholder", []byte{}, 0644)

	err := s.CreateFolder(ctx, "conv1", "", "sub1")
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}

	err = s.CreateFolder(ctx, "conv1", "", "sub2")
	if err != nil {
		t.Fatalf("CreateFolder sub2: %v", err)
	}

	// Nested folder
	err = s.CreateFolder(ctx, "conv1", "sub1", "nested")
	if err != nil {
		t.Fatalf("CreateFolder nested: %v", err)
	}

	folders, err := s.ListFolders(ctx, "conv1", "")
	if err != nil {
		t.Fatalf("ListFolders: %v", err)
	}

	names := make(map[string]bool)
	for _, f := range folders {
		names[f.Name] = true
	}
	if !names["sub1"] {
		t.Error("ListFolders missing sub1")
	}
	if !names["sub2"] {
		t.Error("ListFolders missing sub2")
	}
	if names[".placeholder"] {
		t.Error("ListFolders returned .placeholder (not a directory)")
	}
}

func TestStore_ListFolders_NonExistent(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	folders, err := s.ListFolders(ctx, "nonexistent", "")
	if err != nil {
		t.Fatalf("ListFolders on nonexistent dir: %v", err)
	}
	if folders != nil {
		t.Errorf("expected nil for nonexistent dir, got %+v", folders)
	}
}

func TestStore_DeleteFolder(t *testing.T) {
	s, fs := newTestStore(t)
	ctx := context.Background()

	afero.WriteFile(fs, "/data/files/conv1/sub/a.txt", []byte("data"), 0644)
	afero.WriteFile(fs, "/data/files/conv1/sub/b.txt", []byte("data"), 0644)

	err := s.DeleteFolder(ctx, "conv1", "sub")
	if err != nil {
		t.Fatalf("DeleteFolder: %v", err)
	}

	exists, _ := afero.DirExists(fs, "/data/files/conv1/sub")
	if exists {
		t.Error("directory still exists after DeleteFolder")
	}
}

func TestStore_RenameFolder(t *testing.T) {
	s, fs := newTestStore(t)
	ctx := context.Background()

	afero.WriteFile(fs, "/data/files/conv1/old/a.txt", []byte("data"), 0644)

	err := s.RenameFolder(ctx, "conv1", "old", "new")
	if err != nil {
		t.Fatalf("RenameFolder: %v", err)
	}

	existsOld, _ := afero.DirExists(fs, "/data/files/conv1/old")
	if existsOld {
		t.Error("old directory still exists after RenameFolder")
	}
	existsNew, _ := afero.DirExists(fs, "/data/files/conv1/new")
	if !existsNew {
		t.Error("new directory not found after RenameFolder")
	}
}

func TestStore_MoveFolder(t *testing.T) {
	s, fs := newTestStore(t)
	ctx := context.Background()

	afero.WriteFile(fs, "/data/files/conv1/src/sub/a.txt", []byte("data"), 0644)
	fs.MkdirAll("/data/files/conv1/dst", 0755)

	err := s.MoveFolder(ctx, "conv1", "src/sub", "dst")
	if err != nil {
		t.Fatalf("MoveFolder: %v", err)
	}

	exists, _ := afero.Exists(fs, "/data/files/conv1/dst/sub/a.txt")
	if !exists {
		t.Error("moved file not found at destination")
	}
}

func TestStore_MoveFile(t *testing.T) {
	s, fs := newTestStore(t)
	ctx := context.Background()

	afero.WriteFile(fs, "/data/files/src.txt", []byte("data"), 0644)
	fs.MkdirAll("/data/files/dst", 0755)

	err := s.MoveFile(ctx, "src.txt", "dst/moved.txt")
	if err != nil {
		t.Fatalf("MoveFile: %v", err)
	}

	exists, _ := afero.Exists(fs, "/data/files/dst/moved.txt")
	if !exists {
		t.Error("moved file not found at destination")
	}
	existsOld, _ := afero.Exists(fs, "/data/files/src.txt")
	if existsOld {
		t.Error("source file still exists after MoveFile")
	}
}

func TestStore_ListFilesInFolder(t *testing.T) {
	s, fs := newTestStore(t)
	ctx := context.Background()

	afero.WriteFile(fs, "/data/files/conv1/a.txt", []byte("a"), 0644)
	afero.WriteFile(fs, "/data/files/conv1/b.txt", []byte("b"), 0644)
	fs.MkdirAll("/data/files/conv1/sub", 0755)

	files, err := s.ListFilesInFolder(ctx, "conv1", "")
	if err != nil {
		t.Fatalf("ListFilesInFolder: %v", err)
	}

	names := make(map[string]bool)
	for _, f := range files {
		names[f.Name] = true
	}
	if !names["a.txt"] {
		t.Error("ListFilesInFolder missing a.txt")
	}
	if !names["b.txt"] {
		t.Error("ListFilesInFolder missing b.txt")
	}
	if names["sub"] {
		t.Error("ListFilesInFolder returned sub directory as file")
	}
}

func TestStore_ListFilesInFolder_NonExistent(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	files, err := s.ListFilesInFolder(ctx, "nope", "")
	if err != nil {
		t.Fatalf("ListFilesInFolder on nonexistent: %v", err)
	}
	if files != nil {
		t.Errorf("expected nil for nonexistent, got %+v", files)
	}
}

func TestStore_WalkFiles(t *testing.T) {
	s, fs := newTestStore(t)

	afero.WriteFile(fs, "/data/files/conv1/a.txt", []byte("a"), 0644)
	afero.WriteFile(fs, "/data/files/conv1/sub/b.txt", []byte("b"), 0644)
	afero.WriteFile(fs, "/data/files/conv1/sub/c.txt", []byte("c"), 0644)

	var walked []string
	err := s.WalkFiles("conv1", func(relPath string, info os.FileInfo) error {
		walked = append(walked, relPath)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkFiles: %v", err)
	}

	if len(walked) != 3 {
		t.Fatalf("WalkFiles visited %d files, want 3", len(walked))
	}
	// Order isn't guaranteed
	m := make(map[string]bool)
	for _, w := range walked {
		m[w] = true
	}
	if !m["a.txt"] {
		t.Error("WalkFiles missing a.txt")
	}
	if !m["sub/b.txt"] {
		t.Error("WalkFiles missing sub/b.txt")
	}
	if !m["sub/c.txt"] {
		t.Error("WalkFiles missing sub/c.txt")
	}
}

func TestStore_RelPath(t *testing.T) {
	s, _ := newTestStore(t)

	cases := []struct{ input, want string }{
		{"/data/files/foo.txt", "foo.txt"},
		{"/data/files/sub/bar.txt", "sub/bar.txt"},
		{"/other/path.txt", "../../other/path.txt"},
	}
	for _, tc := range cases {
		got := s.RelPath(tc.input)
		if got != tc.want {
			t.Errorf("RelPath(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestSafeRelPath(t *testing.T) {
	s, _ := newTestStore(t)

	okCases := []string{
		"foo.txt",
		"sub/bar.txt",
		"a/b/c.txt",
		"",
	}
	for _, c := range okCases {
		got, err := s.safeRelPath(c)
		if err != nil {
			t.Errorf("safeRelPath(%q) unexpected error: %v", c, err)
		}
		// Should be cleaned but same logical path
		if c != "" && !strings.HasPrefix(got, "/") && !strings.Contains(got, "..") {
			// ok
		}
	}

	badCases := []string{
		"../etc/passwd",
		"a/../../etc/shadow",
	}
	for _, c := range badCases {
		_, err := s.safeRelPath(c)
		if err == nil {
			t.Errorf("safeRelPath(%q) expected error", c)
		}
	}
}

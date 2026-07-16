package file

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

// Store abstracts file storage using afero.
type Store struct {
	fs       afero.Fs
	basePath string
}

func NewStore(fs afero.Fs, basePath string) *Store {
	return &Store{fs: fs, basePath: basePath}
}

func (s *Store) Save(ctx context.Context, path string, r io.Reader) (int64, error) {
	if _, err := s.safeRelPath(path); err != nil {
		return 0, err
	}
	fullPath := s.fullPath(path)
	if err := s.fs.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return 0, err
	}
	f, err := s.fs.Create(fullPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return io.Copy(f, r)
}

func (s *Store) Open(ctx context.Context, path string) (io.ReadCloser, error) {
	if _, err := s.safeRelPath(path); err != nil {
		return nil, err
	}
	return s.fs.Open(s.fullPath(path))
}

func (s *Store) Delete(ctx context.Context, path string) error {
	if _, err := s.safeRelPath(path); err != nil {
		return err
	}
	return s.fs.Remove(s.fullPath(path))
}

func (s *Store) Exists(ctx context.Context, path string) (bool, error) {
	if _, err := s.safeRelPath(path); err != nil {
		return false, err
	}
	return afero.Exists(s.fs, s.fullPath(path))
}

func (s *Store) Size(ctx context.Context, path string) (int64, error) {
	if _, err := s.safeRelPath(path); err != nil {
		return 0, err
	}
	fi, err := s.fs.Stat(s.fullPath(path))
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}

func (s *Store) BasePath() string {
	return s.basePath
}

// --- Folder operations (filesystem-based) ---

type FolderInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	ModTime int64  `json:"mod_time"`
}

// ListFolders returns subdirectories under a given relative path.
func (s *Store) ListFolders(ctx context.Context, convID, parentPath string) ([]FolderInfo, error) {
	if _, err := s.safeRelPath(parentPath); err != nil {
		return nil, err
	}
	dirPath := filepath.Join(s.basePath, convID, parentPath)
	entries, err := afero.ReadDir(s.fs, dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var folders []FolderInfo
	for _, e := range entries {
		if e.IsDir() {
			folders = append(folders, FolderInfo{
				Name:    e.Name(),
				Path:    filepath.Join(parentPath, e.Name()),
				ModTime: e.ModTime().UnixMilli(),
			})
		}
	}
	return folders, nil
}

// CreateFolder creates a directory under the conv's file space.
func (s *Store) CreateFolder(ctx context.Context, convID, parentPath, name string) error {
	if _, err := s.safeRelPath(filepath.Join(parentPath, name)); err != nil {
		return err
	}
	dirPath := filepath.Join(s.basePath, convID, parentPath, name)
	return s.fs.MkdirAll(dirPath, 0o755)
}

// DeleteFolder removes a directory and all its contents.
func (s *Store) DeleteFolder(ctx context.Context, convID, folderPath string) error {
	if _, err := s.safeRelPath(folderPath); err != nil {
		return err
	}
	return s.fs.RemoveAll(filepath.Join(s.basePath, convID, folderPath))
}

// RenameFolder renames a directory.
func (s *Store) RenameFolder(ctx context.Context, convID, oldPath, newPath string) error {
	if _, err := s.safeRelPath(oldPath); err != nil {
		return err
	}
	if _, err := s.safeRelPath(newPath); err != nil {
		return err
	}
	return s.fs.Rename(
		filepath.Join(s.basePath, convID, oldPath),
		filepath.Join(s.basePath, convID, newPath),
	)
}

// MoveFolder moves a directory to a new parent.
func (s *Store) MoveFolder(ctx context.Context, convID, srcPath, dstParent string) error {
	if _, err := s.safeRelPath(srcPath); err != nil {
		return err
	}
	if _, err := s.safeRelPath(dstParent); err != nil {
		return err
	}
	name := filepath.Base(srcPath)
	return s.fs.Rename(
		filepath.Join(s.basePath, convID, srcPath),
		filepath.Join(s.basePath, convID, dstParent, name),
	)
}

// MoveFile moves a file to a different folder.
func (s *Store) MoveFile(ctx context.Context, srcRelPath, dstRelPath string) error {
	if _, err := s.safeRelPath(srcRelPath); err != nil {
		return err
	}
	if _, err := s.safeRelPath(dstRelPath); err != nil {
		return err
	}
	if err := s.fs.MkdirAll(filepath.Dir(s.fullPath(dstRelPath)), 0o755); err != nil {
		return err
	}
	return s.fs.Rename(s.fullPath(srcRelPath), s.fullPath(dstRelPath))
}

// ListFilesInFolder returns file entries directly under a folder path.
func (s *Store) ListFilesInFolder(ctx context.Context, convID, folderPath string) ([]FileInfo, error) {
	dirPath := filepath.Join(s.basePath, convID, folderPath)
	entries, err := afero.ReadDir(s.fs, dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var files []FileInfo
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, FileInfo{
				Name:    e.Name(),
				ModTime: e.ModTime().UnixMilli(),
			})
		}
	}
	return files, nil
}

// FileInfo is a minimal file entry from the filesystem.
type FileInfo struct {
	Name    string `json:"name"`
	ModTime int64  `json:"mod_time"`
}

// ConvFileSpace returns the filesystem path for a conversation's file root.
func (s *Store) ConvFileSpace(convID string) string {
	return filepath.Join(s.basePath, convID)
}

// EnsureConvSpace creates the conv's file root directory if needed.
func (s *Store) EnsureConvSpace(convID string) error {
	return s.fs.MkdirAll(filepath.Join(s.basePath, convID), 0o755)
}

// WalkFiles walks all non-directory entries under a conv's file space.
func (s *Store) WalkFiles(convID string, fn func(relPath string, info fs.FileInfo) error) error {
	root := filepath.Join(s.basePath, convID)
	return afero.Walk(s.fs, root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		return fn(rel, info)
	})
}

// RelPath returns the relative path from basePath for a full path.
func (s *Store) RelPath(fullPath string) string {
	rel, _ := filepath.Rel(s.basePath, fullPath)
	return rel
}

// safeRelPath validates that a relative user-supplied path does not contain
// directory traversal components ("..") and returns a cleaned relative path.
func (s *Store) safeRelPath(rel string) (string, error) {
	// Check raw input for ".." before filepath.Clean can resolve it.
	if strings.Contains(filepath.ToSlash(rel), "..") {
		return "", fmt.Errorf("path %q contains directory traversal", rel)
	}
	clean := filepath.Clean(rel)
	clean = strings.TrimPrefix(clean, "/")
	if filepath.IsAbs(clean) {
		return "", fmt.Errorf("path %q is absolute", rel)
	}
	// Check that the resolved path stays within the base directory.
	resolved := filepath.Join(s.basePath, clean)
	basePrefix := s.basePath + string(filepath.Separator)
	if !strings.HasPrefix(resolved, basePrefix) && resolved != s.basePath {
		return "", fmt.Errorf("path %q escapes base directory", rel)
	}
	return clean, nil
}

func (s *Store) fullPath(relative string) string {
	clean := filepath.Clean(relative)
	clean = strings.TrimPrefix(clean, "/")
	resolved := filepath.Join(s.basePath, clean)
	if strings.Contains(clean, "..") || (!strings.HasPrefix(resolved, s.basePath+string(filepath.Separator)) && resolved != s.basePath) {
		return filepath.Join(s.basePath, "blocked")
	}
	return resolved
}

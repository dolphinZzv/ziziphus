package file

import (
	"context"
	"io"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

// Store abstracts file storage using afero.
// Supports local filesystem (OsFs) and can be swapped with MemFs for tests
// or with afero-based S3/GCS backends in production.
type Store struct {
	fs       afero.Fs
	basePath string
}

func NewStore(fs afero.Fs, basePath string) *Store {
	return &Store{fs: fs, basePath: basePath}
}

func (s *Store) Save(ctx context.Context, path string, r io.Reader) (int64, error) {
	fullPath := s.fullPath(path)
	if err := s.fs.MkdirAll(s.basePath, 0755); err != nil {
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
	return s.fs.Open(s.fullPath(path))
}

func (s *Store) Delete(ctx context.Context, path string) error {
	return s.fs.Remove(s.fullPath(path))
}

func (s *Store) Exists(ctx context.Context, path string) (bool, error) {
	return afero.Exists(s.fs, s.fullPath(path))
}

func (s *Store) Size(ctx context.Context, path string) (int64, error) {
	fi, err := s.fs.Stat(s.fullPath(path))
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}

func (s *Store) BasePath() string {
	return s.basePath
}

func (s *Store) fullPath(relative string) string {
	clean := filepath.Clean(relative)
	clean = strings.TrimPrefix(clean, "/")
	return filepath.Join(s.basePath, clean)
}

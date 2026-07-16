package db

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"ziziphus/pkg/model"
)

func TestNewFileRepo(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewFileRepo(mock)
	if repo == nil {
		t.Fatal("NewFileRepo returned nil")
	}
}

func TestFileRepo_Insert(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewFileRepo(mock)
	f := &model.FileInfo{
		FileID:       "f1",
		UploaderID:   "u1",
		Name:         "photo.jpg",
		Size:         1024,
		ContentType:  0, // image
		Width:        800,
		Height:       600,
		URL:          "/files/f1.jpg",
		ThumbnailURL: "/files/f1_thumb.jpg",
		CreatedAt:    time.Now().UnixMilli(),
	}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO files (file_id, uploader_id, name, size, content_type, width, height, path, thumbnail_path, conv_id, folder_path, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`)).
		WithArgs(f.FileID, f.UploaderID, f.Name, f.Size, f.ContentType,
			f.Width, f.Height, f.URL, f.ThumbnailURL, f.ConvID, f.FolderPath, AnyTime{}).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = repo.Insert(context.Background(), f)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestFileRepo_Insert_Error(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewFileRepo(mock)

	mock.ExpectExec(`INSERT INTO files`).
		WithArgs("", "", "", int64(0), 0, int32(0), int32(0), "", "", "", "", AnyTime{}).
		WillReturnError(context.DeadlineExceeded)

	err = repo.Insert(context.Background(), &model.FileInfo{})
	if err == nil {
		t.Fatal("expected error for empty file insert")
	}
}

func TestFileRepo_GetByID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewFileRepo(mock)
	now := time.Now()

	rows := pgxmock.NewRows([]string{"file_id", "uploader_id", "name", "size", "content_type", "width", "height", "path", "thumbnail_path", "conv_id", "COALESCE(folder_path,'')", "created_at"}).
		AddRow("f1", "u1", "photo.jpg", int64(1024), 0, int32(800), int32(600), "/files/f1.jpg", "/files/f1_thumb.jpg", nil, "", now)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT file_id, uploader_id, name, size, content_type, width, height, path, thumbnail_path, conv_id, COALESCE(folder_path,''), created_at FROM files WHERE file_id = $1`)).
		WithArgs("f1").
		WillReturnRows(rows)

	got, err := repo.GetByID(context.Background(), "f1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.FileID != "f1" {
		t.Errorf("FileID = %q, want f1", got.FileID)
	}
	if got.Name != "photo.jpg" {
		t.Errorf("Name = %q, want photo.jpg", got.Name)
	}
	if got.CreatedAt != now.UnixMilli() {
		t.Errorf("CreatedAt = %d, want %d", got.CreatedAt, now.UnixMilli())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestFileRepo_GetByID_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewFileRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT file_id, uploader_id, name, size, content_type, width, height, path, thumbnail_path, conv_id, COALESCE(folder_path,''), created_at FROM files WHERE file_id = $1`)).
		WithArgs("nonexistent").
		WillReturnError(context.DeadlineExceeded)

	_, err = repo.GetByID(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

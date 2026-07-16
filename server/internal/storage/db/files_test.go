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

func TestFileRepo_ListByConvID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewFileRepo(mock)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM files WHERE conv_id = $1 AND folder_path = ''`)).
		WithArgs("conv_1").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))

	rows := pgxmock.NewRows([]string{"file_id", "uploader_id", "uploader_name", "name", "size", "content_type", "width", "height", "path", "thumbnail_path", "conv_id", "folder_path", "created_at"}).
		AddRow("f1", "u1", "Alice", "photo.jpg", int64(1024), 0, int32(800), int32(600), "/files/f1.jpg", "/files/f1_thumb.jpg", nil, "", now).
		AddRow("f2", "u2", "Bob", "doc.pdf", int64(2048), 1, int32(0), int32(0), "/files/f2.pdf", "", nil, "folder1", now)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT f.file_id, f.uploader_id, COALESCE(u.name, ''), f.name, f.size, f.content_type, f.width, f.height, f.path, f.thumbnail_path, f.conv_id, COALESCE(f.folder_path,''), f.created_at FROM files f LEFT JOIN users u ON u.id = f.uploader_id WHERE f.conv_id = $1 AND f.folder_path = '' ORDER BY f.created_at DESC LIMIT $2 OFFSET $3`)).
		WithArgs("conv_1", 10, 0).
		WillReturnRows(rows)

	files, total, err := repo.ListByConvID(context.Background(), "conv_1", 1, 10)
	if err != nil {
		t.Fatalf("ListByConvID: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2", len(files))
	}
	if files[0].FileID != "f1" {
		t.Errorf("files[0].FileID = %q, want f1", files[0].FileID)
	}
	if files[0].UploaderName != "Alice" {
		t.Errorf("files[0].UploaderName = %q, want Alice", files[0].UploaderName)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestFileRepo_ListByConvID_Empty(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewFileRepo(mock)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM files WHERE conv_id = \$1 AND folder_path = ''`).
		WithArgs("conv_empty").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	rows := pgxmock.NewRows([]string{"file_id", "uploader_id", "uploader_name", "name", "size", "content_type", "width", "height", "path", "thumbnail_path", "conv_id", "folder_path", "created_at"})
	mock.ExpectQuery(`SELECT f.file_id, f.uploader_id, COALESCE\(u.name, ''\), f.name`).
		WithArgs("conv_empty", 10, 0).
		WillReturnRows(rows)

	files, total, err := repo.ListByConvID(context.Background(), "conv_empty", 1, 10)
	if err != nil {
		t.Fatalf("ListByConvID: %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(files) != 0 {
		t.Errorf("got %d files, want 0", len(files))
	}
}

func TestFileRepo_DeleteByID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewFileRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM files WHERE file_id = $1 AND uploader_id = $2`)).
		WithArgs("f1", "u1").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = repo.DeleteByID(context.Background(), "f1", "u1")
	if err != nil {
		t.Fatalf("DeleteByID: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestFileRepo_ListFilesInFolder(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewFileRepo(mock)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM files WHERE conv_id = $1 AND folder_path = $2`)).
		WithArgs("conv_1", "folder1").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	rows := pgxmock.NewRows([]string{"file_id", "uploader_id", "uploader_name", "name", "size", "content_type", "width", "height", "path", "thumbnail_path", "conv_id", "folder_path", "created_at"}).
		AddRow("f1", "u1", "Alice", "photo.jpg", int64(1024), 0, int32(800), int32(600), "/files/f1.jpg", "/files/f1_thumb.jpg", nil, "folder1", now)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT f.file_id, f.uploader_id, COALESCE(u.name,''), f.name, f.size, f.content_type, f.width, f.height, f.path, f.thumbnail_path, f.conv_id, COALESCE(f.folder_path,''), f.created_at FROM files f LEFT JOIN users u ON u.id = f.uploader_id WHERE f.conv_id = $1 AND f.folder_path = $2 ORDER BY f.created_at DESC LIMIT $3 OFFSET $4`)).
		WithArgs("conv_1", "folder1", 20, 0).
		WillReturnRows(rows)

	files, total, err := repo.ListFilesInFolder(context.Background(), "conv_1", "folder1", 1, 20)
	if err != nil {
		t.Fatalf("ListFilesInFolder: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
}

func TestFileRepo_UpdateFolderPath(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewFileRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE files SET folder_path = $1 WHERE file_id = $2`)).
		WithArgs("new_folder", "f1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.UpdateFolderPath(context.Background(), "f1", "new_folder")
	if err != nil {
		t.Fatalf("UpdateFolderPath: %v", err)
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

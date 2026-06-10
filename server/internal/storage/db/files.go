package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"siciv.space/agent/panda_ai/pkg/model"
)

type FileRepo struct {
	pool *pgxpool.Pool
}

func NewFileRepo(pool *pgxpool.Pool) *FileRepo {
	return &FileRepo{pool: pool}
}

func (r *FileRepo) Insert(ctx context.Context, f *model.FileInfo) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO files (file_id, uploader_id, name, size, content_type, width, height, path, thumbnail_path, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		f.FileID, f.UploaderID, f.Name, f.Size, f.ContentType,
		f.Width, f.Height, f.URL, f.ThumbnailURL, time.UnixMilli(f.CreatedAt),
	)
	return err
}

func (r *FileRepo) GetByID(ctx context.Context, fileID string) (*model.FileInfo, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT file_id, uploader_id, name, size, content_type, width, height, path, thumbnail_path, created_at
		 FROM files WHERE file_id = $1`, fileID)
	var f model.FileInfo
	var createdAt time.Time
	err := row.Scan(&f.FileID, &f.UploaderID, &f.Name, &f.Size, &f.ContentType,
		&f.Width, &f.Height, &f.URL, &f.ThumbnailURL, &createdAt)
	if err != nil {
		return nil, err
	}
	f.CreatedAt = createdAt.UnixMilli()
	return &f, nil
}

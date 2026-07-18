package db

import (
	"context"
	"time"

	"ziziphus/pkg/model"
)

type FileRepo struct {
	pool DBPool
}

func NewFileRepo(pool DBPool) *FileRepo {
	return &FileRepo{pool: pool}
}

func (r *FileRepo) Insert(ctx context.Context, f *model.FileInfo) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO files (file_id, uploader_id, name, size, content_type, width, height, path, thumbnail_path, conv_id, folder_path, visibility, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		f.FileID, f.UploaderID, f.Name, f.Size, f.ContentType,
		f.Width, f.Height, f.URL, f.ThumbnailURL, f.ConvID, f.FolderPath, f.Visibility, time.UnixMilli(f.CreatedAt),
	)
	return err
}

func (r *FileRepo) GetByID(ctx context.Context, fileID string) (*model.FileInfo, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT file_id, uploader_id, name, size, content_type, width, height, path, thumbnail_path, conv_id, COALESCE(folder_path,''), COALESCE(visibility,'public'), created_at
		 FROM files WHERE file_id = $1`, fileID)
	var f model.FileInfo
	var createdAt time.Time
	var convID *string
	err := row.Scan(&f.FileID, &f.UploaderID, &f.Name, &f.Size, &f.ContentType,
		&f.Width, &f.Height, &f.URL, &f.ThumbnailURL, &convID, &f.FolderPath, &f.Visibility, &createdAt)
	if err != nil {
		return nil, err
	}
	f.CreatedAt = createdAt.UnixMilli()
	if convID != nil {
		f.ConvID = *convID
	}
	return &f, nil
}

func (r *FileRepo) ListByConvID(ctx context.Context, convID string, page, size int) ([]*model.FileInfo, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM files WHERE conv_id = $1 AND folder_path = ''`, convID).Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * size
	rows, err := r.pool.Query(ctx,
		`SELECT f.file_id, f.uploader_id, COALESCE(u.name, ''), f.name, f.size, f.content_type, f.width, f.height, f.path, f.thumbnail_path, f.conv_id, COALESCE(f.folder_path,''), COALESCE(f.visibility,'public'), f.created_at
		 FROM files f LEFT JOIN users u ON u.id = f.uploader_id
		 WHERE f.conv_id = $1 AND f.folder_path = '' ORDER BY f.created_at DESC LIMIT $2 OFFSET $3`, convID, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var files []*model.FileInfo
	for rows.Next() {
		var f model.FileInfo
		var createdAt time.Time
		var cid *string
		if err := rows.Scan(&f.FileID, &f.UploaderID, &f.UploaderName, &f.Name, &f.Size, &f.ContentType, &f.Width, &f.Height, &f.URL, &f.ThumbnailURL, &cid, &f.FolderPath, &f.Visibility, &createdAt); err != nil {
			return nil, 0, err
		}
		f.CreatedAt = createdAt.UnixMilli()
		if cid != nil {
			f.ConvID = *cid
		}
		files = append(files, &f)
	}
	return files, total, nil
}

func (r *FileRepo) DeleteByID(ctx context.Context, fileID, uploaderID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM files WHERE file_id = $1 AND uploader_id = $2`, fileID, uploaderID)
	return err
}

func (r *FileRepo) ListFilesInFolder(ctx context.Context, convID, folderPath string, page, size int) ([]*model.FileInfo, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM files WHERE conv_id = $1 AND folder_path = $2`, convID, folderPath).Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * size
	rows, err := r.pool.Query(ctx,
		`SELECT f.file_id, f.uploader_id, COALESCE(u.name,''), f.name, f.size, f.content_type, f.width, f.height, f.path, f.thumbnail_path, f.conv_id, COALESCE(f.folder_path,''), COALESCE(f.visibility,'public'), f.created_at
		 FROM files f LEFT JOIN users u ON u.id = f.uploader_id
		 WHERE f.conv_id = $1 AND f.folder_path = $2 ORDER BY f.created_at DESC LIMIT $3 OFFSET $4`, convID, folderPath, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var files []*model.FileInfo
	for rows.Next() {
		var ff model.FileInfo
		var ca time.Time
		var cid *string
		if err := rows.Scan(&ff.FileID, &ff.UploaderID, &ff.UploaderName, &ff.Name, &ff.Size, &ff.ContentType, &ff.Width, &ff.Height, &ff.URL, &ff.ThumbnailURL, &cid, &ff.FolderPath, &ff.Visibility, &ca); err != nil {
			return nil, 0, err
		}
		ff.CreatedAt = ca.UnixMilli()
		if cid != nil {
			ff.ConvID = *cid
		}
		files = append(files, &ff)
	}
	return files, total, nil
}

func (r *FileRepo) UpdateFolderPath(ctx context.Context, fileID, folderPath string) error {
	_, err := r.pool.Exec(ctx, `UPDATE files SET folder_path = $1 WHERE file_id = $2`, folderPath, fileID)
	return err
}

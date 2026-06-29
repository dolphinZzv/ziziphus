package db

import (
	"context"
	"time"

	"siciv.space/agent/panda_ai/pkg/model"
)

type FileRepo struct {
	pool DBPool
}

func NewFileRepo(pool DBPool) *FileRepo {
	return &FileRepo{pool: pool}
}

func (r *FileRepo) Insert(ctx context.Context, f *model.FileInfo) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO files (file_id, uploader_id, name, size, content_type, width, height, path, thumbnail_path, conv_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		f.FileID, f.UploaderID, f.Name, f.Size, f.ContentType,
		f.Width, f.Height, f.URL, f.ThumbnailURL, f.ConvID, time.UnixMilli(f.CreatedAt),
	)
	return err
}

func (r *FileRepo) GetByID(ctx context.Context, fileID string) (*model.FileInfo, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT file_id, uploader_id, name, size, content_type, width, height, path, thumbnail_path, conv_id, created_at
		 FROM files WHERE file_id = $1`, fileID)
	var f model.FileInfo
	var createdAt time.Time
	var convID *string
	err := row.Scan(&f.FileID, &f.UploaderID, &f.Name, &f.Size, &f.ContentType,
		&f.Width, &f.Height, &f.URL, &f.ThumbnailURL, &convID, &createdAt)
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
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM files WHERE conv_id = $1`, convID).Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * size
	rows, err := r.pool.Query(ctx,
		`SELECT f.file_id, f.uploader_id, COALESCE(u.name, ''), f.name, f.size, f.content_type, f.width, f.height, f.path, f.thumbnail_path, f.conv_id, f.created_at
		 FROM files f LEFT JOIN users u ON u.id = f.uploader_id
		 WHERE f.conv_id = $1 ORDER BY f.created_at DESC LIMIT $2 OFFSET $3`, convID, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var files []*model.FileInfo
	for rows.Next() {
		var f model.FileInfo
		var createdAt time.Time
		var cid *string
		if err := rows.Scan(&f.FileID, &f.UploaderID, &f.UploaderName, &f.Name, &f.Size, &f.ContentType, &f.Width, &f.Height, &f.URL, &f.ThumbnailURL, &cid, &createdAt); err != nil {
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

func (r *FileRepo) InsertToFolder(ctx context.Context, f *model.FileInfo, folderID int64) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO files (file_id, uploader_id, name, size, content_type, width, height, path, thumbnail_path, conv_id, folder_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		f.FileID, f.UploaderID, f.Name, f.Size, f.ContentType,
		f.Width, f.Height, f.URL, f.ThumbnailURL, f.ConvID, folderID, time.UnixMilli(f.CreatedAt),
	)
	return err
}

func (r *FileRepo) CreateFolder(ctx context.Context, folder *model.FileFolder) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx,
		`INSERT INTO file_folders (conv_id, name, parent_id, created_by) VALUES ($1,$2,$3,$4) RETURNING folder_id`,
		folder.ConvID, folder.Name, folder.ParentID, folder.CreatedBy).Scan(&id)
	return id, err
}

func (r *FileRepo) ListFolders(ctx context.Context, convID string, parentID int64) ([]*model.FileFolder, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT f.folder_id, f.conv_id, f.name, f.parent_id, COALESCE(u.name,''), f.created_at
		 FROM file_folders f LEFT JOIN users u ON u.id = f.created_by
		 WHERE f.conv_id = $1 AND f.parent_id = $2 ORDER BY f.name`, convID, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var folders []*model.FileFolder
	for rows.Next() {
		f := &model.FileFolder{}
		var ca time.Time
		if err := rows.Scan(&f.FolderID, &f.ConvID, &f.Name, &f.ParentID, &f.CreatedBy, &ca); err != nil {
			return nil, err
		}
		f.CreatedAt = ca.UnixMilli()
		folders = append(folders, f)
	}
	return folders, nil
}

func (r *FileRepo) ListFilesInFolder(ctx context.Context, convID string, folderID int64, page, size int) ([]*model.FileInfo, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM files WHERE conv_id=$1 AND folder_id=$2`, convID, folderID).Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * size
	rows, err := r.pool.Query(ctx,
		`SELECT f.file_id, f.uploader_id, COALESCE(u.name,''), f.name, f.size, f.content_type, f.width, f.height, f.path, f.thumbnail_path, f.conv_id, f.folder_id, f.created_at
		 FROM files f LEFT JOIN users u ON u.id = f.uploader_id
		 WHERE f.conv_id=$1 AND f.folder_id=$2 ORDER BY f.created_at DESC LIMIT $3 OFFSET $4`, convID, folderID, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var files []*model.FileInfo
	for rows.Next() {
		var ff model.FileInfo
		var ca time.Time
		if err := rows.Scan(&ff.FileID, &ff.UploaderID, &ff.UploaderName, &ff.Name, &ff.Size, &ff.ContentType, &ff.Width, &ff.Height, &ff.URL, &ff.ThumbnailURL, &ff.ConvID, &ff.FolderID, &ca); err != nil {
			return nil, 0, err
		}
		ff.CreatedAt = ca.UnixMilli()
		files = append(files, &ff)
	}
	return files, total, nil
}

func (r *FileRepo) DeleteFolder(ctx context.Context, folderID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM file_folders WHERE folder_id=$1`, folderID)
	return err
}

func (r *FileRepo) MoveFile(ctx context.Context, fileID string, folderID int64) error {
	_, err := r.pool.Exec(ctx, `UPDATE files SET folder_id=$1 WHERE file_id=$2`, folderID, fileID)
	return err
}

func (r *FileRepo) MoveFolder(ctx context.Context, folderID, parentID int64) error {
	_, err := r.pool.Exec(ctx, `UPDATE file_folders SET parent_id=$1 WHERE folder_id=$2`, parentID, folderID)
	return err
}

func (r *FileRepo) RenameFolder(ctx context.Context, folderID int64, name string) error {
	_, err := r.pool.Exec(ctx, `UPDATE file_folders SET name=$1 WHERE folder_id=$2`, name, folderID)
	return err
}

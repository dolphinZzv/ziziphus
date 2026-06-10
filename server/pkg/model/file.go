package model

// FileInfo represents an uploaded file's metadata.
type FileInfo struct {
	FileID       string `json:"file_id"`
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	Size         int64  `json:"size"`
	Name         string `json:"name"`
	ContentType  int    `json:"content_type"` // 0=image, 1=file, 2=audio, 3=video
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	UploaderID   string `json:"uploader_id"`
	CreatedAt    int64  `json:"created_at"`
}

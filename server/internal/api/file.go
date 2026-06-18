package api

import (
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"siciv.space/agent/panda_ai/internal/auth"
	"siciv.space/agent/panda_ai/internal/storage/file"
	"siciv.space/agent/panda_ai/pkg/i18n"
	"siciv.space/agent/panda_ai/pkg/logger"
	"siciv.space/agent/panda_ai/pkg/model"
)

type idGenerator interface {
	NextID() int64
}

type fileDB interface {
	Insert(ctx context.Context, f *model.FileInfo) error
	GetByID(ctx context.Context, fileID string) (*model.FileInfo, error)
}

type FileHandler struct {
	store   *file.Store
	fileDB  fileDB
	idGen   idGenerator
	baseURL string
}

func NewFileHandler(store *file.Store, fileDB fileDB, idGen idGenerator, baseURL string) *FileHandler {
	return &FileHandler{store: store, fileDB: fileDB, idGen: idGen, baseURL: baseURL}
}

type uploadResp struct {
	FileID       string `json:"file_id"`
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	Size         int64  `json:"size"`
	Name         string `json:"name"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
}

// Upload handles POST /api/v1/files/upload (multipart/form-data).
func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10MB max

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.file_too_large"))
		return
	}

	fileType := 1 // default: file
	switch r.FormValue("file_type") {
	case "0":
		fileType = 0 // image
	case "1":
		fileType = 1 // file
	case "2":
		fileType = 2 // audio
	case "3":
		fileType = 3 // video
	}

	uploadedFile, header, err := r.FormFile("file")
	if err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_file"))
		return
	}
	defer uploadedFile.Close()

	fileID := fmt.Sprintf("file_%d", h.idGen.NextID())
	ext := filepath.Ext(header.Filename)
	savePath := fileID + ext

	data, err := io.ReadAll(uploadedFile)
	if err != nil {
		logger.Error("read uploaded file failed", "error", err)
		BadRequest(w, r, i18n.T(r.Context(), "err.file_upload_failed"))
		return
	}

	if _, err := h.store.Save(r.Context(), savePath, strings.NewReader(string(data))); err != nil {
		logger.Error("save file failed", "error", err)
		BadRequest(w, r, i18n.T(r.Context(), "err.file_upload_failed"))
		return
	}

	url := h.baseURL + "/" + savePath

	// Image dimension detection
	var width, height int
	if fileType == 0 {
		w, h, err := decodeImageDimensions(data)
		if err == nil {
			width = w
			height = h
		}
	}

	// Store metadata
	userID := auth.UserFromCtx(r.Context())
	now := time.Now().UnixMilli()
	finfo := &model.FileInfo{
		FileID:      fileID,
		URL:         url,
		Size:        int64(len(data)),
		Name:        header.Filename,
		ContentType: fileType,
		Width:       width,
		Height:      height,
		UploaderID:  userID,
		CreatedAt:   now,
	}
	if err := h.fileDB.Insert(r.Context(), finfo); err != nil {
		logger.Error("save file metadata failed", "error", err)
	}

	JSON(w, uploadResp{
		FileID: fileID,
		URL:    url,
		Size:   finfo.Size,
		Name:   finfo.Name,
		Width:  width,
		Height: height,
	})
}

// GetInfo handles GET /api/v1/files/{file_id}.
func (h *FileHandler) GetInfo(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "file_id")
	finfo, err := h.fileDB.GetByID(r.Context(), fileID)
	if err != nil {
		NotFound(w, r)
		return
	}
	JSON(w, uploadResp{
		FileID:       finfo.FileID,
		URL:          finfo.URL,
		ThumbnailURL: finfo.ThumbnailURL,
		Size:         finfo.Size,
		Name:         finfo.Name,
		Width:        finfo.Width,
		Height:       finfo.Height,
	})
}

// ServeFile serves stored files via HTTP (GET /files/*).
func (h *FileHandler) ServeFile(w http.ResponseWriter, r *http.Request) {
	filePath := chi.URLParam(r, "*")
	rc, err := h.store.Open(r.Context(), filePath)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rc.Close()

	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	case ".webp":
		w.Header().Set("Content-Type", "image/webp")
	case ".mp4":
		w.Header().Set("Content-Type", "video/mp4")
	case ".mp3":
		w.Header().Set("Content-Type", "audio/mpeg")
	case ".pdf":
		w.Header().Set("Content-Type", "application/pdf")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Header().Set("Cache-Control", "public, max-age=86400")
	io.Copy(w, rc)
}

func decodeImageDimensions(data []byte) (int, int, error) {
	cfg, _, err := image.DecodeConfig(strings.NewReader(string(data)))
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

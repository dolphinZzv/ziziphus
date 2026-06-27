package api

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	xdraw "golang.org/x/image/draw"
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
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.file_too_large"))
		return
	}

	fileType := 1
	switch r.FormValue("file_type") {
	case "0":
		fileType = 0
	case "1":
		fileType = 1
	case "2":
		fileType = 2
	case "3":
		fileType = 3
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

	fileURL := h.baseURL + "/" + savePath

	var width, height int
	if fileType == 0 {
		w, h, err := decodeImageDimensions(data)
		if err == nil {
			width = w
			height = h
		}
	}

	userID := auth.UserFromCtx(r.Context())
	now := time.Now().UnixMilli()
	finfo := &model.FileInfo{
		FileID:      fileID,
		URL:         fileURL,
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
		URL:    fileURL,
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
// Supports ?w=xxx&h=xxx for on-the-fly image resizing with center crop.
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

	data, err := io.ReadAll(rc)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	ext := strings.ToLower(filepath.Ext(filePath))

	// Check for resize params
	wStr := r.URL.Query().Get("w")
	hStr := r.URL.Query().Get("h")
	if (wStr != "" || hStr != "") && isImageExt(ext) {
		tw, _ := strconv.Atoi(wStr)
		th, _ := strconv.Atoi(hStr)
		if tw > 0 || th > 0 {
			if resized, ct, err := resizeImage(data, tw, th, ext); err == nil {
				w.Header().Set("Content-Type", ct)
				w.Header().Set("Cache-Control", "public, max-age=2592000")
				w.Write(resized)
				return
			}
		}
	}

	w.Header().Set("Content-Type", contentTypeByExt(ext))
	w.Header().Set("Cache-Control", "public, max-age=2592000")
	w.Write(data)
}

func contentTypeByExt(ext string) string {
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".mp4":
		return "video/mp4"
	case ".mp3":
		return "audio/mpeg"
	case ".pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}

func isImageExt(ext string) bool {
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif":
		return true
	}
	return false
}

func decodeImageDimensions(data []byte) (int, int, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

// resizeImage resizes/crops data to target w×h.
// If only one dimension is given, the other scales proportionally.
// If both are given and aspect ratios differ, center-crop then scale.
func resizeImage(data []byte, tw, th int, ext string) ([]byte, string, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", err
	}

	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	if sw == 0 || sh == 0 {
		return data, contentTypeByExt(ext), nil
	}

	// Determine crop rect
	crop := sb
	if tw > 0 && th > 0 {
		sr := float64(sw) / float64(sh)
		dr := float64(tw) / float64(th)
		if sr > dr {
			cw := int(float64(sh) * dr)
			crop = image.Rect((sw-cw)/2, 0, (sw-cw)/2+cw, sh)
		} else if sr < dr {
			ch := int(float64(sw) / dr)
			crop = image.Rect(0, (sh-ch)/2, sw, (sh-ch)/2+ch)
		}
	} else if tw > 0 {
		th = int(float64(tw) * float64(sh) / float64(sw))
	} else if th > 0 {
		tw = int(float64(th) * float64(sw) / float64(sh))
	}

	// Crop
	var cropped image.Image
	if crop != sb {
		if sub, ok := src.(interface {
			SubImage(r image.Rectangle) image.Image
		}); ok {
			cropped = sub.SubImage(crop)
		} else {
			tmp := image.NewRGBA(image.Rect(0, 0, crop.Dx(), crop.Dy()))
			for y := 0; y < crop.Dy(); y++ {
				for x := 0; x < crop.Dx(); x++ {
					tmp.Set(x, y, src.At(crop.Min.X+x, crop.Min.Y+y))
				}
			}
			cropped = tmp
		}
	} else {
		cropped = src
	}

	// Scale
	dst := image.NewRGBA(image.Rect(0, 0, tw, th))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), cropped, cropped.Bounds(), draw.Src, nil)

	var buf bytes.Buffer
	if ext == ".png" || ext == ".gif" {
		if err := png.Encode(&buf, dst); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "image/png", nil
	}
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 85}); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), "image/jpeg", nil
}

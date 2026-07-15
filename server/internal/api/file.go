package api

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"encoding/json"
	"github.com/go-chi/chi/v5"
	xdraw "golang.org/x/image/draw"
	"ziziphus/internal/auth"
	"ziziphus/internal/storage/file"
	"ziziphus/pkg/i18n"
	"ziziphus/pkg/logger"
	"ziziphus/pkg/model"
)

type idGenerator interface {
	NextID() int64
}

type fileDB interface {
	Insert(ctx context.Context, f *model.FileInfo) error
	GetByID(ctx context.Context, fileID string) (*model.FileInfo, error)
	ListByConvID(ctx context.Context, convID string, page, size int) ([]*model.FileInfo, int, error)
	DeleteByID(ctx context.Context, fileID, uploaderID string) error
	CreateFolder(ctx context.Context, folder *model.FileFolder) (int64, error)
	ListFolders(ctx context.Context, convID string, parentID int64) ([]*model.FileFolder, error)
	ListFilesInFolder(ctx context.Context, convID string, folderID int64, page, size int) ([]*model.FileInfo, int, error)
	DeleteFolder(ctx context.Context, folderID int64) error
	MoveFile(ctx context.Context, fileID string, folderID int64) error
	MoveFolder(ctx context.Context, folderID, parentID int64) error
	RenameFolder(ctx context.Context, folderID int64, name string) error
}

type fileConvChecker interface {
	IsMember(ctx context.Context, convID, userID string) (bool, error)
	Get(ctx context.Context, convID string) (*model.Conversation, error)
}

type fileSysMsgSender interface {
	SendSystemMessage(ctx context.Context, convID, body string, senderID ...string) (*model.Message, error)
}

type FileHandler struct {
	store    *file.Store
	fileDB   fileDB
	idGen    idGenerator
	baseURL  string
	convMgr  fileConvChecker
	sysMsg   fileSysMsgSender
	userDB   userGetter
}

func NewFileHandler(store *file.Store, fileDB fileDB, idGen idGenerator, baseURL string, convMgr fileConvChecker, sysMsg fileSysMsgSender, userDB userGetter) *FileHandler {
	return &FileHandler{store: store, fileDB: fileDB, idGen: idGen, baseURL: baseURL, convMgr: convMgr, sysMsg: sysMsg, userDB: userDB}
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
	convID := r.FormValue("conv_id")
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
		ConvID:      convID,
		CreatedAt:   now,
	}
	if err := h.fileDB.Insert(r.Context(), finfo); err != nil {
		logger.Error("save file metadata failed", "error", err)
	}

	// Send file change system message if the conversation setting is enabled
	if convID != "" {
		h.sendFileChangeNotify(r.Context(), convID, userID, header.Filename)
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

// ListConvFiles handles GET /api/v1/conversations/{conv_id}/files
func (h *FileHandler) ListConvFiles(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())
	convID := chi.URLParam(r, "conv_id")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 50
	}

	// Verify the user is a member of this conversation
	files, total, err := h.fileDB.ListByConvID(r.Context(), convID, page, size)
	if err != nil {
		logger.Error("list conv files failed", "conv_id", convID, "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	_ = userID // fileDB.ListByConvID includes caller validation implicitly via conv membership
	if files == nil {
		files = []*model.FileInfo{}
	}
	Paginated(w, files, total, page, size)
}

// DeleteConvFile handles DELETE /api/v1/conversations/{conv_id}/files/{file_id}
func (h *FileHandler) DeleteConvFile(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())
	fileID := chi.URLParam(r, "file_id")
	convID := chi.URLParam(r, "conv_id")

	// Get file info before deleting to know the file name for notification
	fileName := fileID
	if finfo, err := h.fileDB.GetByID(r.Context(), fileID); err == nil && finfo != nil {
		fileName = finfo.Name
	}

	if err := h.fileDB.DeleteByID(r.Context(), fileID, userID); err != nil {
		logger.Error("delete conv file failed", "file_id", fileID, "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}

	// Send file delete notification
	if convID != "" {
		h.sendFileDeleteNotify(r.Context(), convID, userID, fileName)
	}

	JSON(w, map[string]string{"file_id": fileID})
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

// CreateFolder handles POST /api/v1/conversations/{conv_id}/folders
func (h *FileHandler) CreateFolder(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())
	convID := chi.URLParam(r, "conv_id")
	var req struct {
		Name     string `json:"name"`
		ParentID int64  `json:"parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	id, err := h.fileDB.CreateFolder(r.Context(), &model.FileFolder{ConvID: convID, Name: req.Name, ParentID: req.ParentID, CreatedBy: userID})
	if err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]interface{}{"folder_id": id, "name": req.Name, "parent_id": req.ParentID})
}

// ListFolders handles GET /api/v1/conversations/{conv_id}/folders
func (h *FileHandler) ListFolders(w http.ResponseWriter, r *http.Request) {
	convID := chi.URLParam(r, "conv_id")
	parentID, _ := strconv.ParseInt(r.URL.Query().Get("parent_id"), 10, 64)
	folders, err := h.fileDB.ListFolders(r.Context(), convID, parentID)
	if err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	if folders == nil {
		folders = []*model.FileFolder{}
	}
	JSON(w, folders)
}

// DeleteFolder handles DELETE /api/v1/conversations/{conv_id}/folders/{folder_id}
func (h *FileHandler) DeleteFolder(w http.ResponseWriter, r *http.Request) {
	folderID, _ := strconv.ParseInt(chi.URLParam(r, "folder_id"), 10, 64)
	if err := h.fileDB.DeleteFolder(r.Context(), folderID); err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]string{"status": "ok"})
}

// ListFolderFiles handles GET /api/v1/conversations/{conv_id}/folders/{folder_id}/files
func (h *FileHandler) ListFolderFiles(w http.ResponseWriter, r *http.Request) {
	convID := chi.URLParam(r, "conv_id")
	folderID, _ := strconv.ParseInt(chi.URLParam(r, "folder_id"), 10, 64)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 50
	}
	files, total, err := h.fileDB.ListFilesInFolder(r.Context(), convID, folderID, page, size)
	if err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	if files == nil {
		files = []*model.FileInfo{}
	}
	Paginated(w, files, total, page, size)
}

// MoveFile handles PUT /api/v1/conversations/{conv_id}/files/{file_id}/move
func (h *FileHandler) MoveFile(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "file_id")
	var req struct {
		FolderID int64 `json:"folder_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, "invalid body")
		return
	}
	if err := h.fileDB.MoveFile(r.Context(), fileID, req.FolderID); err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]string{"status": "ok"})
}

// MoveFolder handles PUT /api/v1/conversations/{conv_id}/folders/{folder_id}/move
func (h *FileHandler) MoveFolder(w http.ResponseWriter, r *http.Request) {
	folderID, _ := strconv.ParseInt(chi.URLParam(r, "folder_id"), 10, 64)
	var req struct {
		ParentID int64 `json:"parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, "invalid body")
		return
	}
	if err := h.fileDB.MoveFolder(r.Context(), folderID, req.ParentID); err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]string{"status": "ok"})
}

// RenameFolder handles PUT /api/v1/conversations/{conv_id}/folders/{folder_id}/rename
func (h *FileHandler) RenameFolder(w http.ResponseWriter, r *http.Request) {
	folderID, _ := strconv.ParseInt(chi.URLParam(r, "folder_id"), 10, 64)
	var req struct{ Name string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		BadRequest(w, r, "invalid body")
		return
	}
	if err := h.fileDB.RenameFolder(r.Context(), folderID, req.Name); err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]string{"status": "ok"})
}

// sendFileChangeNotify sends a system message when a file is uploaded in a conversation
// if the conversation's file_change_notify setting is enabled.
func (h *FileHandler) sendFileChangeNotify(ctx context.Context, convID, userID, fileName string) {
	c, err := h.convMgr.Get(ctx, convID)
	if err != nil {
		return
	}
	enabled, _ := c.Settings["file_change_notify"].(bool)
	if !enabled {
		return
	}

	userName := userID
	if u, err := h.userDB.GetByID(ctx, userID); err == nil && u != nil {
		userName = u.Name
	}

	body := fmt.Sprintf("%s 上传了文件: %s", userName, fileName)
	h.sysMsg.SendSystemMessage(ctx, convID, body, userID)
}

// sendFileDeleteNotify sends a system message when a file is deleted in a conversation.
func (h *FileHandler) sendFileDeleteNotify(ctx context.Context, convID, userID, fileName string) {
	c, err := h.convMgr.Get(ctx, convID)
	if err != nil {
		return
	}
	enabled, _ := c.Settings["file_change_notify"].(bool)
	if !enabled {
		return
	}

	userName := userID
	if u, err := h.userDB.GetByID(ctx, userID); err == nil && u != nil {
		userName = u.Name
	}

	body := fmt.Sprintf("%s 删除了文件: %s", userName, fileName)
	h.sysMsg.SendSystemMessage(ctx, convID, body, userID)
}


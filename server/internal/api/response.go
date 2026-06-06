package api

import (
	"encoding/json"
	"net/http"

	"github.com/dolphinz/im-server/pkg/model"
)

type APIResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

type PaginatedData struct {
	Items interface{} `json:"items"`
	Total int         `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
}

func JSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	resp := APIResponse{Code: 0, Msg: "ok", Data: data}
	json.NewEncoder(w).Encode(resp)
}

func Error(w http.ResponseWriter, httpStatus int, appErr *model.AppError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	resp := APIResponse{Code: appErr.Code, Msg: appErr.Message, Data: nil}
	json.NewEncoder(w).Encode(resp)
}

func Paginated(w http.ResponseWriter, items interface{}, total, page, size int) {
	JSON(w, PaginatedData{
		Items: items,
		Total: total,
		Page:  page,
		Size:  size,
	})
}

func BadRequest(w http.ResponseWriter, msg string) {
	Error(w, http.StatusBadRequest, model.NewAppError(model.ErrBadMessage, msg))
}

func NotFound(w http.ResponseWriter) {
	Error(w, http.StatusNotFound, &model.AppError{Code: model.ErrNotFound, Message: "资源不存在"})
}

func Unauthorized(w http.ResponseWriter) {
	Error(w, http.StatusUnauthorized, &model.AppError{Code: model.ErrNoPermission, Message: "未授权"})
}

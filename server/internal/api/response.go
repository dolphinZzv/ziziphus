package api

import (
	"encoding/json"
	"net/http"

	"ziziphus/pkg/i18n"
	"ziziphus/pkg/model"
)

type APIResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Key  string `json:"key,omitempty"`
	Data any    `json:"data"`
}

type PaginatedData struct {
	Items any `json:"items"`
	Total int `json:"total"`
	Page  int `json:"page"`
	Size  int `json:"size"`
}

func JSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	resp := APIResponse{Code: 0, Msg: "ok", Data: data}
	json.NewEncoder(w).Encode(resp)
}

func Error(w http.ResponseWriter, r *http.Request, httpStatus int, appErr *model.AppError) {
	if appErr.Key != "" {
		translated := i18n.T(r.Context(), appErr.Key)
		appErr = &model.AppError{Code: appErr.Code, Message: translated, Key: appErr.Key}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	resp := APIResponse{Code: appErr.Code, Msg: appErr.Message, Key: appErr.Key, Data: nil}
	json.NewEncoder(w).Encode(resp)
}

func Paginated(w http.ResponseWriter, items any, total, page, size int) {
	JSON(w, PaginatedData{
		Items: items,
		Total: total,
		Page:  page,
		Size:  size,
	})
}

func BadRequest(w http.ResponseWriter, r *http.Request, msg string) {
	Error(w, r, http.StatusBadRequest, model.NewAppError(model.ErrBadMessage, msg))
}

func NotFound(w http.ResponseWriter, r *http.Request) {
	Error(w, r, http.StatusNotFound, &model.AppError{Code: model.ErrNotFound, Message: "resource not found", Key: "err.resource_not_found"})
}

func Unauthorized(w http.ResponseWriter, r *http.Request) {
	Error(w, r, http.StatusUnauthorized, &model.AppError{Code: model.ErrNoPermission, Message: "unauthorized", Key: "err.unauthorized"})
}

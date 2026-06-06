package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/dolphinz/im-server/pkg/model"
)

type ctxKey string

const (
	CtxKeyUserID  ctxKey = "uid"
	CtxKeySession ctxKey = "session"
)

func UserFromCtx(ctx context.Context) string {
	uid, _ := ctx.Value(CtxKeyUserID).(string)
	return uid
}

func AuthMiddleware(service *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractBearerToken(r)
			if tokenStr == "" {
				writeAuthError(w)
				return
			}
			claims, err := service.ParseToken(tokenStr)
			if err != nil {
				writeAuthError(w)
				return
			}
			ctx := context.WithValue(r.Context(), CtxKeyUserID, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func WSAuthMiddleware(service *Service) func(ctx context.Context, token string) (context.Context, error) {
	return func(ctx context.Context, token string) (context.Context, error) {
		claims, err := service.ParseToken(token)
		if err != nil {
			return nil, err
		}
		return context.WithValue(ctx, CtxKeyUserID, claims.UserID), nil
	}
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return r.URL.Query().Get("token")
}

func writeAuthError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"code":4002,"msg":"未授权","data":null}`))
}

type HTTPResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func WriteJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if code != 0 {
		w.WriteHeader(http.StatusBadRequest)
	}
	resp := HTTPResponse{Code: code, Msg: "ok", Data: data}
	if code != 0 {
		if appErr, ok := data.(*model.AppError); ok {
			resp.Msg = appErr.Message
			resp.Code = appErr.Code
			resp.Data = nil
		} else {
			resp.Msg = "error"
		}
	}
	w.Write(marshalJSON(resp))
}

func marshalJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

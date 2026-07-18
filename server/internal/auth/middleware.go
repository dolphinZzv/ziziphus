package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"ziziphus/pkg/i18n"
	"ziziphus/pkg/model"
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

// tokenParser is satisfied by *Service and allows mocking in tests.
type tokenParser interface {
	ParseToken(tokenStr string) (*Claims, error)
}

// apiKeyLookup is satisfied by *db.UserRepo.
type apiKeyLookup interface {
	GetByAPIKey(ctx context.Context, apiKey string) (*model.User, error)
}

func AuthMiddleware(service tokenParser) func(http.Handler) http.Handler {
	return AuthMiddlewareWithAPIKey(service, nil)
}

func AuthMiddlewareWithAPIKey(service tokenParser, keyLookup apiKeyLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractBearerToken(r)
			if tokenStr == "" {
				writeAuthError(w, r)
				return
			}
			claims, err := service.ParseToken(tokenStr)
			if err != nil {
				// JWT failed, try API key lookup
				if keyLookup != nil && strings.HasPrefix(tokenStr, "sk-") {
					user, lookupErr := keyLookup.GetByAPIKey(r.Context(), tokenStr)
					if lookupErr == nil && user != nil {
						ctx := context.WithValue(r.Context(), CtxKeyUserID, user.ID)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
				writeAuthError(w, r)
				return
			}
			ctx := context.WithValue(r.Context(), CtxKeyUserID, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func WSAuthMiddleware(service tokenParser, keyLookup apiKeyLookup) func(ctx context.Context, token string) (context.Context, error) {
	return func(ctx context.Context, token string) (context.Context, error) {
		claims, err := service.ParseToken(token)
		if err == nil {
			return context.WithValue(ctx, CtxKeyUserID, claims.UserID), nil
		}
		// JWT failed, try API key lookup
		if keyLookup != nil && strings.HasPrefix(token, "sk-") {
			user, lookupErr := keyLookup.GetByAPIKey(ctx, token)
			if lookupErr == nil && user != nil {
				return context.WithValue(ctx, CtxKeyUserID, user.ID), nil
			}
		}
		return nil, err
	}
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

func writeAuthError(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	msg := i18n.TWithLang(i18n.DetectLanguage(r), "err.unauthorized")
	json.NewEncoder(w).Encode(map[string]any{
		"code": model.ErrNoPermission,
		"msg":  msg,
		"data": nil,
	})
}

type HTTPResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

func WriteJSON(w http.ResponseWriter, code int, data any) {
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
	_, _ = w.Write(marshalJSON(resp))
}

func marshalJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

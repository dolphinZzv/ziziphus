package i18n

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// Lang represents a supported language.
type Lang string

const (
	LangZH Lang = "zh-Hans"
	LangEN Lang = "en"
)

type ctxKey string

const langCtxKey ctxKey = "lang"

// DetectLanguage parses the Accept-Language header and returns the best match.
// Defaults to LangZH if no supported language is found.
func DetectLanguage(r *http.Request) Lang {
	accept := r.Header.Get("Accept-Language")
	if accept == "" {
		return LangZH
	}
	// Simple parser: take the first language tag
	lang := strings.Split(accept, ",")[0]
	lang = strings.TrimSpace(lang)
	lang = strings.Split(lang, ";")[0]
	lang = strings.Split(lang, "-")[0] // "zh-CN" -> "zh"
	switch lang {
	case "en":
		return LangEN
	default:
		return LangZH
	}
}

// WithLang stores the language in the context.
func WithLang(ctx context.Context, lang Lang) context.Context {
	return context.WithValue(ctx, langCtxKey, lang)
}

// LangFromCtx retrieves the language from the context.
// Returns LangZH if not set.
func LangFromCtx(ctx context.Context) Lang {
	lang, _ := ctx.Value(langCtxKey).(Lang)
	if lang == "" {
		return LangZH
	}
	return lang
}

// T translates the given key using the language from context.
// Supports format args via fmt.Sprintf.
func T(ctx context.Context, key string, args ...interface{}) string {
	lang := LangFromCtx(ctx)
	entry, ok := Messages[key]
	if !ok {
		return key
	}
	msg, ok := entry[lang]
	if !ok {
		msg, ok = entry[LangZH]
		if !ok {
			return key
		}
	}
	if len(args) == 0 {
		return msg
	}
	return fmt.Sprintf(msg, args...)
}

// TWithLang translates the given key with an explicit language.
func TWithLang(lang Lang, key string, args ...interface{}) string {
	entry, ok := Messages[key]
	if !ok {
		return key
	}
	msg, ok := entry[lang]
	if !ok {
		msg, ok = entry[LangZH]
		if !ok {
			return key
		}
	}
	if len(args) == 0 {
		return msg
	}
	return fmt.Sprintf(msg, args...)
}

// Middleware detects the request language and stores it in the context.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := DetectLanguage(r)
		ctx := WithLang(r.Context(), lang)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

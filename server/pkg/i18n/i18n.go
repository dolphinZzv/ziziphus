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
	LangJA Lang = "ja"
	LangFR Lang = "fr"
	LangDE Lang = "de"
	LangES Lang = "es"
	LangKO Lang = "ko"
	LangRU Lang = "ru"
)

type ctxKey string

const langCtxKey ctxKey = "lang"

// ParseLang converts a language code string to a Lang constant.
// Accepts both short codes (zh, en, ja, fr, de, es, ko, ru) and
// canonical server codes (zh-Hans, en, ja, fr, de, es, ko, ru).
// Returns LangZH for unrecognized codes.
func ParseLang(code string) Lang {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "zh", "zh-cn", "zh-hans", "zh-hk", "zh-tw":
		return LangZH
	case "en", "en-us", "en-gb":
		return LangEN
	case "ja", "ja-jp":
		return LangJA
	case "fr", "fr-fr":
		return LangFR
	case "de", "de-de":
		return LangDE
	case "es", "es-es":
		return LangES
	case "ko", "ko-kr":
		return LangKO
	case "ru", "ru-ru":
		return LangRU
	default:
		return LangZH
	}
}

// DetectLanguage detects the language from the request.
// Priority: X-Language header (set by frontend) > Accept-Language header.
// Defaults to LangZH if no supported language is found.
func DetectLanguage(r *http.Request) Lang {
	// Frontend sends language via X-Language header (e.g. "en", "zh", "ja")
	if xlang := r.Header.Get("X-Language"); xlang != "" {
		return ParseLang(xlang)
	}

	accept := r.Header.Get("Accept-Language")
	if accept == "" {
		return LangZH
	}
	// Simple parser: take the first language tag
	lang := strings.Split(accept, ",")[0]
	lang = strings.TrimSpace(lang)
	lang = strings.Split(lang, ";")[0]
	return ParseLang(lang)
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

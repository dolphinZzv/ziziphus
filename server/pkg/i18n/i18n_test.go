package i18n

import (
	"context"
	"net/http"
	"testing"
)

func TestParseLang(t *testing.T) {
	tests := []struct {
		name string
		code string
		want Lang
	}{
		{"zh short code", "zh", LangZH},
		{"zh-Hans canonical", "zh-Hans", LangZH},
		{"zh-CN", "zh-CN", LangZH},
		{"en", "en", LangEN},
		{"en-US", "en-US", LangEN},
		{"en-GB", "en-GB", LangEN},
		{"ja", "ja", LangJA},
		{"ja-JP", "ja-JP", LangJA},
		{"fr", "fr", LangFR},
		{"fr-FR", "fr-FR", LangFR},
		{"de", "de", LangDE},
		{"de-DE", "de-DE", LangDE},
		{"es", "es", LangES},
		{"es-ES", "es-ES", LangES},
		{"ko", "ko", LangKO},
		{"ko-KR", "ko-KR", LangKO},
		{"ru", "ru", LangRU},
		{"ru-RU", "ru-RU", LangRU},
		{"unknown falls back to zh", "xx", LangZH},
		{"empty falls back to zh", "", LangZH},
		{"case insensitive", "EN", LangEN},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseLang(tt.code); got != tt.want {
				t.Errorf("ParseLang(%q) = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

func TestDetectLanguage_FromXLanguage(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   Lang
	}{
		{"X-Language en", "en", LangEN},
		{"X-Language zh", "zh", LangZH},
		{"X-Language ja", "ja", LangJA},
		{"X-Language fr", "fr", LangFR},
		{"X-Language de", "de", LangDE},
		{"X-Language es", "es", LangES},
		{"X-Language ko", "ko", LangKO},
		{"X-Language ru", "ru", LangRU},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{Header: http.Header{}}
			r.Header.Set("X-Language", tt.header)
			if got := DetectLanguage(r); got != tt.want {
				t.Errorf("DetectLanguage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectLanguage_XLanguageTakesPriority(t *testing.T) {
	r := &http.Request{Header: http.Header{}}
	r.Header.Set("X-Language", "fr")
	r.Header.Set("Accept-Language", "en-US,en;q=0.9")
	if got := DetectLanguage(r); got != LangFR {
		t.Errorf("DetectLanguage() = %v, want %v (X-Language should win)", got, LangFR)
	}
}

func TestDetectLanguage_FromAcceptLanguage(t *testing.T) {
	tests := []struct {
		name   string
		accept string
		want   Lang
	}{
		{name: "empty header defaults to zh", accept: "", want: LangZH},
		{name: "en first", accept: "en-US,en;q=0.9", want: LangEN},
		{name: "en only", accept: "en", want: LangEN},
		{name: "zh default", accept: "zh-CN,zh;q=0.8", want: LangZH},
		{name: "ja", accept: "ja-JP", want: LangJA},
		{name: "fr", accept: "fr-FR,fr;q=0.9", want: LangFR},
		{name: "de", accept: "de-DE,de;q=0.9", want: LangDE},
		{name: "es", accept: "es-ES", want: LangES},
		{name: "ko", accept: "ko-KR", want: LangKO},
		{name: "ru", accept: "ru-RU", want: LangRU},
		{name: "multiple with en", accept: "fr,en;q=0.8,zh;q=0.5", want: LangFR},
		{name: "only xx falls back", accept: "xx-XX", want: LangZH},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{Header: http.Header{}}
			r.Header.Set("Accept-Language", tt.accept)
			if got := DetectLanguage(r); got != tt.want {
				t.Errorf("DetectLanguage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithLang_LangFromCtx(t *testing.T) {
	ctx := context.Background()

	if got := LangFromCtx(ctx); got != LangZH {
		t.Errorf("LangFromCtx() default = %v, want %v", got, LangZH)
	}

	ctx = WithLang(ctx, LangEN)
	if got := LangFromCtx(ctx); got != LangEN {
		t.Errorf("LangFromCtx() = %v, want %v", got, LangEN)
	}

	ctx = WithLang(ctx, LangJA)
	if got := LangFromCtx(ctx); got != LangJA {
		t.Errorf("LangFromCtx() = %v, want %v", got, LangJA)
	}

	ctx = WithLang(ctx, LangRU)
	if got := LangFromCtx(ctx); got != LangRU {
		t.Errorf("LangFromCtx() = %v, want %v", got, LangRU)
	}
}

func TestT_ExistingKey(t *testing.T) {
	ctx := WithLang(context.Background(), LangEN)
	got := T(ctx, "auth.user_not_found")
	if got != "User not found" {
		t.Errorf("T() = %q, want %q", got, "User not found")
	}

	ctx = WithLang(context.Background(), LangZH)
	got = T(ctx, "auth.user_not_found")
	if got != "用户不存在" {
		t.Errorf("T() = %q, want %q", got, "用户不存在")
	}
}

func TestT_WithArgs(t *testing.T) {
	ctx := WithLang(context.Background(), LangEN)
	got := T(ctx, "sys.group_created", "Alice", "My Group")
	if got != "Alice created the group \"My Group\"" {
		t.Errorf("T() = %q, want %q", got, "Alice created the group \"My Group\"")
	}
}

func TestT_MissingKey(t *testing.T) {
	ctx := WithLang(context.Background(), LangEN)
	got := T(ctx, "nonexistent.key")
	if got != "nonexistent.key" {
		t.Errorf("T() = %q, want the key itself", got)
	}
}

func TestT_KeyMissingInLangFallsBackToZH(t *testing.T) {
	Messages["test.zh_only"] = map[Lang]string{LangZH: "中文"}
	defer delete(Messages, "test.zh_only")

	ctx := WithLang(context.Background(), LangEN)
	got := T(ctx, "test.zh_only")
	if got != "中文" {
		t.Errorf("T() = %q, want the zh fallback", got)
	}
}

func TestT_NoTranslationForKey(t *testing.T) {
	Messages["test.empty"] = map[Lang]string{}
	defer delete(Messages, "test.empty")

	ctx := WithLang(context.Background(), LangEN)
	got := T(ctx, "test.empty")
	if got != "test.empty" {
		t.Errorf("T() = %q, want the key itself", got)
	}
}

func TestTWithLang(t *testing.T) {
	got := TWithLang(LangEN, "auth.wrong_password")
	if got != "Wrong password" {
		t.Errorf("TWithLang() = %q, want %q", got, "Wrong password")
	}

	got = TWithLang(LangZH, "auth.wrong_password")
	if got != "密码错误" {
		t.Errorf("TWithLang() = %q, want %q", got, "密码错误")
	}
}

func TestTWithLang_MissingKey(t *testing.T) {
	got := TWithLang(LangEN, "no.such.key")
	if got != "no.such.key" {
		t.Errorf("TWithLang() = %q, want the key itself", got)
	}
}

func TestTWithLang_FallbackToZH(t *testing.T) {
	Messages["test.fallback"] = map[Lang]string{LangZH: "中文"}
	defer delete(Messages, "test.fallback")

	got := TWithLang(LangEN, "test.fallback")
	if got != "中文" {
		t.Errorf("TWithLang() = %q, want zh fallback", got)
	}
}

func TestTWithLang_WithArgs(t *testing.T) {
	got := TWithLang(LangZH, "sys.group_created", "Bob", "我的群")
	if got != "Bob 创建了群「我的群」" {
		t.Errorf("TWithLang() = %q, want %q", got, "Bob 创建了群「我的群」")
	}
}

func TestMiddleware_StoresLangInContext(t *testing.T) {
	tests := []struct {
		name  string
		xlang string
		want  Lang
	}{
		{"X-Language en", "en", LangEN},
		{"X-Language ja", "ja", LangJA},
		{"X-Language ru", "ru", LangRU},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{Header: http.Header{}}
			r.Header.Set("X-Language", tt.xlang)
			handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				lang := LangFromCtx(r.Context())
				if lang != tt.want {
					t.Errorf("lang = %v, want %v", lang, tt.want)
				}
			}))
			handler.ServeHTTP(nil, r)
		})
	}
}

func TestMiddleware_DefaultLang(t *testing.T) {
	r := &http.Request{Header: http.Header{}} // no headers
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := LangFromCtx(r.Context())
		if lang != LangZH {
			t.Errorf("lang = %v, want %v", lang, LangZH)
		}
	}))
	handler.ServeHTTP(nil, r)
}

func TestParseLang_RoundTrip(t *testing.T) {
	// Verify that ParseLang can handle what langToFrontendCode (from api) would send back
	codes := []string{"zh", "en", "ja", "fr", "de", "es", "ko", "ru"}
	for _, code := range codes {
		t.Run(code, func(t *testing.T) {
			parsed := ParseLang(code)
			if parsed == LangZH && code != "zh" {
				t.Errorf("ParseLang(%q) fell back to zh unexpectedly", code)
			}
			// Verify it round-trips: ParseLang can re-parse its own output
			rt := ParseLang(string(parsed))
			if rt != parsed {
				t.Errorf("Round-trip ParseLang(ParseLang(%q)) = %v, want %v", code, rt, parsed)
			}
		})
	}
}

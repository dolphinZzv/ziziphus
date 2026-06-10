package i18n

import (
	"context"
	"net/http"
	"testing"
)

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		name   string
		accept string
		want   Lang
	}{
		{name: "empty header defaults to zh", accept: "", want: LangZH},
		{name: "en first", accept: "en-US,en;q=0.9", want: LangEN},
		{name: "en only", accept: "en", want: LangEN},
		{name: "zh default", accept: "zh-CN,zh;q=0.8", want: LangZH},
		{name: "ja falls back to zh", accept: "ja-JP", want: LangZH},
		{name: "multiple with en", accept: "fr,en;q=0.8,zh;q=0.5", want: LangZH},
		{name: "complex quality", accept: "en;q=0.1,zh;q=0.9", want: LangEN},
		{name: "only fr falls back", accept: "fr-FR", want: LangZH},
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

	// Default is LangZH
	if got := LangFromCtx(ctx); got != LangZH {
		t.Errorf("LangFromCtx() = %v, want %v", got, LangZH)
	}

	ctx = WithLang(ctx, LangEN)
	if got := LangFromCtx(ctx); got != LangEN {
		t.Errorf("LangFromCtx() = %v, want %v", got, LangEN)
	}

	ctx = WithLang(ctx, LangZH)
	if got := LangFromCtx(ctx); got != LangZH {
		t.Errorf("LangFromCtx() = %v, want %v", got, LangZH)
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
	// Add a key that only has zh translation
	Messages["test.zh_only"] = map[Lang]string{LangZH: "中文"}
	defer delete(Messages, "test.zh_only")

	ctx := WithLang(context.Background(), LangEN)
	got := T(ctx, "test.zh_only")
	if got != "中文" {
		t.Errorf("T() = %q, want the zh fallback", got)
	}
}

func TestT_NoTranslationForKey(t *testing.T) {
	// Add a key with an empty translation map
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
	r := &http.Request{Header: http.Header{}}
	r.Header.Set("Accept-Language", "en")
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := LangFromCtx(r.Context())
		if lang != LangEN {
			t.Errorf("lang = %v, want %v", lang, LangEN)
		}
	}))
	handler.ServeHTTP(nil, r)
}

func TestMiddleware_DefaultLang(t *testing.T) {
	r := &http.Request{Header: http.Header{}} // no Accept-Language
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := LangFromCtx(r.Context())
		if lang != LangZH {
			t.Errorf("lang = %v, want %v", lang, LangZH)
		}
	}))
	handler.ServeHTTP(nil, r)
}

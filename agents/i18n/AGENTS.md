# i18n Agent

## Overview

Handles internationalization across Go backend (REST API + WebSocket system messages) and React web frontend. Supports 8 languages: zh, en, ja, fr, de, es, ko, ru.

## Structure

```
pkg/i18n/
  i18n.go              # Lang constants, ParseLang, DetectLanguage, T/TWithLang, Middleware
  i18n_test.go         # 22+ test cases
  messages.go          # Message key vars + registerLang helper
  {zh,en,ja,fr,de,es,ko,ru}.go   # Per-language translations via init()+registerLang()

internal/api/language.go         # langToFrontendCode() mapping + DetectLanguage handler
internal/auth/email_templates/   # verify_code_{lang}.html, reset_password_{lang}.html
internal/auth/mailer.go          # //go:embed + emailTemplates map + subject translations
server/web/src/i18n/
  index.ts             # i18next config with lazy loading
  {zh,en,ja,fr,de,es,ko,ru}.json  # Frontend translation key-value files
server/web/src/stores/ui-store.ts  # Language type + resolveAutoLang()
```

## Key Conventions

- Backend translation files use `init() + registerLang()` pattern (each file self-registers)
- `ParseLang()` normalizes browser locale codes (zh-CN, en-US, ja-JP) to Lang constants
- `DetectLanguage()` priority: X-Language header > Accept-Language > LangZH
- Frontend lazy-loads non-zh bundles via dynamic `import()` on language change
- Email templates embedded via `//go:embed` at compile time
- Language preference stored in localStorage key `ziziphus_language`
- Frontend sends `X-Language` header on every request

## Adding a Language

**Backend:** Add LangXX constant + ParseLang mapping + {lang}.go with all message keys + langToFrontendCode()

**Frontend:** Create {lang}.json + add to language selector + update Language type in ui-store

**Email:** Copy template `en` → `{lang}` + translate + add //go:embed + register in emailTemplates map + add subject translations

## Sync Procedures

Run these steps whenever translations are added or modified.

### 1. Backend message keys

```bash
# Extract keys from each language file and compare
for f in server/pkg/i18n/{zh,en,ja,fr,de,es,ko,ru}.go; do
  echo "$(basename $f): $(grep -cE '^\s+"[a-z]' "$f") keys"
  # Show missing vs zh.go
  comm -23 <(grep -oE '"[a-z.]+"' server/pkg/i18n/zh.go | sort) <(grep -oE '"[a-z.]+"' "$f" | sort)
done
```

**Fix**: If a language file is missing keys, add them to its `init() -> registerLang()` call. Copy the key from `en.go`, translate the value. Verify `%s` format specifiers match the English version.

**Verify**: `go build ./...` and `go test ./pkg/i18n/`

### 2. Frontend locale files

```bash
# Deep-compare all leaf keys across all 8 locale files
python3 -c "
import json
langs = ['zh','en','ja','fr','de','es','ko','ru']
def flatten(d, p=''):
    r = {}
    for k,v in d.items():
        kk = f'{p}.{k}' if p else k
        if isinstance(v,dict): r.update(flatten(v,kk))
        else: r[kk]=v
    return r
keys = {}
for l in langs:
    with open(f'server/web/src/i18n/{l}.json') as f:
        keys[l] = set(flatten(json.load(f)).keys())
ref = keys['zh']
for l in langs:
    missing = ref - keys[l]
    if missing: print(f'❌ {l} MISSING: {sorted(missing)}')
    extra = keys[l] - ref
    if extra: print(f'❌ {l} EXTRA: {sorted(extra)}')
ok = all(keys[l]==ref for l in langs)
print('✅ All synced' if ok else '❌ Mismatch found')
"
```

**Fix**: For each missing key, add the entry to the corresponding section in `{lang}.json`. Use `en.json` as reference for the value, then translate.

**Verify**: `cd server/web && npm run build` (or check no i18n errors)

### 3. Email templates

```bash
# Check all template files exist
for tmpl in verify_code reset_password; do
  for lang in zh en ja fr de es ko ru; do
    f="server/internal/auth/email_templates/${tmpl}_${lang}.html"
    [ -f "$f" ] || echo "❌ MISSING: $f"
  done
done
```

**Fix**: Copy the English template to the missing language, translate the visible text (keep `{{.Code}}`, `{{.AppName}}` template variables untouched).

### 4. mailer.go subject translations

Look at both `SendVerificationCodeLang` and `SendPasswordResetCodeLang` methods in `server/internal/auth/mailer.go`. Each has a `subjects` map that must contain entries for all 8 languages.

**Check**: `grep -E '"[a-z]{2}"' server/internal/auth/mailer.go | sort -u` should show 8 language codes.

### 5. i18n.go constants

If adding a new language constant, verify these are in sync:
- `server/pkg/i18n/i18n.go` — Lang constant + ParseLang mapping + DetectLanguage
- `server/internal/api/language.go` — langToFrontendCode() mapping
- `server/web/src/stores/ui-store.ts` — Language union type + resolveAutoLang()

### 6. README files

Root README is English. All other languages live in `docs/README.{lang}.md`.

**Check**: Verify all 7 translated READMEs exist:
```bash
for lang in zh ja fr de es ko ru; do
  [ -f "docs/README.${lang}.md" ] || echo "❌ MISSING: docs/README.${lang}.md"
done
```

**Check**: Each README's top language switcher must link to all 8 versions. Run `grep -c 'README' README.md docs/*.md` — each should show 8 links.

**Fix when adding content**: When the English README gains a new section, reproduce it in ALL 7 translated READMEs. Do not leave any behind.

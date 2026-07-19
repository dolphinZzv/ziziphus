package api

import (
	_ "embed"
	"fmt"
	"net/http"
	"strings"

	"ziziphus/pkg/i18n"
)

//go:embed privacy-zh.txt
var privacyZh string

//go:embed privacy-en.txt
var privacyEn string

//go:embed terms-zh.txt
var termsZh string

//go:embed terms-en.txt
var termsEn string

func legalHTML(title, content string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-Hans">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1.0">
<title>%s</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: #f5f5f5; color: #1a1a1a; line-height: 1.7; padding: 20px;
  }
  .container { max-width: 720px; margin: 0 auto; background: #fff; padding: 40px; border-radius: 12px; }
  h1 { font-size: 24px; margin-bottom: 8px; }
  h2 { font-size: 16px; margin-top: 24px; margin-bottom: 8px; }
  p, li { font-size: 14px; color: #444; }
  ul { padding-left: 20px; }
  li { margin-bottom: 4px; }
  .meta { color: #999; font-size: 13px; margin-bottom: 24px; }
  .back { display: inline-block; margin-bottom: 16px; color: #666; text-decoration: none; font-size: 14px; }
  .back:hover { color: #000; }
  @media (prefers-color-scheme: dark) {
    body { background: #111; color: #eee; }
    .container { background: #1a1a1a; }
    p, li { color: #bbb; }
    .back { color: #888; }
    .back:hover { color: #fff; }
    .meta { color: #666; }
  }
</style>
</head>
<body>
<div class="container">
<a href="javascript:history.back()" class="back">&larr; Back</a>
%s
</div>
</body>
</html>`, title, content)
}

func textToHTML(text string) string {
	lines := strings.Split(text, "\n")
	var html strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			html.WriteString("<br>")
		} else if strings.HasPrefix(trimmed, "- ") {
			html.WriteString(fmt.Sprintf("<li>%s</li>\n", strings.TrimPrefix(trimmed, "- ")))
		} else if strings.HasPrefix(trimmed, "# ") {
			html.WriteString(fmt.Sprintf("<h1>%s</h1>\n", strings.TrimPrefix(trimmed, "# ")))
		} else if strings.HasPrefix(trimmed, "## ") {
			html.WriteString(fmt.Sprintf("<h2>%s</h2>\n", strings.TrimPrefix(trimmed, "## ")))
		} else if strings.HasPrefix(trimmed, "1. ") ||
			strings.HasPrefix(trimmed, "2. ") ||
			strings.HasPrefix(trimmed, "3. ") ||
			strings.HasPrefix(trimmed, "4. ") ||
			strings.HasPrefix(trimmed, "5. ") ||
			strings.HasPrefix(trimmed, "6. ") ||
			strings.HasPrefix(trimmed, "7. ") {
			parts := strings.SplitN(trimmed, ". ", 2)
			if len(parts) == 2 {
				html.WriteString(fmt.Sprintf("<h2>%s. %s</h2>\n", parts[0], parts[1]))
			} else {
				html.WriteString(fmt.Sprintf("<p>%s</p>\n", line))
			}
		} else {
			html.WriteString(fmt.Sprintf("<p>%s</p>\n", line))
		}
	}
	return html.String()
}

func LegalPage(title, contentZh, contentEn string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lang := string(i18n.LangFromCtx(r.Context()))
		content := contentEn
		if strings.HasPrefix(lang, "zh") {
			content = contentZh
		}
		body := textToHTML(content)
		html := legalHTML(title, body)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	}
}

func PrivacyPage() http.HandlerFunc {
	return LegalPage("Privacy Policy 隐私政策", privacyZh, privacyEn)
}

func TermsPage() http.HandlerFunc {
	return LegalPage("Terms of Service 服务条款", termsZh, termsEn)
}

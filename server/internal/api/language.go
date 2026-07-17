package api

import (
	"net/http"

	"ziziphus/pkg/i18n"
)

type detectLangResp struct {
	Language string `json:"language"`
}

// DetectLanguage returns the best-matching language for the current request
// based on the Accept-Language header.
//
//	@summary		Detect language
//	@tags			system
//	@produce		json
//	@success		200	{object}	APIResponse{data=detectLangResp}
//	@router			/i18n/detect [get]
func (h *Handlers) DetectLanguage(w http.ResponseWriter, r *http.Request) {
	lang := i18n.LangFromCtx(r.Context())
	code := string(lang)
	// Map server-side zh-Hans to frontend zh
	if code == "zh-Hans" {
		code = "zh"
	}
	JSON(w, detectLangResp{Language: code})
}

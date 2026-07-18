package api

import (
	"net/http"

	"ziziphus/pkg/i18n"
)

type detectLangResp struct {
	Language string `json:"language"`
}

// DetectLanguage returns the best-matching language for the current request.
// Maps server-side canonical codes to frontend short codes.
//
//	@summary		Detect language
//	@tags			system
//	@produce		json
//	@success		200	{object}	APIResponse{data=detectLangResp}
//	@router			/i18n/detect [get]
func (h *Handlers) DetectLanguage(w http.ResponseWriter, r *http.Request) {
	lang := i18n.LangFromCtx(r.Context())
	code := langToFrontendCode(lang)
	JSON(w, detectLangResp{Language: code})
}

// langToFrontendCode maps server-side Lang codes to frontend short codes.
func langToFrontendCode(lang i18n.Lang) string {
	switch lang {
	case i18n.LangZH:
		return "zh"
	case i18n.LangEN:
		return "en"
	case i18n.LangJA:
		return "ja"
	case i18n.LangFR:
		return "fr"
	case i18n.LangDE:
		return "de"
	case i18n.LangES:
		return "es"
	case i18n.LangKO:
		return "ko"
	case i18n.LangRU:
		return "ru"
	default:
		return "zh"
	}
}

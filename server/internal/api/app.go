package api

import (
	"net/http"

	"ziziphus/config"
)

type appInfoResp struct {
	Name     string          `json:"name"`
	Headline string          `json:"headline"`
	Env      string          `json:"env"`
	OAuth    oauthStatusResp `json:"oauth"`
}

type oauthStatusResp struct {
	GitHub bool `json:"github"`
	Google bool `json:"google"`
}

// AppInfo returns runtime application information.
//
//	@summary		Get app info
//	@tags			system
//	@produce		json
//	@success		200	{object}	APIResponse{data=appInfoResp}
//	@router			/app/info [get]
func AppInfo(cfgMgr *config.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := cfgMgr.Get()
		JSON(w, appInfoResp{
			Name:     cfg.App.Name,
			Headline: cfg.App.Headline,
			Env:      cfg.App.Env,
			OAuth: oauthStatusResp{
				GitHub: cfg.OAuth.GitHub.Enabled,
				Google: cfg.OAuth.Google.Enabled,
			},
		})
	}
}

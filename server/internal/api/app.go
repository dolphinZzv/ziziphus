package api

import (
	"net/http"

	"ziziphus/config"
)

type appInfoResp struct {
	Name     string `json:"name"`
	Headline string `json:"headline"`
	Env      string `json:"env"`
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
		app := cfgMgr.Get().App
		JSON(w, appInfoResp{
			Name:     app.Name,
			Headline: app.Headline,
			Env:      app.Env,
		})
	}
}

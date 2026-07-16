package api

import (
	"net/http"

	"ziziphus/config"
)

type announcementResp struct {
	Enabled bool   `json:"enabled"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	URL     string `json:"url"`
}

// Announcement returns the global application announcement.
//
//	@summary		Get global announcement
//	@tags			system
//	@produce		json
//	@success		200	{object}	APIResponse{data=announcementResp}
//	@router			/announcement [get]
func Announcement(cfg config.AnnouncementConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		JSON(w, announcementResp{
			Enabled: cfg.Enabled,
			Title:   cfg.Title,
			Body:    cfg.Body,
			URL:     cfg.URL,
		})
	}
}

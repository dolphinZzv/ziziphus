package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"ziziphus/pkg/version"
)

var serverStartupTime = time.Now()

type healthResp struct {
	Status      string            `json:"status"`
	Components  map[string]string `json:"components"`
	StartupTime string            `json:"startup_time"`
	GitCommit   string            `json:"git_commit"`
}

// Health godoc
//
//	@summary		Health check
//	@tags			system
//	@produce		json
//	@success		200	{object}	APIResponse{data=healthResp}
//	@failure		503	{object}	APIResponse
//	@router			/health [get]
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	status := "ok"
	httpStatus := http.StatusOK
	components := map[string]string{}

	if err := h.DB.Ping(ctx); err != nil {
		components["database"] = err.Error()
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	} else {
		components["database"] = "ok"
	}

	if err := h.RDB.Ping(ctx).Err(); err != nil {
		components["redis"] = err.Error()
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	} else {
		components["redis"] = "ok"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)

	resp := APIResponse{Code: 0, Msg: status, Data: healthResp{
		Status:      status,
		Components:  components,
		StartupTime: serverStartupTime.Format(time.RFC3339),
		GitCommit:   version.GitCommit,
	}}
	json.NewEncoder(w).Encode(resp)
}

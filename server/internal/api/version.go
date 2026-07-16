package api

import (
	"net/http"

	"ziziphus/pkg/version"
)

type versionResp struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
}

// GetVersion returns the current server version and git commit hash.
// @Summary      Get server version
// @Description  Returns the current server version and git commit hash. Public endpoint, no authentication required.
// @Tags         system
// @Success      200 {object} APIResponse
// @Router       /version [get]
func (h *Handlers) GetVersion(w http.ResponseWriter, r *http.Request) {
	JSON(w, versionResp{
		Version:   version.ServerVersion,
		GitCommit: version.GitCommit,
	})
}

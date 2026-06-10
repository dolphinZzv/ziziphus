package api

import (
	"net/http"

	"siciv.space/agent/panda_ai/pkg/version"
)

type versionResp struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
}

func (h *Handlers) GetVersion(w http.ResponseWriter, r *http.Request) {
	JSON(w, versionResp{
		Version:   version.ServerVersion,
		GitCommit: version.GitCommit,
	})
}

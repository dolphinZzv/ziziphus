package api

import (
	"net/http"

	"github.com/dolphinz/im-server/pkg/version"
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

package webembed

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed dist
var embedded embed.FS

// Handler serves the embedded SPA with client-side routing fallback.
func Handler() http.Handler {
	sub, err := fs.Sub(embedded, "dist")
	if err != nil {
		panic("web dist not embedded: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")

		f, err := sub.Open(path)
		if err == nil {
			f.Close()
			if path == "index.html" || path == "" {
				w.Header().Set("Cache-Control", "no-cache")
			}
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback → index.html
		r.URL.Path = "/"
		w.Header().Set("Cache-Control", "no-cache")
		fileServer.ServeHTTP(w, r)
	})
}

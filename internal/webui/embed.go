// Package webui serves the React Admin frontend using go:embed.
// The compiled frontend assets in dist/ are embedded into the Go binary at build time.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist/*
var distFS embed.FS

func RegisterRoutes(mux *http.ServeMux) {
	sub, _ := fs.Sub(distFS, "dist")
	mux.Handle("/", &spaHandler{fs: http.FS(sub)})
}

type spaHandler struct {
	fs http.FileSystem
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f, err := h.fs.Open(r.URL.Path)
	if err == nil {
		f.Close()
		http.FileServer(h.fs).ServeHTTP(w, r)
		return
	}
	r.URL.Path = "/"
	http.FileServer(h.fs).ServeHTTP(w, r)
}

package dashboard

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/*
var staticFiles embed.FS

// StaticHandler returns an http.Handler serving the embedded dashboard UI.
// All unmatched paths serve index.html for client-side routing.
func StaticHandler() http.Handler {
	sub, _ := fs.Sub(staticFiles, "static")
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try serving the file directly first.
		f, err := sub.Open(r.URL.Path[1:]) // strip leading /
		if err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		// Fall back to index.html for SPA routing.
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}

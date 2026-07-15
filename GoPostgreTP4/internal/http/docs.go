package http

import (
	_ "embed"
	"net/http"
	"os"
)

//go:embed docs.html
var docsHTML []byte

// serveDocsHTML serves the Swagger UI page, which itself loads
// /docs/openapi.yaml.
func serveDocsHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(docsHTML)
}

// serveOpenAPISpec serves the repo's openapi.yaml, read from disk so it
// stays in sync with the file without needing a duplicate embedded copy.
// Requires the process to be run from the repo root (as documented in the
// README), same assumption cmd/api already makes.
func serveOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("openapi.yaml")
	if err != nil {
		http.Error(w, "openapi spec not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	_, _ = w.Write(data)
}

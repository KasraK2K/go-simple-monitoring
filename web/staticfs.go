package webstatic

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed assets/* js/*
var assets embed.FS

// GetAssetsHandler returns an HTTP handler for CSS and other static assets.
func GetAssetsHandler() http.Handler {
	assetsFS, err := fs.Sub(assets, "assets")
	if err != nil {
		return http.NotFoundHandler()
	}
	return http.FileServer(http.FS(assetsFS))
}

// GetJSHandler returns an HTTP handler for JavaScript files.
func GetJSHandler() http.Handler {
	jsFS, err := fs.Sub(assets, "js")
	if err != nil {
		return http.NotFoundHandler()
	}
	return http.FileServer(http.FS(jsFS))
}

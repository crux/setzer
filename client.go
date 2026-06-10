package main

import (
	_ "embed"
	"net/http"
)

//go:embed client.js
var clientJS []byte

// handleClient serves the version-locked Setzer client script. Its presence at
// /__setzer/client.js tells a page it's served by Setzer (editing available); on
// GitHub Pages this path 404s, so editors gate on window.Setzer being defined.
func (s *server) handleClient(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(clientJS)
}

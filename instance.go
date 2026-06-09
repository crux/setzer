package main

import (
	"io"
	"net/http"
	"strings"
	"time"
)

// handlePing answers a liveness probe used to detect an already-running Setzer
// (see the listen-error path in main).
func (s *server) handlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = io.WriteString(w, "setzer\n")
}

// setzerResponding reports whether a Setzer instance is already serving at addr.
// It confirms the responder is actually Setzer (not some other program that
// happens to hold the port) via the /__ping endpoint.
func setzerResponding(addr string) bool {
	c := http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := c.Get("http://" + addr + "/__ping")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 16))
	return strings.HasPrefix(string(b), "setzer")
}

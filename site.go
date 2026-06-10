package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

const preparingPage = `<!DOCTYPE html><meta charset="utf-8"><title>Setzer</title>
<body style="font-family:-apple-system,BlinkMacSystemFont,Segoe UI,sans-serif;background:#f7f3ec;color:#23201c;display:grid;place-items:center;height:100vh;margin:0">
<div style="text-align:center"><h1>Preparing…</h1><p>Setzer is cloning the site. Refresh in a moment.</p>
<p><a href="/admin" style="color:#7a2d28">Settings</a></p></div></body>`

// handleSite serves the working clone (the site plus its in-site editor) at the root.
func (s *server) handleSite(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	cfg := s.cfg
	ws := s.ws
	s.mu.RUnlock()

	if !cfg.Configured() {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	if ws == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, preparingPage)
		return
	}
	// Never expose Git metadata through the static server.
	if strings.Contains(r.URL.Path, "/.git") {
		http.NotFound(w, r)
		return
	}
	root := filepath.Join(ws.dir, siteSubdir(cfg.SiteDir))
	http.FileServer(http.Dir(root)).ServeHTTP(w, r)
}

// siteSubdir cleans the configured serve root, anchored so it cannot escape the clone.
func siteSubdir(dir string) string {
	if dir == "" {
		return "."
	}
	clean := strings.TrimPrefix(filepath.Clean("/"+filepath.FromSlash(dir)), string(filepath.Separator))
	if clean == "" {
		return "."
	}
	return clean
}

// handleSave accepts the in-site editor's content document and commits + pushes it.
func (s *server) handleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// CSRF defenses (on top of the loopback bind): same-origin + a JSON body.
	// A cross-site HTML form cannot set application/json, and a cross-origin
	// fetch with it triggers a preflight we never answer.
	if !sameOrigin(r) {
		http.Error(w, "cross-origin request rejected", http.StatusForbidden)
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "expected application/json", http.StatusUnsupportedMediaType)
		return
	}

	s.mu.RLock()
	cfg := s.cfg
	ws := s.ws
	s.mu.RUnlock()
	if !cfg.Configured() || ws == nil {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 5<<20)) // 5 MiB cap
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}
	// Pretty-print so committed content diffs line-by-line (issue #13).
	pretty, err := prettyJSON(body)
	if err != nil {
		http.Error(w, "body is not valid JSON", http.StatusBadRequest)
		return
	}

	auth, err := authFor(cfg)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "auth: " + err.Error()})
		return
	}
	commit, err := ws.save(cfg.ContentPath, pretty, auth)
	if err != nil {
		var pc *pushConflict
		if errors.As(err, &pc) {
			writeJSON(w, http.StatusConflict, map[string]any{
				"error":  "The site changed elsewhere, so this edit couldn't be published directly. It was saved to the branch \"" + pc.branch + "\" — open it on GitHub to merge. The editor now shows the current published content.",
				"branch": pc.branch,
				"url":    compareURL(cfg.RepoURL, cfg.Branch, pc.branch),
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "commit": commit})
}

// handleQuit stops the server — used by the admin "Quit Setzer" button.
func (s *server) handleQuit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !sameOrigin(r) {
		http.Error(w, "cross-origin request rejected", http.StatusForbidden)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	s.signalStop()
}

// signalStop closes the stop channel once, unblocking main to shut down. The
// graceful Shutdown in main lets this request's response flush first.
func (s *server) signalStop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
}

// sameOrigin rejects requests whose Origin host differs from the server's.
// A missing Origin (non-browser clients) is allowed; browsers always send it on POST.
func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return u.Host == r.Host
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// prettyJSON reformats raw JSON with 2-space indentation and a trailing newline,
// preserving the original key order, so committed content diffs line-by-line.
// It also validates: malformed JSON returns an error.
func prettyJSON(raw []byte) ([]byte, error) {
	var b bytes.Buffer
	if err := json.Indent(&b, raw, "", "  "); err != nil {
		return nil, err
	}
	b.WriteByte('\n')
	return b.Bytes(), nil
}

// compareURL builds a GitHub "open a PR" link comparing branch against base.
func compareURL(repoURL, base, branch string) string {
	web := webBase(repoURL)
	if web == "" {
		return ""
	}
	return web + "/compare/" + base + "..." + branch + "?expand=1"
}

// webBase turns a clone URL into its https web base, e.g.
// https://github.com/owner/repo(.git) or git@github.com:owner/repo(.git)
// -> https://github.com/owner/repo
func webBase(repoURL string) string {
	s := strings.TrimSuffix(strings.TrimSpace(repoURL), ".git")
	switch {
	case strings.HasPrefix(s, "git@"):
		return "https://" + strings.Replace(strings.TrimPrefix(s, "git@"), ":", "/", 1)
	case strings.HasPrefix(s, "https://"), strings.HasPrefix(s, "http://"):
		return s
	default:
		return ""
	}
}

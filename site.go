package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
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

// handleSave accepts a multipart file set from the in-site editor and commits +
// pushes it as one commit. Setzer is content-agnostic: it writes the supplied
// bytes to the supplied (sandboxed) paths without inspecting them.
func (s *server) handleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// CSRF guard: strict same-origin. A multipart body can be sent cross-site
	// without a content-type preflight, so a present, matching Origin is the
	// mandatory defense on top of the loopback bind.
	if !sameOrigin(r) {
		http.Error(w, "cross-origin request rejected", http.StatusForbidden)
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

	files, message, err := parseFileSet(r, cfg.SiteDir)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if len(files) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "no files in request"})
		return
	}

	auth, err := authFor(cfg)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "auth: " + err.Error()})
		return
	}
	commit, err := ws.saveFiles(files, message, auth)
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

// resolveUnderSite maps a web-root-relative path to a repo-relative path under
// the configured site dir, neutralising traversal and refusing any .git segment.
func resolveUnderSite(siteDir, webPath string) (string, error) {
	rel := strings.TrimPrefix(path.Clean("/"+webPath), "/")
	if rel == "" || rel == "." {
		return "", fmt.Errorf("invalid path %q", webPath)
	}
	base := strings.TrimPrefix(path.Clean("/"+siteDir), "/")
	full := path.Join(base, rel)
	for _, seg := range strings.Split(full, "/") {
		if seg == ".git" {
			return "", fmt.Errorf("refused path %q", webPath)
		}
	}
	return full, nil
}

// parseFileSet reads a multipart request into a file set. Each file part's field
// name is a web-root-relative path, resolved under siteDir; an optional
// __message field carries the commit message.
func parseFileSet(r *http.Request, siteDir string) ([]fileWrite, string, error) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return nil, "", fmt.Errorf("expected multipart/form-data: %w", err)
	}
	var files []fileWrite
	for name, headers := range r.MultipartForm.File {
		repoPath, err := resolveUnderSite(siteDir, name)
		if err != nil {
			return nil, "", err
		}
		for _, fh := range headers {
			f, err := fh.Open()
			if err != nil {
				return nil, "", fmt.Errorf("open %s: %w", name, err)
			}
			content, err := io.ReadAll(f)
			f.Close()
			if err != nil {
				return nil, "", fmt.Errorf("read %s: %w", name, err)
			}
			files = append(files, fileWrite{path: repoPath, content: content})
		}
	}
	message := ""
	if v := r.MultipartForm.Value["__message"]; len(v) > 0 {
		message = v[0]
	}
	return files, message, nil
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

// sameOrigin requires a present Origin header whose host matches the server's.
// This is Setzer's CSRF guard: browsers set Origin unforgeably on cross-origin
// requests, so a missing or foreign Origin is rejected (fail closed).
func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
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

package main

import (
	"html/template"
	"log"
	"net/http"
	"strings"
)

type adminData struct {
	Configured bool
	Cfg        *Config
}

// handleRoot renders the admin UI: setup form when unconfigured, settings when configured.
func (s *server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s.mu.RLock()
	cfg := s.cfg
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := adminTmpl.Execute(w, adminData{Configured: cfg.Configured(), Cfg: cfg}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleConfig accepts the admin form: persists non-secret config to disk and
// the token to the OS keychain.
func (s *server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	cfg := &Config{
		RepoURL:     strings.TrimSpace(r.FormValue("repo_url")),
		Branch:      strings.TrimSpace(r.FormValue("branch")),
		ContentPath: strings.TrimSpace(r.FormValue("content_path")),
	}
	if cfg.RepoURL == "" {
		http.Error(w, "repository URL is required", http.StatusBadRequest)
		return
	}
	if cfg.Branch == "" {
		cfg.Branch = "main"
	}
	if cfg.ContentPath == "" {
		cfg.ContentPath = "content.json"
	}

	if err := cfg.Save(); err != nil {
		http.Error(w, "save config: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// Only overwrite the stored token when a new one is supplied; a blank field
	// keeps the existing keychain entry.
	if token := strings.TrimSpace(r.FormValue("token")); token != "" {
		if err := SetToken(cfg.RepoURL, token); err != nil {
			http.Error(w, "store token: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	s.mu.Lock()
	s.cfg = cfg
	s.mu.Unlock()

	// Configuring (or re-pointing) the repo kicks off a clone/refresh.
	go func() {
		if err := s.syncWorkspace(); err != nil {
			log.Printf("workspace sync failed: %v", err)
		} else {
			log.Printf("workspace ready for %s", cfg.RepoURL)
		}
	}()

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

var adminTmpl = template.Must(template.New("admin").Parse(`<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><title>Setzer &middot; Setup</title>
<style>
 body{font:16px/1.6 -apple-system,BlinkMacSystemFont,"Segoe UI",Helvetica,sans-serif;
   background:#f7f3ec;color:#23201c;margin:0;display:grid;place-items:center;min-height:100vh}
 .card{background:#fffdf8;border:1px solid #d8d1c4;border-radius:10px;padding:30px 34px;width:min(480px,92vw)}
 h1{margin:.1em 0 .15em} .s{color:#5a554d;margin-top:0}
 .badge{display:inline-block;font-size:.72rem;font-weight:700;letter-spacing:.08em;text-transform:uppercase;
   background:#7a2d28;color:#fff;padding:3px 10px;border-radius:999px}
 .badge.ok{background:#3a6b4a}
 label{display:block;font-size:.8rem;font-weight:600;color:#5a554d;margin:16px 0 0}
 input{display:block;width:100%;margin-top:5px;padding:10px 12px;font-size:1rem;
   border:1px solid #d8d1c4;border-radius:6px;background:#fff;color:#23201c}
 input:focus{outline:2px solid #7a2d28;outline-offset:-1px;border-color:transparent}
 button{margin-top:22px;padding:11px 22px;font-size:.9rem;font-weight:600;border:0;border-radius:999px;
   background:#7a2d28;color:#fff;cursor:pointer}
 button:hover{background:#5e211d}
 .hint{font-size:.78rem;color:#5a554d;margin-top:16px}
</style></head>
<body><div class="card">
 {{if .Configured}}<span class="badge ok">configured</span>{{else}}<span class="badge">not configured</span>{{end}}
 <h1>Setzer</h1>
 <p class="s">{{if .Configured}}Managing <b>{{.Cfg.RepoURL}}</b>. Update settings below.{{else}}Point Setzer at the site's repository and paste a GitHub access token.{{end}}</p>
 <form method="post" action="/config">
  <label>Repository URL
   <input name="repo_url" value="{{.Cfg.RepoURL}}" placeholder="https://github.com/owner/repo.git" required></label>
  <label>Branch
   <input name="branch" value="{{if .Cfg.Branch}}{{.Cfg.Branch}}{{else}}main{{end}}"></label>
  <label>Content path (within the repo)
   <input name="content_path" value="{{if .Cfg.ContentPath}}{{.Cfg.ContentPath}}{{else}}content.json{{end}}"></label>
  <label>GitHub access token (PAT)
   <input name="token" type="password" autocomplete="off"
     placeholder="{{if .Configured}}stored — leave blank to keep{{else}}fine-grained PAT, contents:write{{end}}"></label>
  <button type="submit">Save</button>
 </form>
 <p class="hint">The token is stored in your OS keychain — never on disk or in any repo.</p>
</div></body></html>
`))

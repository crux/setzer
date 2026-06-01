// Setzer — a local, single-binary compositor for static sites.
//
// It serves a static site (with the site's own in-site editor) on localhost and,
// on save, commits the content change to the site's Git repository — which a
// static host such as GitHub Pages then publishes. See docs/0001-architecture.html.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	// Loopback only by default: the admin UI and save endpoint must never be
	// reachable off-host. Binding to 127.0.0.1 enforces that at the socket.
	addr := flag.String("addr", "127.0.0.1:8765", "loopback address to listen on")
	flag.Parse()

	srv := &server{}

	mux := http.NewServeMux()
	mux.HandleFunc("/", srv.handleRoot)

	log.Printf("setzer listening on http://%s", *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		fmt.Fprintln(os.Stderr, "setzer:", err)
		os.Exit(1)
	}
}

// server holds Setzer's runtime state. For now it is empty; configuration,
// the working clone and the keychain-backed token arrive in later increments.
type server struct{}

func (s *server) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, adminPlaceholder)
}

const adminPlaceholder = `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><title>Setzer</title>
<style>
 body{font:16px/1.6 -apple-system,BlinkMacSystemFont,"Segoe UI",Helvetica,sans-serif;
   background:#f7f3ec;color:#23201c;margin:0;display:grid;place-items:center;min-height:100vh}
 .card{background:#fffdf8;border:1px solid #d8d1c4;border-radius:10px;padding:32px 36px;max-width:460px}
 h1{margin:0 0 .2em} .s{color:#5a554d}
 .badge{display:inline-block;font-size:.72rem;font-weight:700;letter-spacing:.08em;text-transform:uppercase;
   background:#7a2d28;color:#fff;padding:3px 10px;border-radius:999px}
</style></head>
<body><div class="card">
 <span class="badge">not configured</span>
 <h1>Setzer</h1>
 <p class="s">The compositor is running. Configuration (repository &amp; access token)
 is not wired up yet — that's the next build increment.</p>
</div></body></html>
`

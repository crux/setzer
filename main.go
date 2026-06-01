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
	"sync"
)

func main() {
	// Loopback only by default: the admin UI and save endpoint must never be
	// reachable off-host. Binding to 127.0.0.1 enforces that at the socket.
	addr := flag.String("addr", "127.0.0.1:8765", "loopback address to listen on")
	flag.Parse()

	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "setzer: load config:", err)
		os.Exit(1)
	}

	srv := &server{cfg: cfg}

	mux := http.NewServeMux()
	mux.HandleFunc("/", srv.handleRoot)
	mux.HandleFunc("/config", srv.handleConfig)

	log.Printf("setzer listening on http://%s", *addr)
	if cfg.Configured() {
		log.Printf("configured for %s (branch %s)", cfg.RepoURL, cfg.Branch)
	} else {
		log.Printf("not configured — open the address above to set up")
	}
	if err := http.ListenAndServe(*addr, mux); err != nil {
		fmt.Fprintln(os.Stderr, "setzer:", err)
		os.Exit(1)
	}
}

// server holds Setzer's runtime state. The working clone and the /__save
// handler arrive in later increments. cfg is guarded by mu because the admin
// UI can replace it while requests are in flight.
type server struct {
	mu  sync.RWMutex
	cfg *Config
}

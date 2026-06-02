// Setzer — a local, single-binary compositor for static sites.
//
// It serves a static site (with the site's own in-site editor) on localhost and,
// on save, commits the content change to the site's Git repository — which a
// static host such as GitHub Pages then publishes. See docs/0001-architecture.html.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

func main() {
	// Loopback only by default: the admin UI and save endpoint must never be
	// reachable off-host. Binding to 127.0.0.1 enforces that at the socket.
	addr := flag.String("addr", "127.0.0.1:8765", "loopback address to listen on")
	open := flag.Bool("open", false, "open the admin UI in a browser on start")
	flag.Parse()

	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "setzer: load config:", err)
		os.Exit(1)
	}

	srv := &server{cfg: cfg, stop: make(chan struct{})}

	mux := http.NewServeMux()
	mux.HandleFunc("/admin", srv.handleAdmin)   // config UI
	mux.HandleFunc("/config", srv.handleConfig) // POST config
	mux.HandleFunc("/__save", srv.handleSave)   // POST content -> commit + push
	mux.HandleFunc("/__quit", srv.handleQuit)   // POST -> stop the server
	mux.HandleFunc("/", srv.handleSite)         // serve the working clone (or setup)

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "setzer: listen:", err)
		os.Exit(1)
	}
	url := "http://" + ln.Addr().String() + "/"
	log.Printf("setzer listening on %s", url)
	if cfg.Configured() {
		log.Printf("configured for %s (branch %s)", cfg.RepoURL, cfg.Branch)
		// Clone/refresh the working copy in the background so startup isn't
		// blocked on the network.
		go func() {
			if err := srv.syncWorkspace(); err != nil {
				log.Printf("workspace sync failed: %v", err)
			} else {
				log.Printf("workspace ready")
			}
		}()
	} else {
		log.Printf("not configured — open the address above to set up")
	}

	// The .app bundle sets SETZER_OPEN so a double-click pops the browser.
	if *open || os.Getenv("SETZER_OPEN") != "" {
		go openBrowser(url)
	}

	httpSrv := &http.Server{Handler: mux}
	go func() {
		if err := httpSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
			fmt.Fprintln(os.Stderr, "setzer:", err)
			os.Exit(1)
		}
	}()

	<-srv.stop // closed by /__quit
	log.Printf("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
}

// server holds Setzer's runtime state. cfg and ws are guarded by mu because the
// admin UI can replace the config (and trigger a re-clone) while requests are in
// flight. stop is closed once to trigger a graceful shutdown.
type server struct {
	mu   sync.RWMutex
	cfg  *Config
	ws   *workspace
	stop chan struct{}
}

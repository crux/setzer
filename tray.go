//go:build darwin || windows

package main

import (
	_ "embed"
	"runtime"

	"fyne.io/systray"
	"github.com/gen2brain/beeep"
)

//go:embed icon.png
var iconPNG []byte

//go:embed icon.ico
var iconICO []byte

// runTray shows a menu-bar/tray presence (Open · status · Quit, plus a conflict
// item) and surfaces publish/conflict events as OS notifications. It blocks,
// owning the main thread as macOS requires, until the user quits. A web /__quit
// tears the tray down too; both quit paths funnel through srv.stop.
func runTray(srv *server, url string) {
	onReady := func() {
		if runtime.GOOS == "windows" {
			systray.SetIcon(iconICO)
		} else {
			systray.SetTemplateIcon(iconPNG, iconPNG)
		}
		systray.SetTooltip("Setzer")

		mOpen := systray.AddMenuItem("Open Setzer", "Open the editor in your browser")
		systray.AddSeparator()
		mStatus := systray.AddMenuItem(trayStatus(srv), "")
		mStatus.Disable()
		mConflict := systray.AddMenuItem("⚠ Merge conflict on GitHub", "An edit was offloaded to a branch — merge it")
		mConflict.Hide()
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit Setzer", "Stop Setzer")

		var conflictURL string

		// A web /__quit closes srv.stop — tear the tray down to match.
		go func() {
			<-srv.stop
			systray.Quit()
		}()

		// Publish/conflict events -> OS notification + status line.
		go func() {
			for ev := range srv.events {
				_ = beeep.Notify(ev.title, ev.body, "")
				switch ev.kind {
				case "published":
					mStatus.SetTitle(trayStatus(srv))
					mConflict.Hide()
				case "conflict":
					conflictURL = ev.url
					mStatus.SetTitle("⚠ unmerged: " + ev.branch)
					mConflict.Show()
				}
			}
		}()

		go func() {
			for {
				select {
				case <-mOpen.ClickedCh:
					openBrowser(url)
				case <-mConflict.ClickedCh:
					if conflictURL != "" {
						openBrowser(conflictURL)
					}
				case <-mQuit.ClickedCh:
					srv.signalStop() // -> stop watcher above quits the tray
					return
				}
			}
		}()
	}
	systray.Run(onReady, func() { srv.signalStop() })
}

func trayStatus(srv *server) string {
	if srv.dev != "" {
		return "● dev: " + srv.dev
	}
	srv.mu.RLock()
	cfg := srv.cfg
	srv.mu.RUnlock()
	if cfg != nil && cfg.Configured() {
		return "● " + cfg.RepoURL
	}
	return "● not configured"
}

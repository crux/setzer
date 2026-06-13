//go:build !darwin && !windows

package main

// runTray has no menu-bar/tray on this platform; run headless until /__quit.
func runTray(srv *server, url string) {
	<-srv.stop
}

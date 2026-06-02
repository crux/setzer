package main

import (
	"os/exec"
	"runtime"
)

// openBrowser opens url in the default browser. Best-effort: errors are ignored
// (the URL is also logged, so the user can open it manually).
func openBrowser(url string) {
	var name string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		name, args = "open", []string{url}
	case "windows":
		name, args = "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default: // linux, *bsd
		name, args = "xdg-open", []string{url}
	}
	_ = exec.Command(name, args...).Start()
}

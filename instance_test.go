package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSetzerResponding(t *testing.T) {
	// A server that answers /__ping like Setzer.
	setzer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/__ping" {
			_, _ = w.Write([]byte("setzer\n"))
			return
		}
		http.NotFound(w, r)
	}))
	defer setzer.Close()
	if !setzerResponding(addrOf(setzer)) {
		t.Fatal("expected Setzer to be detected on its own /__ping")
	}

	// Some other program holding the port.
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html>not setzer</html>"))
	}))
	defer other.Close()
	if setzerResponding(addrOf(other)) {
		t.Fatal("must not mistake a foreign server for Setzer")
	}

	// Nothing listening.
	if setzerResponding("127.0.0.1:1") {
		t.Fatal("must be false when nothing responds")
	}
}

func addrOf(s *httptest.Server) string {
	return strings.TrimPrefix(s.URL, "http://")
}

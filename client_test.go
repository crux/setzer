package main

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleClientServesLib(t *testing.T) {
	s := &server{}
	rec := httptest.NewRecorder()
	s.handleClient(rec, httptest.NewRequest("GET", "/__setzer/client.js", nil))

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "javascript") {
		t.Fatalf("Content-Type = %q, want javascript", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "window.Setzer") || !strings.Contains(body, "publish") {
		t.Fatal("served client is missing window.Setzer / publish")
	}
}

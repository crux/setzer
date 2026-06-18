package main

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleReady(t *testing.T) {
	check := func(s *server, want string) {
		t.Helper()
		rec := httptest.NewRecorder()
		s.handleReady(rec, httptest.NewRequest("GET", "/__ready", nil))
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("ready body = %q, want it to contain %q", rec.Body.String(), want)
		}
	}
	check(&server{dev: "/some/dir"}, `"ready":true`)        // dev mode serves immediately
	check(&server{cfg: &Config{}}, `"ready":false`)         // unconfigured, no clone
	check(&server{cfg: &Config{RepoURL: "x"}}, `"ready":false`) // configured but ws still nil (cloning)
}

func TestResolveUnderSite(t *testing.T) {
	cases := []struct {
		siteDir, web, want string
		ok                 bool
	}{
		{"docs", "content.json", "docs/content.json", true},
		{"docs", "img/cover.jpg", "docs/img/cover.jpg", true},
		{".", "content.json", "content.json", true},
		{"", "content.json", "content.json", true},
		{"docs", "../secret", "docs/secret", true},            // .. neutralised, stays under docs
		{"docs", "../../etc/passwd", "docs/etc/passwd", true}, // cannot escape the site dir
		{"docs", ".git/config", "", false},                   // .git refused
		{"", ".git/hooks/x", "", false},                      // .git refused at root
		{"docs", "", "", false},                              // empty path
	}
	for _, c := range cases {
		got, err := resolveUnderSite(c.siteDir, c.web)
		if c.ok {
			if err != nil || got != c.want {
				t.Errorf("resolveUnderSite(%q,%q) = %q,%v; want %q,nil", c.siteDir, c.web, got, err, c.want)
			}
		} else if err == nil {
			t.Errorf("resolveUnderSite(%q,%q) = %q,nil; want error", c.siteDir, c.web, got)
		}
	}
}

func TestParseFileSet(t *testing.T) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	tp, _ := mw.CreateFormFile("content.json", "content.json")
	_, _ = tp.Write([]byte("{\"v\":1}\n"))
	ip, _ := mw.CreateFormFile("img/x.bin", "img/x.bin")
	_, _ = ip.Write([]byte{0x00, 0x01, 0x02, 0xff}) // binary, must survive intact
	_ = mw.WriteField("__message", "msg")
	_ = mw.Close()

	req := httptest.NewRequest("POST", "/__save", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	files, msg, err := parseFileSet(req, "docs")
	if err != nil {
		t.Fatal(err)
	}
	if msg != "msg" {
		t.Fatalf("message = %q, want %q", msg, "msg")
	}
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2", len(files))
	}
	byPath := map[string][]byte{}
	for _, f := range files {
		byPath[f.path] = f.content // paths resolved under "docs"
	}
	if string(byPath["docs/content.json"]) != "{\"v\":1}\n" {
		t.Errorf("content.json = %q", byPath["docs/content.json"])
	}
	if !bytes.Equal(byPath["docs/img/x.bin"], []byte{0x00, 0x01, 0x02, 0xff}) {
		t.Errorf("binary part not preserved: %v", byPath["docs/img/x.bin"])
	}
}

func TestHandleSaveDevWritesNoGit(t *testing.T) {
	dir := t.TempDir()
	srv := &server{dev: dir}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	p, _ := mw.CreateFormFile("content.json", "content.json")
	_, _ = p.Write([]byte("{\"v\":1}\n"))
	_ = mw.Close()

	req := httptest.NewRequest("POST", "/__save", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Origin", "http://"+req.Host) // strict-origin guard must pass
	rec := httptest.NewRecorder()
	srv.handleSave(rec, req) // dispatches to handleSaveDev in dev mode

	if rec.Code != 200 {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "content.json")); string(got) != "{\"v\":1}\n" {
		t.Fatalf("written content = %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		t.Fatal("dev mode must not create a git repo")
	}
}

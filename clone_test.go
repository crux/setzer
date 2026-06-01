package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// TestCloneOrPull exercises both the clone and the open+pull paths against a
// local file-based "remote" — deterministic, no network.
func TestCloneOrPull(t *testing.T) {
	remoteDir := t.TempDir()
	remote, err := git.PlainInit(remoteDir, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(remoteDir, "content.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wt, err := remote.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Add("content.json"); err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Commit("init", &git.CommitOptions{
		Author: &object.Signature{Name: "t", Email: "t@example.com", When: time.Unix(0, 0).UTC()},
	}); err != nil {
		t.Fatal(err)
	}
	head, err := remote.Head()
	if err != nil {
		t.Fatal(err)
	}
	branch := head.Name().Short()

	work := filepath.Join(t.TempDir(), "work")

	// First call clones.
	ws, err := cloneOrPull(work, remoteDir, branch, nil)
	if err != nil {
		t.Fatalf("clone: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ws.dir, "content.json")); err != nil {
		t.Fatalf("expected content.json in clone: %v", err)
	}

	// Second call opens and pulls (already up to date).
	if _, err := cloneOrPull(work, remoteDir, branch, nil); err != nil {
		t.Fatalf("re-pull: %v", err)
	}
}

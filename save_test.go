package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// TestSavePushesToRemote drives the full write -> commit -> push loop against a
// local bare "remote", then verifies the remote actually received the change.
func TestSavePushesToRemote(t *testing.T) {
	// Seed a working repo with one commit.
	seed := t.TempDir()
	sr, err := git.PlainInit(seed, false)
	if err != nil {
		t.Fatal(err)
	}
	writeCommit(t, sr, seed, "content.json", "{\"v\":1}\n", "seed")
	branch := headBranch(t, sr)

	// Push it into a bare remote.
	bare := t.TempDir()
	if _, err := git.PlainInit(bare, true); err != nil {
		t.Fatal(err)
	}
	if _, err := sr.CreateRemote(&gitconfig.RemoteConfig{Name: "origin", URLs: []string{bare}}); err != nil {
		t.Fatal(err)
	}
	refspec := gitconfig.RefSpec("refs/heads/" + branch + ":refs/heads/" + branch)
	if err := sr.Push(&git.PushOptions{RemoteName: "origin", RefSpecs: []gitconfig.RefSpec{refspec}}); err != nil {
		t.Fatal(err)
	}

	// Setzer clones the bare remote and saves new content.
	work := filepath.Join(t.TempDir(), "work")
	ws, err := cloneOrPull(work, bare, branch, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("{\"v\":2}\n")
	if _, err := ws.saveFiles([]fileWrite{{path: "content.json", content: want}}, "", nil); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Verify the remote received it by re-cloning.
	check := filepath.Join(t.TempDir(), "check")
	if _, err := cloneOrPull(check, bare, branch, nil); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(check, "content.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("remote content = %q, want %q", got, want)
	}
}

func writeCommit(t *testing.T, r *git.Repository, dir, name, body, msg string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	wt, err := r.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Add(name); err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{Name: "t", Email: "t@example.com", When: time.Unix(0, 0).UTC()},
	}); err != nil {
		t.Fatal(err)
	}
}

func headBranch(t *testing.T, r *git.Repository) string {
	t.Helper()
	h, err := r.Head()
	if err != nil {
		t.Fatal(err)
	}
	return h.Name().Short()
}

package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

// TestSaveOffloadsOnDivergence: when the remote moved under us, save() must not
// wedge — it offloads the commit to a draft branch and returns local to origin.
func TestSaveOffloadsOnDivergence(t *testing.T) {
	// Bare remote seeded with v1.
	seed := t.TempDir()
	sr, err := git.PlainInit(seed, false)
	if err != nil {
		t.Fatal(err)
	}
	writeCommit(t, sr, seed, "content.json", "{\"v\":1}\n", "seed")
	branch := headBranch(t, sr)

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

	// Setzer's clone, based on v1.
	work := filepath.Join(t.TempDir(), "work")
	ws, err := cloneOrPull(work, bare, branch, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Another machine advances the remote to v2.
	other := filepath.Join(t.TempDir(), "other")
	or, err := git.PlainClone(other, false, &git.CloneOptions{
		URL: bare, ReferenceName: plumbing.NewBranchReferenceName(branch), SingleBranch: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	writeCommit(t, or, other, "content.json", "{\"v\":2}\n", "remote v2")
	if err := or.Push(&git.PushOptions{RemoteName: "origin", RefSpecs: []gitconfig.RefSpec{refspec}}); err != nil {
		t.Fatal(err)
	}

	// Setzer saves v3 on its now-stale base -> non-fast-forward -> offload.
	if _, serr := ws.save("content.json", []byte("{\"v\":3}\n"), nil); true {
		var pc *pushConflict
		if !errors.As(serr, &pc) {
			t.Fatalf("expected pushConflict, got: %v", serr)
		}
		if !strings.HasPrefix(pc.branch, "setzer/draft-") {
			t.Fatalf("unexpected draft branch %q", pc.branch)
		}

		// The draft branch is on the remote and holds v3.
		check := filepath.Join(t.TempDir(), "check")
		if _, err := git.PlainClone(check, false, &git.CloneOptions{
			URL: bare, ReferenceName: plumbing.NewBranchReferenceName(pc.branch), SingleBranch: true,
		}); err != nil {
			t.Fatalf("draft branch not pushed to remote: %v", err)
		}
		if got, _ := os.ReadFile(filepath.Join(check, "content.json")); string(got) != "{\"v\":3}\n" {
			t.Fatalf("draft content = %q, want v3", got)
		}
	}

	// Local working clone returned to origin (v2) — not wedged on v3.
	if local, _ := os.ReadFile(filepath.Join(work, "content.json")); string(local) != "{\"v\":2}\n" {
		t.Fatalf("local content after offload = %q, want v2 (origin)", local)
	}
}

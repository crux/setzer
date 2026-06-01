package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/zalando/go-keyring"
)

// workspace is a local working clone of the configured site repository.
type workspace struct {
	dir    string
	branch string
	repo   *git.Repository
}

// cloneDir returns a stable per-repo cache directory for the working clone.
func cloneDir(repoURL string) (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "setzer", "clones", repoSlug(repoURL)), nil
}

// repoSlug turns a repo URL into a filesystem-safe directory name.
func repoSlug(repoURL string) string {
	s := strings.TrimSuffix(repoURL, ".git")
	for _, p := range []string{"https://", "http://", "ssh://", "git@"} {
		s = strings.TrimPrefix(s, p)
	}
	return strings.NewReplacer("/", "-", ":", "-", "@", "-").Replace(s)
}

// authFor builds git auth from the keychain PAT. A missing token yields nil auth
// (which still works for public repositories).
func authFor(repoURL string) (transport.AuthMethod, error) {
	token, err := GetToken(repoURL)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	// GitHub accepts any non-empty username when authenticating with a token.
	return &githttp.BasicAuth{Username: "setzer", Password: token}, nil
}

// ensureWorkspace clones/opens the configured repo under the cache dir.
func ensureWorkspace(cfg *Config, auth transport.AuthMethod) (*workspace, error) {
	dir, err := cloneDir(cfg.RepoURL)
	if err != nil {
		return nil, err
	}
	return cloneOrPull(dir, cfg.RepoURL, cfg.Branch, auth)
}

// cloneOrPull clones repoURL into dir if absent, otherwise opens it and
// fast-forwards the branch. Separated from ensureWorkspace so it can be tested
// against a local file:// remote without touching the user's cache dir.
func cloneOrPull(dir, repoURL, branch string, auth transport.AuthMethod) (*workspace, error) {
	if branch == "" {
		branch = "main"
	}
	refName := plumbing.NewBranchReferenceName(branch)

	_, statErr := os.Stat(filepath.Join(dir, ".git"))
	switch {
	case errors.Is(statErr, os.ErrNotExist):
		repo, err := git.PlainClone(dir, false, &git.CloneOptions{
			URL:           repoURL,
			Auth:          auth,
			ReferenceName: refName,
			SingleBranch:  true,
		})
		if err != nil {
			return nil, fmt.Errorf("clone: %w", err)
		}
		return &workspace{dir: dir, branch: branch, repo: repo}, nil
	case statErr != nil:
		return nil, statErr
	}

	repo, err := git.PlainOpen(dir)
	if err != nil {
		return nil, fmt.Errorf("open clone: %w", err)
	}
	ws := &workspace{dir: dir, branch: branch, repo: repo}
	if err := ws.pull(auth); err != nil {
		return nil, err
	}
	return ws, nil
}

// pull fast-forwards the working tree to origin's branch.
func (w *workspace) pull(auth transport.AuthMethod) error {
	wt, err := w.repo.Worktree()
	if err != nil {
		return err
	}
	err = wt.Pull(&git.PullOptions{
		RemoteName:    "origin",
		ReferenceName: plumbing.NewBranchReferenceName(w.branch),
		Auth:          auth,
		SingleBranch:  true,
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("pull: %w", err)
	}
	return nil
}

// syncWorkspace ensures the working clone exists and is up to date, then stores
// it on the server. Safe to call from a goroutine.
func (s *server) syncWorkspace() error {
	s.mu.RLock()
	cfg := s.cfg
	s.mu.RUnlock()
	if !cfg.Configured() {
		return nil
	}
	auth, err := authFor(cfg.RepoURL)
	if err != nil {
		return err
	}
	ws, err := ensureWorkspace(cfg, auth)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.ws = ws
	s.mu.Unlock()
	return nil
}

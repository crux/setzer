package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
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

// authFor builds git auth from either the GitHub CLI (when enabled) or the
// keychain PAT. A missing keychain token yields nil auth (works for public repos).
func authFor(cfg *Config) (transport.AuthMethod, error) {
	var token string
	if cfg.UseGH {
		t, err := ghToken()
		if err != nil {
			return nil, fmt.Errorf("gh auth token: %w", err)
		}
		token = t
	} else {
		t, err := GetToken(cfg.RepoURL)
		if err != nil {
			if errors.Is(err, keyring.ErrNotFound) {
				return nil, nil
			}
			return nil, err
		}
		token = t
	}
	if token == "" {
		return nil, nil
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

// pushConflict means the publish couldn't fast-forward because the remote moved;
// the local commit was offloaded to a side branch for the human to merge.
type pushConflict struct{ branch string }

func (e *pushConflict) Error() string { return "publish conflict: offloaded to " + e.branch }

// fileWrite is one file to commit: a repo-relative path and its raw bytes.
type fileWrite struct {
	path    string
	content []byte
}

// saveFiles writes each file into the clone, commits them as a single commit,
// and pushes to origin. On a non-fast-forward it offloads the commit to a draft
// branch and returns *pushConflict. Setzer is content-agnostic: it writes the
// given bytes to the given (sandboxed) repo-relative paths without inspecting
// them.
func (w *workspace) saveFiles(files []fileWrite, message string, auth transport.AuthMethod) (string, error) {
	if len(files) == 0 {
		return "", fmt.Errorf("no files to save")
	}
	wt, err := w.repo.Worktree()
	if err != nil {
		return "", err
	}
	for _, f := range files {
		// Anchor to "/" then trim, so any ".." is neutralised before it can escape the clone.
		rel := strings.TrimPrefix(path.Clean("/"+f.path), "/")
		if rel == "" || rel == "." {
			return "", fmt.Errorf("invalid path %q", f.path)
		}
		full := filepath.Join(w.dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(full, f.content, 0o644); err != nil {
			return "", err
		}
		if _, err := wt.Add(rel); err != nil {
			return "", fmt.Errorf("git add %s: %w", rel, err)
		}
	}

	hash, err := wt.Commit(commitMessage(message, files), &git.CommitOptions{
		Author: &object.Signature{Name: "Setzer", Email: "setzer@localhost", When: time.Now()},
	})
	if err != nil {
		return "", fmt.Errorf("commit: %w", err)
	}

	refSpec := gitconfig.RefSpec(fmt.Sprintf("refs/heads/%s:refs/heads/%s", w.branch, w.branch))
	err = w.repo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       auth,
		RefSpecs:   []gitconfig.RefSpec{refSpec},
	})
	switch {
	case err == nil, errors.Is(err, git.NoErrAlreadyUpToDate):
		return hash.String(), nil
	case isNonFastForward(err):
		branch, oerr := w.offloadToBranch(hash, auth)
		if oerr != nil {
			return "", fmt.Errorf("publish conflict, and offloading it to a branch failed: %w", oerr)
		}
		return "", &pushConflict{branch: branch}
	default:
		return "", fmt.Errorf("push: %w", err)
	}
}

// commitMessage returns the editor-supplied message, or a default describing the set.
func commitMessage(message string, files []fileWrite) string {
	if strings.TrimSpace(message) != "" {
		return message
	}
	if len(files) == 1 {
		return "Update " + files[0].path + " via Setzer"
	}
	return fmt.Sprintf("Update %d files via Setzer", len(files))
}

// offloadToBranch preserves a commit that couldn't fast-forward: it creates and
// pushes a "setzer/draft-<ts>" branch holding the commit, then returns the local
// branch to current origin/<branch>. The reset is safe — the work is already on
// the pushed branch — so nothing is lost; the human merges the draft on GitHub.
func (w *workspace) offloadToBranch(commit plumbing.Hash, auth transport.AuthMethod) (string, error) {
	name := "setzer/draft-" + time.Now().UTC().Format("20060102-150405")
	ref := plumbing.NewBranchReferenceName(name)
	if err := w.repo.Storer.SetReference(plumbing.NewHashReference(ref, commit)); err != nil {
		return "", err
	}
	rs := gitconfig.RefSpec(fmt.Sprintf("%s:%s", ref, ref))
	if err := w.repo.Push(&git.PushOptions{RemoteName: "origin", Auth: auth, RefSpecs: []gitconfig.RefSpec{rs}}); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return "", fmt.Errorf("push draft branch: %w", err)
	}
	// Refresh origin/<branch>, then reset local <branch> to it (clean main).
	if err := w.repo.Fetch(&git.FetchOptions{RemoteName: "origin", Auth: auth}); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return "", fmt.Errorf("fetch: %w", err)
	}
	remoteRef, err := w.repo.Reference(plumbing.NewRemoteReferenceName("origin", w.branch), true)
	if err != nil {
		return "", err
	}
	wt, err := w.repo.Worktree()
	if err != nil {
		return "", err
	}
	if err := wt.Reset(&git.ResetOptions{Commit: remoteRef.Hash(), Mode: git.HardReset}); err != nil {
		return "", fmt.Errorf("reset to origin/%s: %w", w.branch, err)
	}
	return name, nil
}

// isNonFastForward detects a push rejected because the remote advanced.
func isNonFastForward(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "non-fast-forward")
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
	auth, err := authFor(cfg)
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

# Setzer

A single-binary local **compositor** for static sites.

Setzer runs on your machine, serves a static site — together with the site's own
in-site editor — on localhost, and on save commits the content change to the
site's Git repository, which a static host such as GitHub Pages then publishes.

Setzer is content-schema-agnostic and reusable across sites: each site brings its
own design and editor; Setzer only **serves** and **commits**. The name is the
German word for a print-shop *compositor* — it doesn't author or design, it
arranges prepared content and hands it to the press.

- **Storage / versioning / hosting:** the site's Git repo (e.g. GitHub Pages)
- **Runtime:** one local Go binary — nothing to deploy, no cloud secrets
- **Auth:** a GitHub token held locally in the OS keychain

Full design rationale: [`docs/0001-architecture.html`](docs/0001-architecture.html).

## Build & run

```sh
make build      # compile ./setzer
make run        # build and serve http://127.0.0.1:8765
make test       # run the tests
```

`make` on its own lists all targets.

## Configure

Open <http://127.0.0.1:8765> — when unconfigured it redirects to `/admin`. Fill in:

| Field | Meaning | Example |
|-------|---------|---------|
| Repository URL | the site repo Setzer commits to | `https://github.com/owner/site.git` |
| Branch | branch to serve and push | `main` |
| Site directory | publish root within the repo | `site` (or `.` for repo root) |
| Content path | the editable content file, repo-relative | `site/js/content.json` |
| Access token | GitHub PAT — see below | — |

On Save, Setzer clones the repo, then serves it at `/`. The in-site editor posts
its content to `/__save`, which writes the file, commits, and pushes.

## GitHub access token (PAT)

Setzer pushes commits on your behalf, so it needs a token with write access to
the **one** repository. Use a **fine-grained** personal access token:

1. GitHub → **Settings → Developer settings → Personal access tokens →
   Fine-grained tokens → Generate new token**.
2. **Resource owner:** the account or org that owns the repo.
3. **Repository access:** *Only select repositories* → choose the site repo.
4. **Permissions → Repository permissions → Contents: Read and write.**
   (That is the only permission needed; Metadata: read is included automatically.)
5. Choose an expiration, generate, and copy the token.
6. Paste it into Setzer's `/admin` form and Save.

The token is stored in your **OS keychain** (macOS Keychain, Windows Credential
Manager, or libsecret on Linux) — never written to disk or to any repository.
On macOS you may be prompted once to allow keychain access.

To rotate: generate a new token, paste it into `/admin` again, then revoke the
old one on GitHub.

## Status

Early MVP. The tool builds, runs, configures, clones, and serves the site, and
the `/__save` → commit → push loop is covered by tests. To publish from the
editor end to end, the site's in-site editor must post its content to `/__save`
(the site-side half of the contract).

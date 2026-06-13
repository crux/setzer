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
How a site talks to Setzer: [`docs/client-contract.md`](docs/client-contract.md).

## Install

### macOS — Homebrew (recommended)

```sh
brew install --cask crux/tap/setzer
```

Launches cleanly — the cask clears macOS's quarantine flag, so there's no
Gatekeeper prompt.

### macOS — direct download (DMG)

Download `Setzer-<version>.dmg` from the
[latest release](https://github.com/crux/setzer/releases/latest), open it, and
drag **Setzer.app** into **Applications**.

Setzer is unsigned (not notarized), so the **first launch is blocked** —
*"Apple could not verify that 'Setzer.app' is free of malware…"*. To allow it on
**macOS Sequoia** (the old right-click → Open trick no longer works):

1. Click **Done** on the warning (not "Move to Trash").
2. Open **System Settings → Privacy & Security**, scroll to **Security**.
3. Next to *"Setzer.app was blocked…"*, click **Open Anyway** and authenticate.
4. Click **Open Anyway** once more to confirm — it launches and is remembered.

(Homebrew skips this step; it's the easy path.)

### Windows — installer

Download `Setzer-Setup-<version>.exe` from the
[latest release](https://github.com/crux/setzer/releases/latest) and run it.
It's unsigned, so **SmartScreen** warns once — click **More info → Run anyway** —
then click through the installer (per-user, no admin). Launch **Setzer** from the
**Start Menu**; uninstall via **Settings → Apps**.

## Build & run

```sh
make build      # compile ./setzer
make run        # build and serve http://127.0.0.1:8765
make test       # run the tests
make app        # build build/Setzer.app (macOS)
```

`make` on its own lists all targets.

### Develop a site locally

For fast iteration on a site (and its editor) without commit/push round-trips:

```sh
setzer -dev path/to/site/docs
```

Setzer serves that directory **live** (edits show on a refresh) and the in-site
editor works, but saves write straight into the directory with **no git** — so
you can pick/crop/publish and `git diff` the result locally. Add `-no-tray` to
run fully headless (CI/scripting). When a change is good, `git commit && push`.

### macOS app

`make app` produces a double-clickable **`Setzer.app`**. Launching it opens the
editor in your browser and shows a **menu-bar icon** (no Dock icon) with
**Open Setzer** · status · **Quit Setzer**, and posts OS notifications when you
publish — or when a publish hits a conflict. Run headless with `-no-tray`. It's
an unsigned bundle — see [Install](#install) for the first-launch Gatekeeper steps.

## Releasing

Cut a release with one command — CI does everything else:

```sh
make release VERSION=0.2.0
```

`make release` checks you're on a clean, pushed `main`, then tags `v0.2.0` and
pushes it (the equivalent of `git tag v0.2.0 && git push origin v0.2.0`), which
triggers the workflow below.

On a `v*` tag, [`.github/workflows/release.yml`](.github/workflows/release.yml)
runs on a macOS runner and:

1. `make dist` → builds a universal (arm64 + amd64) `Setzer.app`, patches its
   version from the tag, and packages a drag-install **DMG** (`dist/`).
2. Builds the **Windows installer** (`make windows`, via `makensis`) and creates
   the GitHub release with the **DMG + installer** attached.
3. Generates the cask via [`scripts/write-cask.sh`](scripts/write-cask.sh) and
   pushes it to [`crux/homebrew-tap`](https://github.com/crux/homebrew-tap), so
   `brew install --cask crux/tap/setzer` serves the new version.

**One-time prerequisite:** the repo needs a `HOMEBREW_TAP_TOKEN` secret — a token
with write access to `crux/homebrew-tap`:

```sh
gh secret set HOMEBREW_TAP_TOKEN --repo crux/setzer
```

Build locally without releasing: `make dist` (macOS DMG) or `make windows`
(Windows installer — needs `brew install makensis`). Output in `dist/`.

The moving parts: `Makefile` (`build` / `app` / `dist` / `windows`),
`packaging/macos/Info.plist` + `packaging/windows/setzer.nsi`,
`scripts/write-cask.sh`, and the release workflow. Architecture and rationale
live in [`docs/0001-architecture.html`](docs/0001-architecture.html).

## Configure

Open <http://127.0.0.1:8765> — when unconfigured it redirects to `/admin`. Fill in:

| Field | Meaning | Example |
|-------|---------|---------|
| Repository URL | the site repo Setzer commits to | `https://github.com/owner/site.git` |
| Branch | branch to serve and push | `main` |
| Site directory | publish root within the repo | `docs` (or `.` for repo root) |
| Access token | GitHub PAT — see below | — |

On Save, Setzer clones the repo, then serves it at `/`. The in-site editor posts
its changed files to `/__save`, which writes them, commits, and pushes — Setzer
is content-agnostic and never needs to know *where* content lives (the editor
does). See [`docs/client-contract.md`](docs/client-contract.md).

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

### Shortcut: reuse your `gh` login

If you already use the [GitHub CLI](https://cli.github.com), tick **“Use my
GitHub CLI (gh) login”** in `/admin` instead of pasting a token — Setzer will use
`gh auth token`. Note this token is broad-scoped (all your repos) and tied to
your `gh` session, so it's best for your own use rather than a least-privilege
or hand-it-to-someone setup.

## Publishing — hosting the site

Setzer commits and **pushes** your edits to the repo. For those edits to appear
on a live site, the repo needs **static hosting that publishes on push** — Setzer
doesn't host; Git and your host do.

The natural pairing is **GitHub Pages**, with no build step:

- Put the site in the repo's `/docs` folder (or root) and add an empty
  **`.nojekyll`** so Pages serves the files verbatim (no Jekyll, no runner).
- Repo **Settings → Pages → Deploy from a branch → `main` / `/docs`**.
- Free Pages requires a **public** repo (private needs a paid plan).
- Set Setzer's **Site directory** to match that folder (e.g. `docs`).

Then every Setzer publish (a push to `main`) is served by Pages automatically.
`wiener-bluut` is set up exactly this way.

## Status

Working end to end and verified against a live repo: content-agnostic multipart
publishing, in-browser image upload with crop, single-active-editor coordination,
safe conflict offload to a branch, a `-dev` loop, and a menu-bar presence with
notifications. `wiener-bluut` is the reference site, served via GitHub Pages.

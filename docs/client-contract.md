# Setzer client contract

How a site speaks to Setzer. A site that follows this is "Setzer-ready": it
serves statically on its own (e.g. GitHub Pages) and, when the local Setzer tool
is running, can publish edits back to its Git repo.

The contract is two interactions over an **opaque content document**. Setzer
never inspects the content's shape — the site owns its schema, design, and
editor. Setzer only serves the files and commits the document.

## 1. Load — read the content

Fetch the content document (plain JSON) at page load and render from it:

```js
const data = await fetch('content.json', { cache: 'no-store' }).then(r => r.json());
```

`content.json` is the single source of truth, committed in the repo. This works
in public hosting with no Setzer present.

## 2. Publish — write the content

When the editor saves, POST the whole content document back to Setzer:

```js
const res = await fetch('/__save', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(data),
});
```

Setzer writes it to the configured content path, commits, and pushes.

### Responses

| Status | Body | Meaning |
|--------|------|---------|
| `200`  | `{ ok: true, commit: "<sha>" }` | committed and pushed |
| `409`  | `{ error, branch, url }` | **the site changed elsewhere** — the edit couldn't fast-forward, so Setzer offloaded it to branch `branch` (push it / merge on GitHub at `url`) and returned to the current published content. Show `error`; offer `url`. |
| other  | `{ error }` | surface the message to the user |

On `409` the editor should surface the message and the `url` (a GitHub compare/PR
link), then reload — the served content is now the current published version.

### Requirements

- **`Content-Type: application/json`** — required (a cross-site form can't set
  it; this is part of Setzer's CSRF defense).
- **Same-origin** — the page Setzer serves; also enforced.

## Inert in public

`/__save` only exists while the local Setzer tool is running. On public hosting
the POST simply fails, so the editor is present but cannot publish. Handle the
failure gracefully (e.g. a toast: "publishing needs the local tool") — no extra
gating needed. Security is by **capability** (the token lives only in Setzer),
not by hiding the editor.

## Reference implementation

`wiener-bluut` — `site/js/app.js`: the `load()` and `publish()` functions and the
editor's "Veröffentlichen" button. The future `setzer new` template will ship
this glue pre-wired.

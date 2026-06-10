# Setzer client contract

How a site speaks to Setzer. A site that follows this is "Setzer-ready": it
serves statically on its own (e.g. GitHub Pages) and, when the local Setzer tool
is running, can publish edits back to its Git repo.

Setzer is **content- and layout-agnostic**. It never inspects what it commits and
holds no knowledge of which file is "the content" — the site owns its schema,
design, editor, and file layout. Setzer only serves the working clone and commits
a **set of files** the editor hands it. `content.json` is a site convention, not
something Setzer knows about.

The site talks to Setzer through the served client library, which Setzer exposes
as `window.Setzer` when — and only when — it is serving the page.

## The served client (`window.Setzer`)

Setzer serves a version-locked client at **`/__setzer/client.js`**. Include it
before your app, and gate the editor on it:

```html
<script src="/__setzer/client.js"></script>   <!-- 404s on public hosting -->
<script src="js/app.js"></script>
```

- Present (`window.Setzer` defined) ⇒ served by Setzer ⇒ editing is available.
- Absent (the script 404s on GitHub Pages) ⇒ no editor at all.

| Member | Purpose |
|--------|---------|
| `version` | client protocol version |
| `load(url)` | fetch + JSON-parse a file (convenience) |
| `publish(files, opts?)` | commit a set of files (below) |
| `isPrimary()` / `onPrimary(cb)` | single-active-editor-tab coordination |

## 1. Load — read the content

Fetch your content at page load and render from it. Works in public hosting with
no Setzer present:

```js
const data = await fetch('content.json', { cache: 'no-store' }).then(r => r.json());
```

## 2. Publish — commit a set of files

When the editor publishes, hand Setzer the files that changed. Each file is a
**web-root-relative path** (the *same* path you use to `fetch`/reference it) plus
its bytes. Setzer writes them all under its serving root and commits them as
**one** commit, then pushes.

```js
await window.Setzer.publish([
  { path: 'content.json',  content: JSON.stringify(data, null, 2) }, // text
  { path: 'img/cover.jpg', content: blob },                          // Blob/File — binary, as-is
], { message: 'Update Termine + cover image' });
```

- `path` — **web-root-relative**, the same path the browser uses. Setzer resolves
  it under its serving root and **sandboxes** it there (`../` neutralised, escapes
  refused). Any path, any file type.
- `content` — a string (text) or a `Blob`/`File`/`ArrayBuffer` (binary), sent
  **as-is** (no base64). The editor owns formatting — e.g. pretty-print JSON
  (`JSON.stringify(data, null, 2)`) for readable commit diffs.
- `opts.message` — optional commit message; the editor knows the semantic change.

### On the wire

`publish()` sends one **`multipart/form-data`** POST to `/__save`: one part per
file, where the part's field **name is the path** and its body is the raw bytes;
an optional `__message` field carries the commit message. You normally just call
`window.Setzer.publish`; this is what it does.

### Responses

| Status | Meaning |
|--------|---------|
| `200`  | `{ ok: true, commit }` — committed and pushed |
| `409`  | `{ error, branch, url }` — **the site changed elsewhere**: the edit couldn't fast-forward, so Setzer offloaded it to branch `branch` (merge on GitHub at `url`) and returned to the current published content. `publish()` rejects with an `Error` whose `.conflict` is this body. |
| other  | `{ error }` — surface the message |

On `409`, surface the message + `url`, then reload — the served content is now the
current published version.

## Security

`/__save` has real side effects (commit + push with your token), and a loopback
bind does **not** stop your own browser from carrying a cross-site request to it
(CSRF). So:

- **Strict `Origin`** — the request is accepted only when `Origin` is present and
  matches Setzer's own. The browser sets `Origin` unforgeably on cross-origin
  requests, so this is the CSRF guard. Missing or foreign `Origin` ⇒ rejected.
  (With `multipart/form-data` there is no `Content-Type` preflight to lean on, so
  the `Origin` check is strict and mandatory.)
- **Path sandbox** — every path is confined under the serving root.
- **Capability** — the Git token lives only in Setzer; the editor never holds it.

## Inert in public

`/__save` and `/__setzer/client.js` exist only while the local Setzer tool runs.
On public hosting they 404, `window.Setzer` is undefined, and the editor is
absent. Security is by capability, not by hiding the editor.

## Reference implementation

`wiener-bluut` — `docs/js/app.js`: gates the editor on `window.Setzer`, builds the
file set, and calls `window.Setzer.publish`. The future `setzer new` template will
ship this glue pre-wired.

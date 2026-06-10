/* Setzer client — served by the running Setzer tool at /__setzer/client.js.
 *
 * Its mere presence (window.Setzer being defined) means "this page is served by
 * Setzer, so editing and publishing are available here." When the same site is
 * served by GitHub Pages, this path 404s and window.Setzer stays undefined — so
 * editors should gate themselves on it. Served from the Setzer binary, so the
 * protocol stays version-locked to the tool.
 */
(function () {
  "use strict";

  var SAVE_URL = "/__save";

  // load fetches and JSON-parses content from url (e.g. "content.json").
  function load(url) {
    return fetch(url, { cache: "no-store" }).then(function (r) {
      if (!r.ok) throw new Error("HTTP " + r.status);
      return r.json();
    });
  }

  // publish commits a set of files. `files` is an array of { path, content },
  // where path is web-root-relative and content is a string (text) or a
  // Blob/File/ArrayBuffer (binary). opts may carry { message }. Sends one
  // multipart/form-data POST (one part per file, field name = path); Setzer
  // writes all files under its serving root and commits them as a single commit.
  // Resolves with the server JSON; on HTTP 409 it rejects with an Error whose
  // `.conflict` is { error, branch, url }.
  function publish(files, opts) {
    opts = opts || {};
    var fd = new FormData();
    files.forEach(function (f) {
      var part = (f.content instanceof Blob) ? f.content : new Blob([f.content]);
      fd.append(f.path, part, f.path);
    });
    if (opts.message) fd.append("__message", opts.message);
    return fetch(SAVE_URL, { method: "POST", body: fd }).then(function (r) {
      return r.json().catch(function () { return {}; }).then(function (body) {
        if (r.status === 409) {
          var ce = new Error((body && body.error) || "Content changed elsewhere.");
          ce.conflict = body || {};
          throw ce;
        }
        if (!r.ok) throw new Error((body && body.error) || ("HTTP " + r.status));
        return body;
      });
    });
  }

  // -- single active editor tab -----------------------------------------------
  // Exactly one open tab holds the "setzer-editor" Web Lock and is "primary"
  // (may edit); others wait. When the primary tab closes, a waiting tab is
  // promoted automatically. Browsers can't focus or close other tabs, so a
  // secondary tab simply stays read-only. No Web Locks support -> assume primary
  // (no coordination, same as before).
  var primary = false;
  var listeners = [];

  function setPrimary(v) {
    if (v === primary) return;
    primary = v;
    listeners.forEach(function (cb) { try { cb(primary); } catch (e) {} });
  }

  // onPrimary(cb) registers a primary-state listener and fires it immediately
  // with the current state.
  function onPrimary(cb) {
    listeners.push(cb);
    try { cb(primary); } catch (e) {}
  }

  if (navigator.locks && navigator.locks.request) {
    navigator.locks.request("setzer-editor", function () {
      setPrimary(true);
      // Hold the lock until this tab goes away: the promise never resolves, so
      // the lock releases only on tab close -> the next waiter is promoted.
      return new Promise(function () {});
    }).catch(function () { /* lock error: stay secondary */ });
  } else {
    setPrimary(true);
  }

  window.Setzer = {
    version: "1",
    load: load,
    publish: publish,
    isPrimary: function () { return primary; },
    onPrimary: onPrimary
  };
})();

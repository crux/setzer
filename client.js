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

  // -- one open editor at a time (always recoverable) -------------------------
  // Coordinated over BroadcastChannel by request-response, NOT Web Locks: a tab
  // that wants to edit asks "anyone editing?" and only a live, responsive editor
  // answers. A closed, crashed, or HUNG holder can't answer, so the slot is
  // simply free — no lock can get stuck. An open editor can always be stolen by
  // the human; the stolen tab stands down. Worst case (a rare simultaneous open)
  // is two edits, which the git layer offloads to a branch — never lost.
  var chan = ("BroadcastChannel" in window) ? new BroadcastChannel("setzer-editing") : null;
  var myId = Math.random().toString(36).slice(2);
  var editing = false;
  var onStolen = null;

  if (chan) chan.addEventListener("message", function (e) {
    var m = e.data || {};
    if (m.type === "who" && editing) {
      chan.postMessage({ type: "here", id: myId }); // I'm the editor — answer the probe
    } else if (m.type === "take" && m.from !== myId && editing) {
      editing = false;                              // someone took over — stand down
      if (onStolen) { try { onStolen(); } catch (e) {} }
    }
  });

  // beginEdit(ok, busy): probe for a live editor (~250ms, waiting for an answer).
  // ok() if the slot is free — or the holder is gone/hung and can't answer; else
  // busy(steal), where steal() forces a takeover and then opens.
  function beginEdit(ok, busy) {
    if (!chan) { editing = true; ok(); return; }
    var answered = false;
    var probe = function (e) { if ((e.data || {}).type === "here") answered = true; };
    chan.addEventListener("message", probe);
    chan.postMessage({ type: "who", from: myId });
    setTimeout(function () {
      chan.removeEventListener("message", probe);
      if (answered) {
        busy(function steal() {
          chan.postMessage({ type: "take", from: myId });
          editing = true;
          ok();
        });
      } else {
        editing = true;
        ok();
      }
    }, 250);
  }

  function endEdit() { editing = false; } // release the slot when the editor closes

  // onEditorStolen(cb): fired when another tab takes over — the site should close
  // its editor.
  function onEditorStolen(cb) { onStolen = cb; }

  window.Setzer = {
    version: "2",
    load: load,
    publish: publish,
    beginEdit: beginEdit,
    endEdit: endEdit,
    onEditorStolen: onEditorStolen
  };
})();

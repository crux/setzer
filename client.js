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

  // publish sends content to Setzer, which commits and pushes it. Resolves with
  // the server JSON on success. On HTTP 409 it rejects with an Error whose
  // `.conflict` is { error, branch, url } — the edit was offloaded to a branch
  // for the human to merge on GitHub.
  function publish(data) {
    return fetch(SAVE_URL, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(data)
    }).then(function (r) {
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

  window.Setzer = { version: "1", load: load, publish: publish };
})();

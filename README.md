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

> Status: early MVP — not yet usable.

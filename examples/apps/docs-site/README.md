# docs-site

A Mintlify-style docs site: sidebar nav loaded from `nav.json`, content
pages rendered from markdown with Prism syntax highlighting.

## Run it

```sh
wave serve examples/apps/docs-site/server.yaml --port 8402
```

## Try it

- http://127.0.0.1:8402/                              — the docs shell
- http://127.0.0.1:8402/docs/getting-started.md       — direct page
- http://127.0.0.1:8402/docs/api-reference.md         — see the syntax highlighter
- http://127.0.0.1:8402/assets/nav.json               — raw sidebar config

## What to look at

- `server.yaml` — three routes: `file` (shell), `file-server`
  (markdown rendering), `static` (assets).
- `assets/nav.json` — sidebar definition. Edit and refresh to add pages.
- `docs/*.md` — drop a new file here, then add it to `nav.json` to
  surface it in the sidebar.
- `web/index.html` — vanilla JS fetches `nav.json` and updates an iframe.

## Caveats

- The main pane is an `<iframe>` so each page navigation is a full
  document load. A real Mintlify clone would render the markdown into
  the same DOM. The iframe approach keeps this example free of any
  client-side markdown parser.
- Search is filename/label-based only — no full-text index.

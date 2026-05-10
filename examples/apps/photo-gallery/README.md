# photo-gallery

A static photo gallery: thumbnails (lazy-loaded) on the index page,
click anything to open a lightbox.

## Run it

```sh
wave serve examples/apps/photo-gallery/server.yaml --port 8404
```

## Try it

- http://127.0.0.1:8404/                       — the gallery
- http://127.0.0.1:8404/photos/manifest.json   — the photo list
- http://127.0.0.1:8404/photos/01-sunset.svg   — direct image link

## What to look at

- `server.yaml` — three `static`/`file` routes. Wave does no work
  beyond serving bytes; all behavior is in the client.
- `photos/manifest.json` — declarative list of `{ file, caption }`.
  Add a new entry + drop the matching file in `photos/` to extend.
- `web/index.html` — vanilla JS fetches the manifest and renders the
  grid with `loading="lazy"` and a small lightbox overlay.

## Caveats

- The placeholder images are tiny SVGs to keep the repo small. Drop in
  real JPEGs and update `manifest.json`.
- `manifest.json` is hand-maintained. A "real" version would generate
  the listing server-side (e.g. via a `storage-access` route over the
  filesystem backend).

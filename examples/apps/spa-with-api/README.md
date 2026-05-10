# spa-with-api

A vanilla-JS SPA + JSON API. Wave's startup bundler concatenates the
four `frontend/src/*.js` files into a single cache-busted bundle, then
auto-serves the resulting `dist/` directory at `/`.

## Run it

```sh
wave serve examples/apps/spa-with-api/server.yaml --port 8403
```

You should see a `[BUNDLE]` log line at startup confirming the build.

## Try it

- http://127.0.0.1:8403/app        — the SPA (loads `app.bundle.js?v=…`)
- http://127.0.0.1:8403/api/items  — the JSON the SPA fetches
- http://127.0.0.1:8403/app.bundle.js — the concatenated, minified bundle
- Look at `dist/index.html` after startup — Wave generated it from your
  `frontend/index.html` and injected the bundle script tag.

## What to look at

- `server.yaml` — the `build:` block is the entire bundler config.
  Note we only declare ONE explicit route (`/api/items`); the bundler
  auto-registers `/`.
- `frontend/src/*.js` — concatenated in the order listed under
  `js_files`. `util.js` first (defines globals), then `store`, `views`,
  `app`. Each later file uses the `window.Wave*` globals defined above.
- `frontend/index.html` — your HTML shell. The bundler injects
  `<script src="/app.bundle.js?v=...">` just before `</body>`.

## Caveats

- `dist/` is regenerated every startup — don't commit it.
- No framework, no ES modules. The bundler doesn't transpile or
  resolve imports, so we use `window.*` globals.
- The bundler auto-mounts `dist/` at `/` via a static route, but Wave's
  static handler does not auto-resolve directory-default `index.html`,
  so we expose the SPA at `/app` via an explicit `file` route.

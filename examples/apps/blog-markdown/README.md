# blog-markdown

A minimal markdown blog. Posts live as `.md` files in `posts/`; Wave
renders each one to HTML with Prism syntax highlighting.

## Run it

```sh
wave serve examples/apps/blog-markdown/server.yaml --port 8401
```

## Try it

- http://127.0.0.1:8401/                    — index page
- http://127.0.0.1:8401/posts/welcome.md    — a rendered post
- http://127.0.0.1:8401/posts/              — directory listing

## What to look at

- `server.yaml` — three routes: `file` (templated index), `file-server`
  (markdown rendering with `prettify: true`), and `static` (CSS).
- `posts/*.md` — frontmatter is included for human readers. Wave's
  current renderer displays it as part of the markdown body; treat the
  frontmatter as metadata you read with your eyes for now.
- `web/index.html` — uses `{{.Request.Host}}` to demonstrate template
  variable injection.

## Caveats

- The post list on `/` is hand-curated. Wave doesn't yet have a built-in
  loop-over-directory template helper, so adding a post means editing
  `web/index.html` as well as dropping a file in `posts/`.
- Frontmatter is rendered as text (no YAML stripping pass).

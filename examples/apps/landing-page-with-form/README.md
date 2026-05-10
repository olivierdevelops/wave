# landing-page-with-form

A marketing landing page with a `Contact us` form. The form POSTs to
`/contact`, which validates the inputs, inserts a row into SQLite via
`storage-access`, and returns a thank-you HTML page rendered from the
inline `output_template`.

## Run it

```sh
wave serve examples/apps/landing-page-with-form/server.yaml --port 8405
```

A `contacts.db` SQLite file is created next to `server.yaml` on first
boot.

## Try it

- http://127.0.0.1:8405/             — landing page + form
- Submit the form → see the thank-you page with the new row id.
- http://127.0.0.1:8405/submissions  — JSON listing of every submission

## What to look at

- `server.yaml` — `storage:` declares a SQLite backend with an inline
  schema, three routes wire it up: `file` (page), `storage-access`
  (form POST), `storage-access` (JSON list).
- The POST route's `inputs:` validate length and presence — try
  submitting an empty `name` to see Wave reject it with a 400.
- The thank-you page is rendered straight from `output_template` —
  no separate HTML file needed.

## Caveats

- `contacts.db` is created in the working directory and persists
  across restarts. Delete it to start fresh. Already gitignored.
- For real production use, add CSRF protection (`validate_csrf: true`
  on the POST route + `include_csrf: true` on the page route).

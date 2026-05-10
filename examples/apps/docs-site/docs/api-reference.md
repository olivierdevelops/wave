# API reference

Reference for every route type. For full details see the source under
`usecases/<type>/`.

## `static`

Serves a directory of files raw.

```yaml
- path: /assets/
  type: static
  static:
    dir: ./assets
    file_ignore_patterns: [".env", ".git"]
```

## `file`

Serves a single file. Set `is_template: true` to evaluate it as a Go
`html/template`. Receives a context with `Request`, `Query`, `Headers`.

```yaml
- path: /
  type: file
  file:
    filepath: ./index.html
    is_template: true
```

## `file-server`

Like `static`, but also renders directory indexes and (with
`prettify: true`) converts markdown to HTML on the fly.

```yaml
- path: /docs/
  type: file-server
  file-server:
    dir: ./docs
    prettify: true
```

## `content`

Inline body. The simplest possible "hello world".

```yaml
- path: /hello
  type: content
  content:
    body: "Hello!"
    headers:
      - ["Content-Type", "text/plain"]
```

## `storage-access`

Run a SQL template against a configured `storage:` backend, render the
result as JSON or HTML.

```yaml
- path: /notes
  type: storage-access
  storage-access:
    source: notes
    execute: "SELECT id, title FROM notes"
    response_content_type: application/json
    output_template: "{{toJSON .Data}}"
```

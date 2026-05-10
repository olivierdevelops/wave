# Concepts

Wave is a **declarative HTTP server**: a single YAML file describes every
route, and Wave wires them up at startup.

## Routes

Every entry under `routes:` has three required fields:

| Field    | Description                                             |
| -------- | ------------------------------------------------------- |
| `path`   | URL path (path params allowed: `/users/{id}`)           |
| `method` | HTTP verb. Omit to register a multi-method handler.     |
| `type`   | One of `static`, `file`, `forward`, `api`, `content`, … |

## Route handlers

Each `type` selects a matching handler block. For example, `type: file`
expects a `file:` block:

```yaml
- path: /
  type: file
  file:
    filepath: ./web/index.html
    is_template: true
```

## Inputs

Add an `inputs:` list to validate path/query/body params:

```yaml
inputs:
  - { name: id, source: path, type: int, required: true }
```

Wave coerces and validates each input *before* the handler runs and rejects
bad requests with a single 400 listing every problem at once.

## Storage

The top-level `storage:` map declares SQLite or filesystem backends that
`type: storage-access` routes can target. See the API reference for the
full list of operations.

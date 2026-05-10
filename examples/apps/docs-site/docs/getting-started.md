# Getting started

Welcome to the Wave docs. This page walks you through your **first server**
in under five minutes.

## 1. Install

```sh
go install wave/orchestrator@latest
```

## 2. Write a server.yaml

Create a file with this content:

```yaml
default:
  host: 127.0.0.1
  port: 8080

routes:
  - path: /hello
    method: GET
    type: content
    content:
      body: "Hello from Wave!"
```

## 3. Start the server

```sh
wave serve server.yaml
```

Visit <http://127.0.0.1:8080/hello> and you should see the greeting.

## Next steps

- Read the [Concepts](#concepts) page for the mental model
- Browse the [API Reference](#api-reference) for every route type

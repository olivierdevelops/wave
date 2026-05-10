---
title: A quick tour of markdown features
date: 2026-04-08
tags: [markdown, syntax]
---

# A quick tour of markdown features

## Headings, lists and emphasis

You get the standard fare: **bold**, *italics*, ~~strike~~, `inline code`.

1. Ordered lists
2. work as you expect
3. with arabic numerals

- And bullet lists
- nest cleanly:
  - sub-bullet one
  - sub-bullet two

## Code blocks with syntax highlighting

```go
func main() {
    fmt.Println("Hello from Wave!")
}
```

```javascript
const greet = (name) => `Hello, ${name}!`;
console.log(greet('Wave'));
```

## Tables

| Route type     | What it does                          |
| -------------- | ------------------------------------- |
| `static`       | Serve a directory of files raw         |
| `file`         | Serve one file with optional templating|
| `file-server`  | Browse a directory, render markdown    |

## Links and images

[Visit the Wave repo](https://example.com/wave) — links work like you'd expect.

That's it. Markdown in, HTML out.

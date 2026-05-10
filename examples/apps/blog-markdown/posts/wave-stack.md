---
title: Why we chose Wave for this site
date: 2026-04-15
tags: [wave, deployment]
---

# Why we chose Wave for this site

We previously used a static-site generator that required:

- A Node.js toolchain
- A CI pipeline to rebuild on every push
- A separate CDN for serving the output

With Wave, our entire blog is:

```yaml
routes:
  - path: /
    type: file
    file: { filepath: ./web/index.html, is_template: true }
  - path: /posts/
    type: file-server
    file-server: { dir: ./posts, prettify: true }
```

Add a `.md` file → it's live. No rebuilds. The server **is** the build step.

## Trade-offs

This approach is great for small-to-medium content sites. If you need
incremental builds across thousands of posts or fancy plugins, you'll
still want Hugo. But for a blog that ships in 60 seconds, Wave wins.

# Deploy with Docker

## The official image

```sh
docker run --rm -p 8080:8080 \
  -v $(pwd)/server.capy:/app/server.capy \
  ghcr.io/olivierdevelops/wave:latest \
  serve /app/server.capy --listen :8080
```

- Distroless nonroot base (~25 MB)
- CGO-enabled, so the built-in SQLite backend works
- Image tags: `latest`, `vX.Y.Z`, `vX.Y.Z-amd64`, `vX.Y.Z-arm64`
- Multi-arch: amd64 + arm64 picked automatically by Docker

## Long-running container

Same image, run with a named volume for the SQLite data dir, a
read-only mount for `server.capy`, a restart policy, and a healthcheck
that re-execs the binary's `version` subcommand:

```sh
docker volume create wave_data

docker run -d \
  --name wave \
  --restart unless-stopped \
  -p 8080:8080 \
  -v "$(pwd)/server.capy:/app/server.capy:ro" \
  -v wave_data:/app/data \
  -e JWT_SECRET=change-me-in-prod \
  --health-cmd '/usr/local/bin/wave version' \
  --health-interval 30s --health-timeout 3s --health-retries 3 \
  ghcr.io/olivierdevelops/wave:latest \
  serve /app/server.capy --listen :8080

docker logs -f wave
```

## Custom image (build from your repo)

If your `server.capy` lives in your own repo, bake it into a slim
derived image:

```dockerfile
FROM ghcr.io/olivierdevelops/wave:latest
COPY server.capy /app/server.capy
COPY assets /app/assets
CMD ["serve", "/app/server.capy", "--listen", "0.0.0.0:8080"]
```

## Volumes you'll typically want

| Mount point | Purpose |
|---|---|
| `/app/server.capy` | The config (read-only) |
| `/app/data` | SQLite DB files (persistent) |
| `/app/outbox.db` | Outbox DB if `outbox_db:` is set |
| `/app/assets` | Static files, templates |

Mount `/app/data` to a host volume or a managed volume — losing
the SQLite file means losing your data.

## Pinning a version

```sh
docker run ghcr.io/olivierdevelops/wave:v0.1.0 ...
```

Pinning is recommended for production. Wave is pre-1.0; reading the
[CHANGELOG](https://github.com/olivierdevelops/wave/blob/main/CHANGELOG.md)
before bumping is the contract.

## Security notes

- The image runs as **UID 65532** (nonroot). Mount volumes with
  matching ownership.
- Distroless contains **no shell** — `docker exec wave sh` won't
  work. Debug with a sidecar or by switching tags to `:debug` if
  ever published.
- The image **does not phone home**. See the [Privacy page](/guide/privacy).

## Verifying the image

Image signatures land alongside binaries via sigstore keyless OIDC:

```sh
cosign verify ghcr.io/olivierdevelops/wave:v0.1.0 \
  --certificate-identity-regexp='https://github.com/olivierdevelops/wave/' \
  --certificate-oidc-issuer='https://token.actions.githubusercontent.com'
```

## See also

- [Fly.io deploy](/guide/deploy-fly)
- [Production checklist](/guide/deploy-checklist)
- [Observability](/guide/concepts-observability) — wire `/metrics`
  scraping

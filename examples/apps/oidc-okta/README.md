# oidc-okta

OpenID Connect via an Okta tenant. At boot Wave fetches Okta's
discovery document (`/.well-known/openid-configuration`), caches
the JWKS, and validates the ID token signature on every request
to a protected route.

## Setup

1. In Okta admin → Applications, create a new app integration.
2. Note the **Okta domain** (e.g. `dev-12345.okta.com`).
3. Note the **client ID**.

## Run it

```sh
SECRET_KEY=dev-key \
OKTA_DOMAIN=dev-12345.okta.com \
OKTA_CLIENT_ID=0oa1b2c3d4 \
wave serve examples/apps/oidc-okta/server.yaml --port 8006
```

## Try it

Get an ID token however your client normally does (e.g. authorization
code flow from your SPA). Then:

```sh
curl http://localhost:8006/api/whoami \
  -H "Authorization: Bearer <ID_TOKEN>"
```

A valid token returns `{"ok": true, ...}`; an invalid or expired
token returns 401.

## What to look at

- `auth.okta.type: oidc` — selects the OIDC verifier.
- `issuer: "https://${ENV:OKTA_DOMAIN}"` — Wave does discovery against this.
- `client_id` — used as the expected `aud` claim during verification.
- `token_location: header` — Bearer tokens, no cookies.

## Caveats

- Boots only when `OKTA_DOMAIN` and `OKTA_CLIENT_ID` are set; the
  discovery fetch happens at startup and fails fast on a bad URL.
- `SECRET_KEY` must also be set (used internally by AuthManager).

# jwt-validating-gateway

A reverse proxy that validates an inbound HS256 JWT on every request
before forwarding to an upstream that doesn't do auth itself. Useful
when you want a single-line "make this upstream require auth" layer
without modifying the upstream.

## Run it

```sh
SECRET_KEY=devsecret wave serve examples/apps/jwt-validating-gateway/server.yaml
```

Listens on `http://localhost:8304`.

## Try it — rejection path

Without a token — Wave's auth middleware rejects with `401` BEFORE the
upstream is contacted:

```sh
curl -i http://localhost:8304/api/get
```

```sh
# Tampered token also rejected.
curl -i http://localhost:8304/api/get -H "Authorization: Bearer not.a.real.token"
```

## Mint a token (HS256, signed with $SECRET_KEY)

Wave's JWT format is HS256 with claims `{user_id, username, session_id, exp}`.
You can hand-roll one with `openssl`:

```sh
SECRET=devsecret
HEADER=$(printf '{"alg":"HS256","typ":"JWT"}' | openssl base64 -A | tr '+/' '-_' | tr -d '=')
PAYLOAD=$(printf '{"user_id":-1,"username":"alice","session_id":"s1","exp":9999999999}' \
  | openssl base64 -A | tr '+/' '-_' | tr -d '=')
SIG=$(printf '%s' "$HEADER.$PAYLOAD" | openssl dgst -sha256 -hmac "$SECRET" -binary \
  | openssl base64 -A | tr '+/' '-_' | tr -d '=')
TOKEN="$HEADER.$PAYLOAD.$SIG"

curl -i http://localhost:8304/api/get -H "Authorization: Bearer $TOKEN"
```

A hand-rolled token still gets rejected, because Wave's JWT auth
double-checks the embedded `session_id` against its in-memory session
store. To produce a token that passes the session check, drive Wave's
`auth-login` route flow in your app — that returns a token whose
`session_id` is registered. The validation-and-rejection behaviour
above is the load-bearing demo.

## What to look at

- `auth.default` in the top-level `auth:` block — defines a JWT
  validator (HS256, header-borne, `Bearer` scheme).
- `auth: [default]` on each route — gates the route on that validator.
  Wave wires the middleware automatically.

## Caveats

- HS256 is fine for "shared secret between two services" but not for a
  public token issuer. Use `oidc` for JWKS-based RS256 validation.
- The user ID from claims is available on the request context inside
  Wave; surfacing it as a downstream header (e.g. `X-User-Id`) for the
  upstream to consume requires either a small custom plugin step or
  `forward_auth` against a sidecar. Out of scope for this example.

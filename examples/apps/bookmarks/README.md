# bookmarks

Multi-user bookmarks where each row is scoped to the authenticated
user's `Username`. Demonstrates `auth:` middleware combined with the
SQL template helper `getUser` and `inputs:` body validation.

## Run it

```sh
wave serve examples/apps/bookmarks/server.yaml --port 8103
```

## Try it

```sh
COOKIE_JAR=/tmp/bm.cookies

# Login (cookie persisted in COOKIE_JAR).
curl -s -c $COOKIE_JAR -X POST localhost:8103/login \
  -H 'content-type: application/x-www-form-urlencoded' \
  -d 'username=alice&password=alicepw'

# Add a bookmark — owner is taken from the JWT, NOT the request body.
curl -s -b $COOKIE_JAR -X POST localhost:8103/bookmarks \
  -H 'content-type: application/json' \
  -d '{"title":"wave","url":"https://github.com/luowensheng/easyserver"}'

# List MY bookmarks.
curl -s -b $COOKIE_JAR localhost:8103/bookmarks
```

## What to look at

- `auth:` declared at the top level, then referenced per route via
  `auth: [user]`.
- The SQL template uses `getUser.Username` wrapped with `wrap` so the
  username flows through as a bound `?` parameter.
- `inputs:` enforces shape on the POST body and locks template scope.

## Caveats

- Real magic-link auth needs `auth_flows.smtp.*` wiring; this example
  uses pre-seeded passwords for simplicity.
- The default JWT secret is in plaintext — swap for `${ENV:JWT_SECRET}`
  before exposing this beyond localhost.

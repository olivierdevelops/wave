# password-jwt

Username/password sign-in. Wave validates credentials against
the in-config `default_logins` table, mints a JWT, and stores it
in an HttpOnly cookie. Hitting a protected route checks the JWT.

## Run it

```sh
SECRET_KEY=dev-key wave serve examples/apps/password-jwt/server.yaml --port 8002
```

## Try it

Browser: open `http://localhost:8002/login`, sign in as `admin/admin`,
land on `/me`.

Curl:

```sh
curl -i -X POST http://localhost:8002/auth/login \
  -d 'username=admin&password=admin'
# capture the wave_session cookie, then:
curl -i --cookie 'wave_session=<JWT>' http://localhost:8002/me
```

## What to look at

- `auth.session.type: jwt` — the JWT/cookie session backend.
- `default_logins:` — two demo users baked into the config.
- `auth-login.for: session` — binds the login route to the session block.
- `auth: ["session"]` on `/me` — the line that requires authentication.

## Caveats

- `SECRET_KEY` env var must be set (used to sign the JWT).
- `default_logins` are dev-only. Wire `user_store: sqlite` for real users.

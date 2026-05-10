# magic-link-login

Passwordless email sign-in. The user submits an email, the server
"sends" a magic link (in dev mode the link is logged to stderr by
the console mailer), and clicking the link logs them in.

## Run it

```sh
SECRET_KEY=dev-key wave serve examples/apps/magic-link-login/server.yaml --port 8001
```

## Try it

1. Open `http://localhost:8001` in a browser, type any email, hit
   "send link".
2. Look at the server stderr — the console mailer prints the full
   email body, including the `?token=...` link.
3. Open the link. You'll be redirected to `/me` with a session cookie.

Or with curl:

```sh
curl -X POST http://localhost:8001/auth/login/request \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com"}'
# then grab the token from server stderr and:
curl -i 'http://localhost:8001/auth/login/consume?token=<TOKEN>'
```

## What to look at

- `type: magic-link-request` route — issues the token and sends the email.
- `callback_url` — where the link points; must match the consume route.
- `email_body` — Go text/template; `.Link`, `.Email`, `.MinutesValid` available.
- `type: magic-link-consume` route — validates the token and creates a session.
- `auth.session` block — the JWT cookie session the consume route hands out.

## Caveats

- `SECRET_KEY` env var must be set (used to sign session JWTs).
- No real mail server. Add `auth_flows.smtp.host` to wire SMTP.

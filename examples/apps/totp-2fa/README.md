# totp-2fa

Magic-link login as the first factor, TOTP (Google Authenticator,
1Password, …) as the second. Demonstrates the `totp-enroll-start`,
`totp-enroll-confirm`, and `totp-verify` route types layered on
top of an existing session.

## Run it

```sh
SECRET_KEY=dev-key wave serve examples/apps/totp-2fa/server.yaml --port 8008
```

## Try it

```sh
# 1. Request a magic link
curl -X POST http://localhost:8008/auth/login/request \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com"}'
# (grab the link from server stderr)

# 2. Open the link in a browser to set the wave_session cookie
#    (or use curl -c cookies.txt -L on the link)

# 3. Start enrollment — returns {"secret":"...","otpauth_url":"otpauth://..."}
curl -b cookies.txt -X POST http://localhost:8008/totp/enroll

# 4. Scan the otpauth_url into Google Authenticator / 1Password,
#    then confirm with the first 6-digit code:
curl -b cookies.txt -X POST http://localhost:8008/totp/confirm \
  -H 'Content-Type: application/json' -d '{"code":"123456"}'

# 5. Verify a fresh code to clear the second factor
curl -b cookies.txt -X POST http://localhost:8008/totp/verify \
  -H 'Content-Type: application/json' -d '{"code":"654321"}'
```

## What to look at

- `type: magic-link-request` / `magic-link-consume` — first factor.
- `type: totp-enroll-start` — generates a secret and otpauth URL.
- `type: totp-enroll-confirm` — persists the secret after one valid code.
- `type: totp-verify` — checks a code; standalone step-up endpoint.
- All three TOTP routes carry `auth: ["session"]` — they only run for
  users who've already cleared the first factor.

## Caveats

- `SECRET_KEY` env var must be set.
- The TOTP store is in-memory; restart wipes enrollments. Wire a
  persistent store for production.
- Magic-link emails go to stderr (console mailer).

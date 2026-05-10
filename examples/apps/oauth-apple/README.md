# oauth-apple

Sign in with Apple. Apple replaces the usual `client_secret` with
a per-request JWT signed by an ECDSA key you download from Apple
Developer; Wave handles that signing internally.

## Setup

1. In Apple Developer → Certificates, IDs & Profiles, create:
   - An **App ID** with "Sign In with Apple" enabled.
   - A **Service ID** (this becomes `APPLE_SERVICE_ID`/`client_id`)
     with the redirect URL `http://localhost:8005/auth/apple/callback`.
   - A **Sign-In with Apple Key** — download the `AuthKey_<KEYID>.p8`
     file. Note the 10-char Key ID.
2. Your 10-char Team ID is in the upper-right of the developer portal.

## Run it

```sh
SECRET_KEY=dev-key \
APPLE_SERVICE_ID=com.example.web \
APPLE_TEAM_ID=ABCD123456 \
APPLE_KEY_ID=ABCDE12345 \
APPLE_PRIVATE_KEY_PATH=/secrets/AuthKey_ABCDE12345.p8 \
wave serve examples/apps/oauth-apple/server.yaml --port 8005
```

## Try it

`http://localhost:8005/login` → "Continue with Apple" → Apple consent
→ `/me`.

Apple only sends real localhost in dev if you've configured a tunnel
or `127.0.0.1` is whitelisted on the Service ID — in practice you'll
want an HTTPS tunnel (e.g. ngrok) for end-to-end verification.

## What to look at

- `oauth.provider: apple` — selects the Apple provider, which signs
  a JWT client_secret per request from the .p8 key.
- `apple_team_id` / `apple_key_id` / `apple_private_key_path` —
  three Apple-specific knobs not present on other providers.
- `client_id` is the Service ID (not the App ID).

## Caveats

- Boots only when all four `APPLE_*` env vars are set. The .p8 file
  must be readable at the path provided.
- `SECRET_KEY` must also be set.

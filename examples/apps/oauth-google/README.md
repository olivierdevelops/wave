# oauth-google

Sign in with Google. Wave handles the full OAuth 2 dance:
authorize redirect, code → token exchange, userinfo fetch,
session cookie.

## Setup

1. Create OAuth credentials in the Google Cloud Console
   (APIs & Services → Credentials → OAuth client ID, type "Web app").
2. Authorized redirect URI:
   `http://localhost:8003/auth/google/callback`
3. Copy the client ID and secret.

## Run it

```sh
SECRET_KEY=dev-key \
GOOGLE_CLIENT_ID=...apps.googleusercontent.com \
GOOGLE_CLIENT_SECRET=... \
wave serve examples/apps/oauth-google/server.yaml --port 8003
```

## Try it

Open `http://localhost:8003/login` and click "Continue with Google".
After consent you're redirected to `/me`.

## What to look at

- `auth.google.type: oauth` — selects the OAuth backend.
- `oauth.provider: google_oauth` — Google-specific authorize/token/userinfo URLs.
- `type: oauth-start` — the route that builds the authorize URL with state.
- `type: oauth-callback` — exchanges the code, fetches userinfo, sets cookie.
- `redirect_uri` — must exactly match what's configured in Google Console.

## Caveats

- Boots only when `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` are set;
  the OAuth provider is built at startup.
- `SECRET_KEY` must also be set (used to sign session JWTs).

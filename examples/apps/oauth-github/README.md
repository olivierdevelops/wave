# oauth-github

Sign in with GitHub. Same shape as `oauth-google` but with the
GitHub-specific `read:user` and `user:email` scopes.

## Setup

1. Create an OAuth App at <https://github.com/settings/developers>.
2. Authorization callback URL:
   `http://localhost:8004/auth/github/callback`
3. Note the client ID and generate a client secret.

## Run it

```sh
SECRET_KEY=dev-key \
GITHUB_CLIENT_ID=Iv1.abc123 \
GITHUB_CLIENT_SECRET=ghs_xxx \
wave serve examples/apps/oauth-github/server.yaml --port 8004
```

## Try it

Open `http://localhost:8004/login`, click "Continue with GitHub",
authorize the app — you land on `/profile`.

## What to look at

- `oauth.provider: github` — selects the GitHub provider implementation.
- `scopes: [read:user, user:email]` — what we ask for; `user:email` is
  needed to receive the user's primary email even if it's private.
- `success_redirect: /profile` — where the callback drops you on success.
- `auth: ["github"]` on `/profile` — the line that requires GitHub auth.

## Caveats

- Boots only when `GITHUB_CLIENT_ID` and `GITHUB_CLIENT_SECRET` are set.
- `SECRET_KEY` must also be set.

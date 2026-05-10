# magic-link-plus-totp

Two-factor sign-in: passwordless magic-link establishes a base session,
then TOTP verification upgrades it to a "trusted" session. Routes that
hold sensitive data require both factors.

## What it shows off

- `type: magic-link-request` + `type: magic-link-consume` — passwordless step 1.
- `type: totp-enroll-start` / `totp-enroll-confirm` / `totp-verify` — second-factor step.
- `auth: [session]` plus `require_claims: { trusted: "yes" }` — claim-based RBAC gate.

## Run

```sh
SECRET_KEY=devsecret \
  wave serve examples/apps/magic-link-plus-totp/server.yaml --port 8601
```

The default `console` mailer logs the magic link to stderr — copy it
into a browser. After signing in, POST to `/totp/enroll` then
`/totp/confirm`, and finally `/totp/verify` with each new code from the
authenticator app to upgrade the session.

## Caveats

- The `trusted` claim is added by the totp-verify handler; older clients
  that haven't re-authenticated will continue to see `/vault` 403.
- For real deployments wire `auth_flows.smtp` so links go to email.

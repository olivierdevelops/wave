# slack-slash-command

POST `/slack/command` receives a Slack slash-command, verifies the
`X-Slack-Signature` header against `SLACK_SIGNING_SECRET`, and returns
a Slack-formatted JSON response.

## What it shows off

- `webhook_sig.provider: slack` — built-in Slack signature verifier
  (HMAC-SHA256 over `v0:timestamp:body`).
- `tolerance_sec: 300` — replay-window protection.
- `type: content` returning Slack Block-Kit JSON.

## Run

```sh
SLACK_SIGNING_SECRET=your-secret wave serve \
  examples/apps/slack-slash-command/server.yaml --port 8606
```

In your Slack app config, point the slash command (e.g. `/wave-status`)
at `https://your-public-host/slack/command`.

## Caveats

- Requires `SLACK_SIGNING_SECRET`. Without it the route boots but
  every request fails signature verification.
- For local testing use `ngrok http 8606` to expose the endpoint.

# email-to-api-ingest

Inbound email handler. Mailgun and SendGrid Inbound Parse routes can
POST form-encoded email content to a public URL; this app validates the
fields with `inputs:` and persists them to SQLite.

## What it shows off

- `inputs.source: form` pulling values out of the multipart body.
- `from: "body-plain"` overriding the input key (Mailgun's hyphenated
  field name).
- Bounded `max:` lengths to refuse runaway messages.
- `type: storage-access` parameterised INSERT.

## Run

```sh
wave serve examples/apps/email-to-api-ingest/server.yaml --port 8608

curl -X POST http://127.0.0.1:8608/email/inbound \
  -F 'from=alice@example.com' \
  -F 'subject=hello' \
  -F 'body-plain=Greetings from your inbox.'

curl http://127.0.0.1:8608/email/recent
```

## Mailgun setup

1. Create a Receiving Route, action `forward("https://your-host/email/inbound")`.
2. Mailgun POSTs `from`, `subject`, `body-plain`, etc. as multipart form data.

## Caveats

- No HMAC verification configured in this minimal example. Add
  `webhook_sig.provider: generic` with Mailgun's signing key for prod.
- Attachments are not stored; only the plain-text body.

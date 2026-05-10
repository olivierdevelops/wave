# vault-secrets-fed

Wave with no plaintext secrets in YAML: the JWT signing key and the
SSE subscribe token are pulled from Vault at boot via the
`vault-secrets` reference plugin (`${PLUGIN:vault:...}` markers).

## Setup

1. Build the plugin:

   ```sh
   cd examples/plugins/vault-secrets && go build -o /tmp/wave-vault .
   ```

2. Run Vault in dev mode and write the two secrets the config refers to:

   ```sh
   vault server -dev -dev-root-token-id=devroot &
   export VAULT_ADDR=http://127.0.0.1:8200
   export VAULT_TOKEN=devroot
   vault kv put secret/wave \
     jwt_secret=$(openssl rand -hex 32) \
     stream_token=$(openssl rand -hex 16)
   ```

## Run it

```sh
wave serve examples/apps/vault-secrets-fed/server.yaml --port 8702
```

## Try it

```sh
# Login (sets a JWT cookie signed with the Vault-fed secret)
curl -c /tmp/jar -XPOST localhost:8702/auth/login -d 'username=admin&password=admin'
curl -b /tmp/jar localhost:8702/me

# Subscribe with the vault-fed token
curl -H "Authorization: Bearer $(vault kv get -field=stream_token secret/wave)" \
  localhost:8702/events
```

## What to look at

`auth.session.secret` and `connections.events.subscribe_auth_token`
both contain `${PLUGIN:vault:...}`. These markers survive the
pre-parse env/file pass; the orchestrator resolves them after the
secrets-kind plugin boots, then walks the `Config` struct and
substitutes in place.

## Caveats

Plugin `command:` paths cannot contain `${PLUGIN:...}` (chicken-and-egg).
If `VAULT_ADDR` is unreachable or the path/key is missing, boot fails
fast with the unresolved marker in the error.

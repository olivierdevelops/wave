# saml-sso

Enterprise SAML 2.0 single sign-on. The SAML protocol bits live
in the `saml-auth` plugin (a separate Go subprocess); the
orchestrator owns sessions, cookies, and JWTs as usual.

## Build the plugin

```sh
cd examples/plugins/saml-auth
go build -o wave-saml .
```

## Setup

You'll need an IdP that publishes SAML metadata (Okta, Azure AD,
ADFS, Auth0, JumpCloud, …) and a self-signed SP cert/key pair.

Generate an SP cert/key for local testing:

```sh
openssl req -x509 -newkey rsa:2048 -nodes \
  -keyout sp.key -out sp.crt -days 365 -subj '/CN=localhost'
```

Configure your IdP with:
- Entity ID: `http://localhost:8007/saml`
- ACS URL:   `http://localhost:8007/saml/acs`

## Run it

```sh
SECRET_KEY=dev-key \
SAML_IDP_METADATA_URL=https://idp.example.com/metadata \
SAML_SP_ENTITY_ID=http://localhost:8007/saml \
SAML_SP_ACS_URL=http://localhost:8007/saml/acs \
SAML_SP_CERT_PATH=$PWD/sp.crt \
SAML_SP_KEY_PATH=$PWD/sp.key \
wave serve examples/apps/saml-sso/server.yaml --port 8007
```

## Try it

1. Open `http://localhost:8007/login`, click "Sign in via SAML".
2. You're redirected to the IdP, authenticate, and bounced back to
   `/saml/acs` with a SAMLResponse.
3. Wave validates it via the plugin and drops you on `/dashboard`.

## What to look at

- `plugins.saml_corp` — declares the SAML subprocess plugin.
- `auth.corp_sso.type: plugin` + `plugin: saml_corp` — binds the
  auth config to the plugin.
- `auth-login` route on `/auth/login` — calling auth-login against
  a plugin-backed config triggers `saml_init`.
- The ACS route is also `auth-login`; the plugin distinguishes
  init vs. callback from the request shape.

## Caveats

- Boots only when the plugin binary exists at the configured path
  AND every `SAML_*` env var is set.
- `SECRET_KEY` must be set.

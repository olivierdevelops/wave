# saml-enterprise-sso

Enterprise SSO end-to-end via the `saml-auth` reference plugin. Wave
owns the cookie/session; the plugin only translates SAML AuthnRequest /
SAMLResponse into a `Claims` struct.

## Setup

1. Build the plugin:

   ```sh
   cd examples/plugins/saml-auth && go build -o /tmp/wave-saml .
   ```

2. Generate (or obtain) an SP cert + key — your IdP will trust this:

   ```sh
   openssl req -x509 -newkey rsa:2048 -nodes -days 365 \
     -keyout /tmp/wave-sp.key -out /tmp/wave-sp.crt \
     -subj "/CN=wave-demo-sp"
   ```

3. Export env vars (point at any SAML IdP — Auth0, Okta, Keycloak,
   samltest.id):

   ```sh
   export SAML_IDP_METADATA_URL="https://samltest.id/saml/idp"
   export SAML_SP_CERT_PATH=/tmp/wave-sp.crt
   export SAML_SP_KEY_PATH=/tmp/wave-sp.key
   ```

   Register `http://127.0.0.1:8703/login/saml/callback` as the ACS URL
   and `http://127.0.0.1:8703/saml` as the SP entity ID with the IdP.

## Run it

```sh
wave serve examples/apps/saml-enterprise-sso/server.yaml --port 8703
```

## Try it

Open `http://127.0.0.1:8703/login` in a browser, click the SSO link,
authenticate at the IdP, land on `/dashboard` with a `corp_session`
cookie set by Wave.

## What to look at

`/login/saml` and `/login/saml/callback` are both `type: auth-login`
routes that differ only in the `X-Auth-Method` header — `saml_init`
vs `saml_callback`. The plugin dispatches on that. Roles in the
returned `Claims` flow through `infra/rbac` middleware.

## Caveats

Without a reachable IdP metadata URL the plugin process exits at
startup. Real production deployments should serve over HTTPS so the
SP-initiated redirect URLs match the cert.

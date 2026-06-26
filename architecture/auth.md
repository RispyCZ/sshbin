# Auth

Authentication is required to configure or delete shares and to access private shares.

## Session auth (OTP)

Login flow:
1. User enters email → `POST /login` calls `auth.Manager.StartLogin()`, which generates a 6-digit code and sends it via the configured `Sender`.
2. User enters code → `POST /verify` calls `auth.Manager.Verify()`. On success, a 32-byte random session token is stored in the DB and set as an `HttpOnly` cookie (`fd_session`).
3. Subsequent requests read the cookie and look up the session. Expired sessions are rejected.

**Sender implementations:**

| Sender | Use |
|--------|-----|
| `auth.LogSender` | Logs OTP to stdout — dev / self-hosted default |
| _(custom)_ | Implement `auth.Sender` interface to wire SMTP, SMS, etc. |

**Planned:** OAuth2 (Keycloak), enterprise SAML.

## Password grants (stateless)

Share-level password access uses a stateless cookie — no DB row per grant.

When a viewer submits the correct password for a share, the server sets a cookie `fd_pw_{shareID}` whose value is `HMAC-SHA256(shareID, secret)`. Subsequent requests to that share verify the HMAC locally without a DB lookup. The secret is generated once on first run and persisted in `app_meta`.

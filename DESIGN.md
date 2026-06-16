# SSO Auth Demo — Design

A learning project to understand how single sign-on works, built in Go. A protected
page is accessible only to authenticated users, who can authenticate either with
email + password or via "Sign in with Google". Email is the single identity.

## Decisions

| # | Topic | Decision |
|---|-------|----------|
| 1 | App shape | Server-rendered Go app (no SPA). |
| 2 | Web framework | Standard library `net/http` (1.22+ routing), no framework. |
| 3 | Sessions | Server-side sessions; opaque random ID in an `HttpOnly` cookie. |
| 4 | Database | SQLite via pure-Go `modernc.org/sqlite`; raw `database/sql`, handwritten SQL, no ORM. |
| 5 | OAuth | `golang.org/x/oauth2` + `oauth2/google`. We write the redirect, `state` check, code→token exchange, and userinfo fetch ourselves. Identity via userinfo endpoint now; isolated `fetchGoogleIdentity` so we can switch to OIDC `id_token` later. |
| 6 | Identity model | Email is identity. Auto-link on **verified** email. `password_hash` and `google_sub` both nullable. |
| 7 | Passwords | bcrypt (`golang.org/x/crypto/bcrypt`) at default cost. Email is the login identifier (no separate username). |
| 8 | Config/secrets | Env vars via gitignored `.env` (`joho/godotenv` in dev), validated at startup, fail-fast. |
| 9 | Structure | Lightly layered (see below). |
| 10 | UI | Minimal `html/template`, shared layout, parsed once at startup. Pages: `/`, `/login`, `/signup`, `/protected`. |
| 11 | OAuth `state` | Random (`crypto/rand`) value in a short-lived `oauth_state` cookie; verified and cleared in callback. |
| 12 | Form CSRF | `SameSite=Lax` session cookie only for now; note that prod should add per-form CSRF tokens. |
| 13 | Session lifetime | Fixed 24h absolute expiry. `sessions(id, user_id, created_at, expires_at)`. Expired rows ignored/lazily deleted on read. |
| 14 | Dev transport | Plain HTTP on `localhost:8080`. `Secure` cookie flag driven by config (`COOKIE_SECURE`, default false). |
| 15 | Validation | Email via `net/mail.ParseAddress`; password min 8 chars, no composition rules. Duplicate email with password → "account exists, please log in"; Google-only collision → "please sign in with Google". Both Google buttons share one `/auth/google` endpoint. |
| 16 | Errors | Re-render template with `Error` field; generic "invalid email or password" to prevent enumeration; preserve typed email. |
| 17 | Post-login redirect | Validated relative `next` (must start with `/`, reject external — open-redirect guard); default `/protected`; carried through `state` cookie for the Google flow. |
| 18 | Scopes | `openid email profile`. Store stable `google_sub`. |
| 19 | Logout | `POST /logout` → delete DB session, clear cookie, redirect `/`. |
| 20 | Tests | Targeted unit tests: `next` validation, password hash/verify, lookup-or-create-or-link store fn (temp SQLite DB). |
| 21 | Docs/setup | `README.md` with Google Cloud Console steps + `.env.example`. App auto-creates tables from `schema.sql` on startup. |

## Package layout

```
cmd/server/main.go        — wiring: config, DB, routes, start server
internal/config           — env loading + validation
internal/store            — DB access (users, sessions), raw SQL
internal/auth             — password hashing, Google OAuth, session create/verify
internal/web              — http handlers + middleware + templates
  internal/web/templates  — .html files
schema.sql                — table definitions
.env.example              — config template
```

## Schema

```sql
CREATE TABLE users (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  email         TEXT NOT NULL UNIQUE,
  password_hash TEXT,            -- nullable: null for Google-only accounts
  google_sub    TEXT,            -- nullable: null for password-only accounts
  name          TEXT,
  created_at    TIMESTAMP NOT NULL
);

CREATE TABLE sessions (
  id         TEXT PRIMARY KEY,   -- opaque random id stored in cookie
  user_id    INTEGER NOT NULL REFERENCES users(id),
  created_at TIMESTAMP NOT NULL,
  expires_at TIMESTAMP NOT NULL
);
```

## Routes

| Method | Path | Purpose |
|--------|------|---------|
| GET  | `/` | Home; shows login/signup or welcome+logout based on session. |
| GET  | `/login` | Email+password form + "Sign in with Google" button. |
| POST | `/login` | Verify credentials, create session, redirect to validated `next`. |
| GET  | `/signup` | Email+password form + "Sign up with Google" button. |
| POST | `/signup` | Validate, create user, create session, redirect. |
| GET  | `/auth/google` | Build auth URL, set `oauth_state` cookie (+ `next`), redirect to Google. |
| GET  | `/auth/google/callback` | Verify `state`, exchange code, fetch identity, create-or-link user, create session, redirect to `next`. |
| POST | `/logout` | Delete session, clear cookie, redirect `/`. |
| GET  | `/protected` | Auth-required (middleware); redirect to `/login?next=/protected` if no valid session. |

## Future enhancements (out of scope now)
- Switch identity resolution from userinfo endpoint to OIDC `id_token` verification (+ PKCE/nonce).
- Per-form CSRF tokens.
- Flash messages (Post/Redirect/Get) instead of re-render.
- HTTPS locally; `Secure=true`.
- Account linking UX (let Google-only users set a password, and vice versa).

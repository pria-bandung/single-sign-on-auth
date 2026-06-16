# PRD — SSO Auth Demo

## Problem Statement

I want to understand, hands-on, how single sign-on actually works. Reading about
OAuth/OIDC in the abstract hasn't given me a concrete mental model of the moving
parts — the redirect to the provider, the `state`/CSRF check, the `code`→token
exchange, fetching the user's identity, and then turning that into a durable
logged-in session. I also want to see how a "sign in with Google" path coexists
with a traditional email + password path for the *same* user, since that account-
linking problem is where SSO gets genuinely subtle.

I need a small, real, end-to-end app I can run on my laptop, click through, and
inspect (cookies, DB rows, redirects) — where nothing important about the auth
flow is hidden behind a library.

## Solution

A minimal server-rendered Go web app with a single protected page. Visitors can:

- Sign up with email + password, or sign up with their Google account.
- Log in with email + password, or "Sign in with Google".
- Reach a protected page only while authenticated; otherwise they're bounced to
  login and returned to the page they wanted after logging in.
- Log out.

Email is the single identity: a user who signed up with a password and later signs
in with Google (same verified email) is recognized as the *same* account, with the
Google identity auto-linked. The OAuth flow is written explicitly using
`golang.org/x/oauth2` so every step is visible and inspectable. Sessions are
server-side (a row in SQLite) keyed by an opaque cookie, so login state is easy to
reason about and logout is a real revocation.

The whole thing runs locally over HTTP on `localhost:8080` with SQLite, so there's
no infrastructure to stand up beyond creating Google OAuth credentials.

## User Stories

1. As a visitor, I want a home page that reflects whether I'm logged in, so that I always know my current auth state.
2. As a logged-out visitor, I want clear "Log in" and "Sign up" entry points, so that I can choose how to get access.
3. As a new user, I want to sign up with my email and a password, so that I can create an account without a third party.
4. As a new user, I want my password to be checked for a minimum length (8 characters), so that I'm nudged toward a non-trivial password.
5. As a new user, I want my email validated for basic format, so that I don't accidentally register a malformed address.
6. As a new user, I want to sign up using my Google account, so that I don't have to create or remember another password.
7. As a returning user, I want to log in with my email and password, so that I can access the protected page.
8. As a returning user, I want to "Sign in with Google", so that I can authenticate without typing a password.
9. As a user who signed up with a password and later clicks "Sign in with Google" using the same verified email, I want the system to recognize me as the same account and link my Google identity, so that I have one account, not two.
10. As a user who signed up with Google and later tries to register that email with a password, I want to be told to sign in with Google instead, so that I'm not confused by a duplicate account.
11. As a user who already has a password account and tries to sign up again with that email, I want to be told the account already exists and to log in, so that I don't create a duplicate.
12. As a user entering wrong credentials, I want a single generic "invalid email or password" message, so that the app doesn't leak whether an email is registered.
13. As a user who mistyped one field, I want my email preserved in the form after an error, so that I don't have to retype everything.
14. As an authenticated user, I want to view the protected page, so that I can confirm the gate works.
15. As an unauthenticated user hitting the protected page, I want to be redirected to log in, so that the page stays private.
16. As a user redirected to log in from a specific page, I want to be sent back to that page after authenticating, so that I land where I intended.
17. As a user, I want the "return-to" redirect to reject external URLs, so that the login flow can't be abused as an open redirect.
18. As an authenticated user, I want a logout button, so that I can end my session.
19. As a user who logged out, I want my session truly invalidated server-side, so that the old cookie can't be reused.
20. As a user, I want my session to last a fixed period (24h) and then require re-login, so that stale sessions don't live forever.
21. As a security-conscious user, I want the Google login protected by a `state` parameter, so that login-CSRF attacks are prevented.
22. As a security-conscious user, I want account linking to happen only when Google reports my email as verified, so that an unverified email can't hijack an account.
23. As an operator, I want the app to fail fast at startup if required configuration is missing, so that I find misconfiguration immediately.
24. As an operator, I want OAuth secrets read from environment variables / a gitignored `.env`, so that secrets never land in source control.
25. As an operator, I want the database tables created automatically on first run, so that there's no separate migration step.
26. As a developer new to this repo, I want a README with Google Cloud Console setup steps and a `.env.example`, so that I can get the OAuth flow running quickly.
27. As a learner, I want the OAuth steps (auth URL, state check, code exchange, identity fetch) written explicitly rather than hidden, so that I can read and understand each step.
28. As a learner, I want the identity-fetch code isolated, so that I can later swap the userinfo endpoint for OIDC `id_token` verification without rewriting the flow.

## Implementation Decisions

**Application shape & stack**
- Server-rendered Go app; no SPA. Standard library `net/http` with 1.22+ method/pattern routing; no web framework.
- Minimal `html/template` views with a shared layout, parsed once at startup. Pages: `/`, `/login`, `/signup`, `/protected`. No CSS framework.
- Runs over plain HTTP on `localhost:8080` in dev. The session cookie `Secure` flag is config-driven (`COOKIE_SECURE`, default `false`).

**Modules (lightly layered)**
- `config` — loads env vars (via `joho/godotenv` in dev) and validates required values at startup, failing fast. Expected vars: `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URL`, `PORT`, `DATABASE_PATH`, `COOKIE_SECURE`, session settings.
- `store` — all persistence behind a small interface over raw `database/sql` + handwritten SQL on SQLite (`modernc.org/sqlite`, pure Go). Responsibilities: create/find users by email, set `google_sub` on link, create/find-valid/delete sessions. Lazily ignores or deletes expired sessions on read.
- `auth` — three concerns: (a) password hashing with bcrypt (`golang.org/x/crypto/bcrypt`, default cost) — `HashPassword`/`VerifyPassword`; (b) session lifecycle (create with 24h absolute expiry, validate, destroy); (c) Google OAuth via `golang.org/x/oauth2` + `oauth2/google` — build auth URL, verify `state`, exchange code, and an isolated `fetchGoogleIdentity` returning `(email, emailVerified, sub, name)` from the userinfo endpoint. The lookup-or-create-or-link decision lives here over the `store` interface.
- `web` — HTTP handlers, the auth middleware that gates `/protected`, the pure `validateNext` open-redirect guard, and template rendering.
- `cmd/server` — wiring: load config, open DB, run schema, build dependencies, register routes, start server.

**Identity & account model**
- Email is the identity and the sole login identifier (no separate username).
- `users(id, email UNIQUE, password_hash NULL, google_sub NULL, name, created_at)`. Password-only users have `google_sub = NULL`; Google-only users have `password_hash = NULL`; linked users have both.
- On Google callback: require `email_verified`, then look up by email → if found, set `google_sub` if absent and log in; if not found, create the user. Store the stable `google_sub`.

**OAuth flow**
- Scopes: `openid email profile`. Both Google buttons (`/signup`, `/login`) share one `GET /auth/google` endpoint.
- `state`: cryptographically random (`crypto/rand`) value set in a short-lived `oauth_state` cookie before redirect; verified and cleared in the callback. The validated `next` target is carried through this cookie for the Google flow.
- Identity resolution uses the userinfo endpoint now; `fetchGoogleIdentity` is isolated so OIDC `id_token` verification (+ PKCE/nonce) can replace it later without touching the rest of the flow.

**Sessions**
- Server-side: `sessions(id, user_id, created_at, expires_at)`; `id` is an opaque random value stored in an `HttpOnly`, `SameSite=Lax`, `Path=/` cookie, `Secure` per config, `Max-Age` ~24h. Fixed 24h absolute expiry (no sliding).
- `POST /logout` deletes the session row, clears the cookie, redirects to `/`.

**Validation, errors, redirects**
- Email validated via `net/mail.ParseAddress`; password minimum 8 chars, no composition rules.
- Login failures re-render the template with a generic "invalid email or password" `Error` field (no enumeration) and preserve the typed email.
- Duplicate-email signup: password account exists → "account exists, please log in"; Google-only account exists → "please sign in with Google".
- `next` redirect must be a relative path starting with `/` (reject absolute/external); default `/protected`.

**CSRF posture**
- OAuth flow protected by the `state` parameter. Form POSTs rely on `SameSite=Lax` for now; per-form CSRF tokens are noted as a production follow-up, not implemented.

**Setup**
- Schema in `schema.sql`, executed on startup if tables are absent (auto-create). `README.md` documents Google Cloud Console OAuth client setup and a `.env.example` template. `.env` is gitignored.

## Testing Decisions

**What makes a good test here:** tests should exercise *external behavior* of a
module through its public interface, not its internals. They should not assert on
SQL strings, private helper calls, or template markup — only on observable
outcomes (return values, persisted state, redirect targets). Each tested unit is
chosen because it is pure or has a narrow, deterministic interface and encodes
security-sensitive logic that is easy to get wrong silently.

**Modules/units that will be tested:**
- `web.validateNext` — pure function. Table-driven test covering: valid relative paths pass; absolute URLs, scheme-relative `//host`, and missing/empty values are rejected and fall back to `/protected`.
- `auth` password hashing — round-trip: a hashed password verifies against the original and fails against a wrong one; two hashes of the same password differ (salting).
- `store` lookup-or-create-or-link — against a temp SQLite DB (fresh file or `:memory:`): creating a user, finding by email, creating a duplicate email is rejected by the UNIQUE constraint, and linking a `google_sub` to an existing email-matched user produces one account with both credentials populated.

**Prior art:** none yet — this is a greenfield repo, so these establish the
pattern. Tests use Go's standard `testing` package, table-driven where natural,
with the temp-DB tests creating and tearing down a throwaway SQLite database per
test.

**Out of test scope:** full HTTP handler tests and end-to-end OAuth integration
(would require mocking/stubbing Google) — covered instead by manual click-through.

## Out of Scope

- OIDC `id_token` verification, PKCE, and nonce (deferred; code is structured to allow the swap).
- Per-form CSRF tokens (relying on `SameSite=Lax` for now).
- Flash messages / Post-Redirect-Get (using template re-render instead).
- Local HTTPS / `Secure=true` by default (plain HTTP on localhost in dev).
- Richer account-linking UX (e.g. letting a Google-only user set a password, or a password user explicitly manage linked providers).
- Password reset, email verification for the local signup path, "remember me" / long-lived sessions, sliding-expiry sessions.
- Roles/authorization beyond "authenticated vs not", rate limiting, brute-force lockout, and any styling beyond minimal HTML.
- A SPA/JSON API, multiple OAuth providers, and any production deployment concerns (background session cleanup jobs, observability, etc.).

## Further Notes

- This is explicitly a learning project; choices favor *visibility of the auth
  mechanics* over production hardening. Several deliberate "lessons" are baked in:
  the `state`/CSRF check, generic login errors to prevent enumeration, the
  `email_verified` gate before linking, and the open-redirect guard on `next`.
- Prerequisite before anything runs: Google OAuth credentials must be created in
  Google Cloud Console (Web application client) with
  `http://localhost:8080/auth/google/callback` as an authorized redirect URI.
- The canonical design reference is `DESIGN.md` in the repo root; this PRD is
  derived from it and the grilling session that produced it.

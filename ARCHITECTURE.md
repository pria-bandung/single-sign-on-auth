# Architecture & Lessons Learned

A walkthrough of how this single sign-on demo is built, how the pieces fit
together, and what the project set out to teach. For *why* each decision was made
see [`DESIGN.md`](DESIGN.md); for setup/run see [`README.md`](README.md).

---

## 1. What the app does

A server-rendered Go web app with one protected page. A visitor can authenticate
two ways, and **email is the single identity** either way:

- **Email + password** — local accounts, bcrypt-hashed.
- **Sign in with Google** — OAuth 2.0 / OpenID Connect.

The same person is **one account** regardless of how they sign in: if a
password account and a Google sign-in share a verified email, the Google identity
is *linked* onto the existing account rather than creating a duplicate.

---

## 2. The big idea: authentication vs. session

The single most important mental model in this project:

> **Authentication** is *how you prove who you are* (a password check, or the
> Google OAuth dance). **A session** is *how the app remembers you're logged in*
> on later requests.

These are deliberately separate. Both the password path and the Google path end
by calling the **same** `startSession` helper. After that point the app doesn't
know or care how you logged in — there is one session mechanism. Conflating these
two concerns (e.g. thinking "OAuth = being logged in") is the classic source of
confusion this project is designed to dispel.

---

## 3. Package layout (lightly layered)

```
cmd/server/main.go        Composition root: load config, open DB, build server, listen
internal/config           Env loading + fail-fast validation
internal/store            All persistence: users, sessions, schema (raw SQL, SQLite)
internal/auth             Credential primitives: bcrypt, session IDs, Google OAuth client
internal/web              The only package that speaks HTTP: handlers, middleware, templates
  internal/web/templates  html/template files (layout + pages)
```

Dependencies point **downward** only:

```
cmd/server ──> web ──> auth ──> store
                 └────────────> store
config is read by cmd/server and handed to the layers that need it
```

`web` depends on `auth` and `store`; `auth` depends on `store` only where the
account-resolution logic needs it. Nothing lower ever imports `web`. This keeps
each layer testable in isolation.

---

## 4. The modules, one at a time

### `config` — fail fast
`Load()` reads environment variables (loading `.env` on a best-effort basis for
dev), applies defaults (`PORT=8080`, `DATABASE_PATH=app.db`, `COOKIE_SECURE=false`),
and returns an error naming **every** missing required variable. The app refuses
to start misconfigured rather than failing mysteriously at request time.

### `store` — a deep module over SQLite
Owns the database and hides all SQL behind a small, typed API:

- `Open(dsn)` opens SQLite (pure-Go `modernc.org/sqlite`) and applies the embedded
  `schema.sql` (idempotent `CREATE TABLE IF NOT EXISTS`), so the DB self-creates
  on first run.
- Users: `CreateUser`, `FindUserByEmail`, and the linking primitive
  `UpsertGoogleUser`.
- Sessions: `CreateSession`, `FindUserBySession`, `DeleteSession`.
- Sentinel errors `ErrNotFound` / `ErrEmailTaken` let callers branch with
  `errors.Is` instead of parsing driver strings.

Two columns are nullable and surface as `*string` in Go: `password_hash`
(nil for Google-only users) and `google_sub` (nil for password-only users).

### `auth` — credential primitives
Pure-ish helpers shared by every login path:

- `HashPassword` / `VerifyPassword` — bcrypt (salt is embedded in the hash).
- `NewSessionID` — a 256-bit `crypto/rand` opaque token.
- `GoogleOAuth` — wraps `golang.org/x/oauth2`: `AuthCodeURL`, `Exchange`, and
  `FetchIdentity` (calls the OIDC userinfo endpoint). The JSON decoding is split
  into a pure `parseGoogleIdentity` so the fragile part is unit-testable.

### `web` — the HTTP layer
The only package that imports `net/http`. Holds parsed templates, the store, the
Google client, and cookie/session settings. Routes (stdlib `net/http` 1.22+
pattern routing):

| Method | Path | Purpose |
|--------|------|---------|
| GET  | `/` | Home; logged-out or welcome+logout |
| GET/POST | `/login` | Email+password login |
| GET/POST | `/signup` | Email+password signup |
| GET  | `/protected` | Gated by `requireUser` middleware |
| POST | `/logout` | Revoke session |
| GET  | `/auth/google` | Start Google flow |
| GET  | `/auth/google/callback` | Finish Google flow |

---

## 5. How a request flows

### Email + password signup
1. `POST /signup` → validate email (`net/mail`) and password length (≥8).
2. Check for an existing account by email; reject duplicates with a message that
   distinguishes a password account ("please log in") from a Google-only one
   ("sign in with Google").
3. `auth.HashPassword` → `store.CreateUser`.
4. `startSession` creates a `sessions` row and sets the cookie → redirect to
   `/protected`.

### Email + password login
1. `POST /login` → `store.FindUserByEmail`.
2. Verify with `auth.VerifyPassword`. **Any** failure (no such user, user has no
   password, wrong password) returns the **same** generic "invalid email or
   password" — one branch-free code path, so the response can't be used to
   enumerate which emails are registered.
3. `startSession` → redirect to a **validated** `next` (default `/protected`).

### Sign in with Google (the OAuth dance, made visible)
1. `GET /auth/google` — generate a random `state`, stash it (and the validated
   `next`) in short-lived cookies, redirect to Google's consent screen with scopes
   `openid email profile`.
2. User consents at Google; Google redirects back to
   `GET /auth/google/callback?code=…&state=…`.
3. **Verify `state`** against the cookie (CSRF defense), then clear the cookies.
4. `Exchange` the `code` for a token; `FetchIdentity` calls the userinfo endpoint.
5. **Require `email_verified`** before trusting the email.
6. `store.UpsertGoogleUser` — lookup-or-create-or-link by email.
7. `startSession` → redirect to `next`.

### Every authenticated request
`requireUser` middleware reads the session cookie and calls
`store.FindUserBySession(id, now)` — a single JOIN that returns the owning user
**only if** the session exists and hasn't expired. No valid session → redirect to
`/login?next=<path>`.

---

## 6. Account linking — the subtle part

```
Google callback with a verified email
        │
        ▼
  FindUserByEmail(email)
        │
   ┌────┴─────────────┐
   │                  │
 found             not found
   │                  │
google_sub null?   CreateUser(google_sub set, no password)
   │
   ├─ yes → UPDATE google_sub  (LINK onto the existing account)
   └─ no  → already linked, just log in
```

This is `store.UpsertGoogleUser`. The result: **one human = one account**, keyed
on email, no matter the mix of login methods. Linking is only trusted when Google
reports the email as verified — otherwise someone could claim an account by
signing up to Google with an unverified address.

---

## 7. Security decisions baked in

| Concern | What the app does |
|---------|-------------------|
| Password storage | bcrypt with per-hash salt |
| Session token | 256-bit `crypto/rand`, opaque; truth lives server-side |
| Session cookie | `HttpOnly`, `SameSite=Lax`, `Path=/`, `Secure` via config |
| OAuth CSRF | random `state` echoed by Google and checked against a cookie |
| Account enumeration | one generic login error for all failure modes |
| Account hijack via email | link only when `email_verified` is true |
| Open redirect | `validateNext` only allows local relative paths |
| Logout | real server-side revocation (`DeleteSession`), not just cookie clearing |
| Secrets | env vars / gitignored `.env`; never committed |

---

## 8. Testing approach (TDD, behavior over implementation)

Each slice was built red → green: write one failing test for a behavior, then the
minimal code to pass it. Tests exercise **public interfaces** and use **real
collaborators** (a temp-file SQLite store) rather than mocks of internal parts, so
they survive refactors:

- `store` — user round-trip, duplicate rejection, session lifecycle/expiry, the
  create-or-link logic, and a storage-format property (timestamps are UTC).
- `auth` — bcrypt round-trip, session-id uniqueness, identity JSON parsing.
- `web` — full signup/login/protected/logout flows via `httptest`, the generic
  login error, the `next` open-redirect guard, and the Google redirect + state
  cookie.

The **only** manually-verified path is the live Google round-trip (token exchange
against real Google can't run headlessly) — everything around it (state cookie,
identity parsing, account linking) is unit-tested.

---

## 9. Lessons learned

1. **Authentication and session are different things.** Building both login paths
   to converge on one `startSession` made this concrete: the session layer is
   completely agnostic to *how* you authenticated.

2. **OAuth is just a redirect dance you can read.** Using low-level
   `golang.org/x/oauth2` instead of a turnkey library kept every step visible:
   build URL → user consents → `code` comes back → exchange for token → fetch
   identity. The `state` parameter is simply a nonce you hand the browser and
   re-check on return.

3. **"One identity" forces real design decisions.** The moment you allow two login
   methods, you must answer "what is a user?" Email-as-identity + verified-email
   linking is what turns two parallel login systems into genuine SSO.

4. **Push correctness to a single boundary.** The timestamp bug (Go's monotonic
   clock leaking into stored times, and local vs. UTC offsets) was fixed once, in
   the `store` layer, by normalizing every timestamp to UTC RFC3339 text. Callers
   keep passing ordinary `time.Now()`. Lesson: text comparisons in the DB are only
   correct if the stored format is lexically sortable — UTC RFC3339 is; Go's
   default `time.Time` rendering is not.

5. **Deep modules with small interfaces pay off in tests.** `FindUserBySession`
   does "validate expiry + load user" in one method, so the middleware has nothing
   to orchestrate and the behavior is testable in one call. Replacing the earlier
   two-call `FindValidSession` design removed dead code and simplified the caller.

6. **Test behavior, not structure, with real dependencies.** Running tests against
   a real temp SQLite database (not a mock) caught actual SQL and format issues a
   mock would have hidden, and the tests didn't break when internals were
   refactored.

7. **Fail fast on config.** Validating required env vars at startup turned a
   confusing mid-request failure into an obvious one-line message.

8. **Generic errors are a feature.** Collapsing every login failure into one
   message and one code path is both simpler code and a real defense against
   account enumeration.

---

## 10. Deliberate deviations from the original design

- **`schema.sql` lives in `internal/store/`** (not the repo root) so `//go:embed`
  bundles it into a single binary.
- **`UpsertGoogleUser` lives in `store`** (not `auth`) — it's storage logic, and
  keeping it there let it be unit-tested directly against a temp DB. The
  `email_verified` *policy* check stays in the `web` callback, before the store
  call.

---

## 11. Known limitations / future work

- Identity comes from the userinfo endpoint; switching to verifying the OIDC
  `id_token` directly (+ PKCE/nonce) is the natural next step (scopes already
  include `openid`).
- Form CSRF currently relies on `SameSite=Lax`; a production app would add
  per-form CSRF tokens.
- No password reset, email verification for local signup, rate limiting, or
  "remember me"; HTTP-only in dev (set `COOKIE_SECURE=true` + HTTPS for real use).
- `UpsertGoogleUser` is not wrapped in a transaction, so a concurrent
  same-email race is theoretically possible (fine for a learning project).

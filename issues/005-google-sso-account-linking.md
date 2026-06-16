## Parent PRD

`issues/prd.md`

## What to build

The full "Sign in with Google" path, end-to-end, including account linking with the
existing email/password accounts. Both Google buttons (on `/login` and `/signup`)
hit a single `GET /auth/google` endpoint, which builds the Google auth URL
(`openid email profile` scopes), generates a random `state`, stores it in a
short-lived `oauth_state` cookie carrying the validated `next`, and redirects to
Google. The callback verifies `state` against the cookie, exchanges the `code` for
a token, and `fetchGoogleIdentity` calls the userinfo endpoint to get
`(email, email_verified, sub, name)`. The flow proceeds only if the email is
verified. It then resolves the account by email: if a user with that email exists,
set `google_sub` if absent and log them in (linking); otherwise create a new user
with the `google_sub` and no password. On success it creates a session (same
backbone as the local path) and redirects to `next`. A user who signed up with
Google and later tries to register that email with a password is told to sign in
with Google instead. `fetchGoogleIdentity` is kept isolated so OIDC `id_token`
verification can replace it later.

See PRD "Implementation Decisions" ("Identity & account model", "OAuth flow") and
"Testing Decisions".

## Acceptance criteria

- [ ] `GET /auth/google` (shared by both buttons) sets a random `crypto/rand` `state` in a short-lived `oauth_state` cookie carrying the validated `next`, and redirects to Google with `openid email profile` scopes.
- [ ] The callback rejects the request if the returned `state` does not match the cookie, and clears the cookie after use.
- [ ] The callback exchanges the code and fetches identity via the userinfo endpoint through an isolated `fetchGoogleIdentity`.
- [ ] The flow proceeds only when Google reports the email as verified.
- [ ] If no user has the email, a new user is created with `google_sub` set and `password_hash` NULL, then logged in.
- [ ] If a user with the email already exists, `google_sub` is set if missing and the user is logged in (one account, both credentials populated where applicable).
- [ ] On success a session is created and the user is redirected to the validated `next` (default `/protected`).
- [ ] Attempting password signup for an email that exists as a Google-only account shows "please sign in with Google".
- [ ] Unit tests cover the lookup-or-create-or-link store logic against a temp SQLite DB (create, find by email, duplicate-email rejected by UNIQUE constraint, linking a `google_sub` to an email-matched user yields one account with both credentials).

## Blocked by

- Blocked by `issues/002-email-password-signup-session-protected.md`
- Blocked by `issues/004-google-oauth-setup-docs.md` (for end-to-end verification)

## User stories addressed

- User story 6
- User story 8
- User story 9
- User story 10
- User story 21
- User story 22
- User story 27
- User story 28

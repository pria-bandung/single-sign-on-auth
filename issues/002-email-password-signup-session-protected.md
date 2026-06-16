## Parent PRD

`issues/prd.md`

## What to build

The simplest complete authentication path, end-to-end, which also establishes the
session backbone and the protected-page gate that later slices reuse. A visitor
submits the signup form (email + password); the app validates the email format and
an 8-character minimum password, hashes the password with bcrypt, and creates the
user. On success it creates a server-side session row, sets the opaque session
cookie (`HttpOnly`, `SameSite=Lax`, `Secure` per config, ~24h), and the user is
authenticated. The auth middleware gates `GET /protected` — authenticated users see
it; the home page now shows a welcome + logout control when authenticated.
`POST /logout` deletes the session row, clears the cookie, and redirects home.
Signing up with an email that already has a password account shows
"account exists, please log in."

See PRD "Implementation Decisions" (modules `store`, `auth`, `web`), "Sessions",
and "Validation, errors, redirects".

## Acceptance criteria

- [ ] `GET /signup` renders the signup form (email + password).
- [ ] `POST /signup` rejects malformed emails and passwords shorter than 8 chars, re-rendering with an error and preserving the typed email.
- [ ] Valid signup creates a `users` row with a bcrypt `password_hash` and `google_sub` NULL.
- [ ] Successful signup creates a `sessions` row (24h absolute expiry) and sets the session cookie with `HttpOnly`, `SameSite=Lax`, `Path=/`, and `Secure` driven by config.
- [ ] Auth middleware allows authenticated users to view `GET /protected`.
- [ ] Home page shows welcome + logout when authenticated.
- [ ] `POST /logout` deletes the session row, clears the cookie, and redirects to `/`; the old cookie no longer grants access.
- [ ] Signing up with an email that already has a password account shows "account exists, please log in" rather than creating a duplicate.
- [ ] Expired sessions are treated as invalid on read.
- [ ] Unit tests cover the password hash/verify round-trip (verifies the original, fails a wrong password, two hashes of the same password differ).

## Blocked by

- Blocked by `issues/001-walking-skeleton.md`

## User stories addressed

- User story 1 (authenticated home)
- User story 3
- User story 4
- User story 5
- User story 11
- User story 14
- User story 15
- User story 18
- User story 19
- User story 20

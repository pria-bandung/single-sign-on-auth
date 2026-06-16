## Parent PRD

`issues/prd.md`

## What to build

The email/password login path plus the "return-to the page you wanted" redirect
behavior. `POST /login` verifies the submitted email + password against the stored
bcrypt hash; on success it creates a session and redirects to a validated `next`
target (default `/protected`). Failures re-render the login form with a single
generic "invalid email or password" message (no account enumeration) and preserve
the typed email. The auth middleware, when blocking an unauthenticated request to
`/protected`, redirects to `/login?next=/protected`. The `next` value is validated
by a pure guard that only accepts local relative paths (must start with `/`,
reject absolute/external/scheme-relative URLs) to prevent open redirects. The home
page wires up the login and signup entry points.

See PRD "Implementation Decisions" ("Validation, errors, redirects", module `web`).

## Acceptance criteria

- [ ] `GET /login` renders the login form.
- [ ] `POST /login` with correct credentials creates a session and redirects to the validated `next`, defaulting to `/protected`.
- [ ] `POST /login` with wrong password OR unknown email shows the same generic "invalid email or password" message.
- [ ] The typed email is preserved on a failed login.
- [ ] Unauthenticated access to `/protected` redirects to `/login?next=/protected`, and after login the user lands on `/protected`.
- [ ] `next` values that are absolute URLs, scheme-relative (`//host`), or empty are rejected and fall back to `/protected`.
- [ ] Home page shows working "Log in" and "Sign up" entry points when logged out.
- [ ] Unit tests cover `validateNext` (table-driven: valid relative paths pass; absolute, scheme-relative, and empty are rejected/fall back).

## Blocked by

- Blocked by `issues/002-email-password-signup-session-protected.md`

## User stories addressed

- User story 2
- User story 7
- User story 12
- User story 13
- User story 16
- User story 17

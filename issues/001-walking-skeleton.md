## Parent PRD

`issues/prd.md`

## What to build

The end-to-end wiring spine for the app. On startup the server loads configuration
from the environment (failing fast if anything required is missing), opens the
SQLite database and auto-creates the schema if the tables are absent, then serves
HTTP on `localhost:8080`. Visiting `GET /` returns the home page rendered through
the shared layout template in its logged-out state (showing entry points to log in
/ sign up). No authentication logic yet — this slice proves config → store →
web → `cmd/server` are wired together and the app boots and renders.

See the PRD "Implementation Decisions" (modules `config`, `store`, `web`,
`cmd/server`) and "Setup" sections.

## Acceptance criteria

- [ ] App reads required config from env (with `.env` loaded in dev) and exits with a clear message if any required value is missing.
- [ ] On startup, the SQLite database is opened at the configured path and `schema.sql` is executed so `users` and `sessions` tables exist.
- [ ] Server listens on the configured port (default `:8080`).
- [ ] `GET /` renders the home page via the shared layout, showing the logged-out state.
- [ ] Templates are parsed once at startup.
- [ ] Running with no prior database file produces a working app on first run (tables auto-created).

## Blocked by

None - can start immediately.

## User stories addressed

- User story 1 (logged-out home)
- User story 23
- User story 24
- User story 25

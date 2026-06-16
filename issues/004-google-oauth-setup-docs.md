## Parent PRD

`issues/prd.md`

## What to build

The human-in-the-loop prerequisite for the Google sign-in path: documentation and
local credentials. Produce a `README.md` that walks through creating an OAuth
client in Google Cloud Console (create project → configure OAuth consent screen →
create OAuth Client ID of type "Web application" → add
`http://localhost:8080/auth/google/callback` as an authorized redirect URI → copy
the Client ID and Secret), how to run the app (`go run ./cmd/server`), and the DB
auto-create note. Add a `.env.example` template listing all required variables, and
populate a local (gitignored) `.env` with the real credentials so Slice 5 can be
verified end-to-end. This slice involves a manual step in Google Cloud Console that
cannot be automated.

See PRD "Setup" and "Further Notes".

## Acceptance criteria

- [ ] `README.md` documents the full Google Cloud Console OAuth client setup with the exact authorized redirect URI.
- [ ] `README.md` documents how to run the app and notes that tables are auto-created on first run.
- [ ] `.env.example` lists every required variable (`GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URL`, `PORT`, `DATABASE_PATH`, `COOKIE_SECURE`, session settings).
- [ ] `.env` is gitignored and is populated locally with working credentials.

## Blocked by

- Blocked by `issues/001-walking-skeleton.md`

## User stories addressed

- User story 26

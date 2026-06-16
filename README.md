# SSO Auth Demo

A small Go web app for learning how single sign-on works. It has one protected
page reachable only by authenticated users, who can sign in either with
**email + password** or with **"Sign in with Google"**. Email is the single
identity, so the same person is one account regardless of how they log in.

See [`DESIGN.md`](DESIGN.md) for the design decisions and [`issues/`](issues/)
for the implementation slices.

## Prerequisites

- Go 1.26+
- A Google account (to create OAuth credentials)

## 1. Create Google OAuth credentials

The Google sign-in flow needs an OAuth 2.0 client. Create one in Google Cloud
Console:

1. Go to <https://console.cloud.google.com/> and create (or select) a project.
2. Open **APIs & Services → OAuth consent screen**. Choose **External**, fill in
   the app name and your email, and add your own Google account under
   **Test users** (so you can sign in while the app is unpublished).
3. Open **APIs & Services → Credentials → Create Credentials → OAuth client ID**.
4. Choose **Application type: Web application**.
5. Under **Authorized redirect URIs**, add exactly:

   ```
   http://localhost:8080/auth/google/callback
   ```

6. Click **Create** and copy the **Client ID** and **Client secret**.

The requested scopes are `openid email profile` — Google will show a basic
sign-in consent screen, no sensitive permissions.

## 2. Configure the app

Copy the example env file and fill in your credentials:

```bash
cp .env.example .env
# then edit .env and paste your Client ID and Client secret
```

`.env` is gitignored — never commit it. The required variables are documented in
[`.env.example`](.env.example). Keep `PORT=8080` so the server URL matches the
redirect URI you registered above.

The app loads `.env` automatically on startup (and validates that the required
variables are present, failing fast with a clear message if any are missing).

## 3. Run

```bash
go run ./cmd/server
```

Then open <http://localhost:8080/>.

The SQLite database (default `app.db`) and its tables are created automatically
on first run — there is no separate migration step.

## What you can do

- **Sign up** with email + password at `/signup`.
- **Log in** at `/login`; an unauthenticated visit to `/protected` redirects you
  there and returns you to `/protected` after login.
- **Sign in with Google** (added in the Google SSO slice).
- **Log out** from the home or protected page.

## Tests

```bash
go test ./...
```

## Security notes (this is a learning project)

- Runs over plain HTTP on localhost; set `COOKIE_SECURE=true` and serve over
  HTTPS in any real deployment.
- Form CSRF protection currently relies on `SameSite=Lax` cookies; a production
  app should add per-form CSRF tokens.
- Never commit `.env` or the `client_secret_*.json` you downloaded from Google
  (both are gitignored). Rotate the secret if it is ever exposed.

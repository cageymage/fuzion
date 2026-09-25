---
name: next-backend-issue
description: Use when the user asks to find, pick up, or work on the next backend issue in the Fuzion repo. Finds an unblocked issue, assigns it, branches, implements with tests, verifies against a real running API, then asks before commit, push and PR.
---

# Next backend issue

Follow [../_shared/issue-workflow.md](../_shared/issue-workflow.md) end to end. This file supplies the backend-specific parts.

## Pick filter (step 2)

Issues labeled `backend`. Issues labeled both `backend` and `frontend` are fine: do only the backend half and note the frontend follow-up in the PR.

## Implementation notes (step 6)

- Go, chi, sqlx, Postgres. Structure and rules: CLAUDE.md and `plans/project-structure.md`.
- One test file per feature package, real `httptest.Server` over router, service, repo and a testcontainers Postgres. Only Discord, Battle.net and Blizzard clients are faked.
- Officer-only routes are wrapped with `auth.RequireOfficer`. Do not add auth logic in handlers.
- Validation lives in `service.go`. Handlers decode JSON and map errors to status codes.
- Tests log in with `srv.LoginAs(t, discordID, username, ...)`; check `internal/testutil/testserver.go` for the current options and add helpers there when the issue says to.

## Verify (step 7)

1. Automated: `go vet -C backend ./...` then `go test -C backend ./...`. Docker must be running (testcontainers).
2. Real API check, when the change has an HTTP surface:
   - Start Postgres: `docker compose up -d db`.
   - Run the API in the background from `backend/` with `DATABASE_URL=postgres://fuzion:fuzion@localhost:5432/fuzion?sslmode=disable`. Discord vars are required by config; dummy values are fine for anything that does not use the OAuth flow. Migrations apply on startup.
   - Seed data if useful: `seed/dev_seed.sql` via `docker compose exec -T db psql -U fuzion -d fuzion < backend/seed/dev_seed.sql`.
   - For authenticated routes there is no dev-login endpoint, so create a session directly:
     ```sql
     INSERT INTO users (id, discord_id, username, is_officer)
       VALUES (gen_random_uuid(), 'dev-officer', 'DevOfficer', true) ON CONFLICT (discord_id) DO UPDATE SET is_officer = true;
     INSERT INTO sessions (token, user_id, expires_at)
       SELECT 'dev-officer-token', id, now() + interval '1 hour' FROM users WHERE discord_id = 'dev-officer'
       ON CONFLICT (token) DO UPDATE SET expires_at = excluded.expires_at;
     ```
     For a member or anonymous check, use a second user with `is_officer = false`, or no cookie. Then call with `curl -i -H "Cookie: fuzion_session=dev-officer-token" ...`.
   - Exercise each acceptance criterion: happy path, validation error, 401 anonymous, 403 member, 404, 409 where the issue lists them. Compare status codes and bodies to the issue.
   - Stop the API process afterwards. Leave the dev database alone unless you created rows that would confuse later runs; say what you created.
3. If there is no HTTP surface (migration only, internal package), say the curl step was skipped and why.
4. If Docker or the API cannot start, report that as the verification result. Do not claim success on unit tests alone without saying the real-API check did not run.

Put a short summary of the curl evidence (method, path, status, key body fields) in the PR body.

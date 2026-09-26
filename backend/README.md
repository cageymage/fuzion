# Fuzion backend

Go API for the Fuzion guild site: chi router, sqlx over Postgres, migrations
embedded in the binary.

## Prerequisites

- Go 1.23+
- Docker (Postgres for local dev, and testcontainers for the test suite)

## Commands

```
go mod tidy                        # resolve dependencies, write go.sum
go run ./cmd/api                   # needs DATABASE_URL
go test ./...                      # sociable tests, real Postgres via testcontainers
go vet ./...
```

Environment:

| Variable          | Default                 | Notes                              |
| ----------------- | ----------------------- | ---------------------------------- |
| `DATABASE_URL`    | required                | `postgres://user:pass@host:5432/db?sslmode=disable` |
| `ADDR`            | `:8080`                 | listen address                     |
| `ALLOWED_ORIGINS` | `http://localhost:5173` | comma-separated CORS origins       |
| `DISCORD_CLIENT_ID`     | required | from the Discord developer portal (OAuth2 tab) |
| `DISCORD_CLIENT_SECRET` | required | same place; never commit it        |
| `DISCORD_REDIRECT_URL`  | required | must be registered on the Discord app; `http://localhost:5173/api/auth/callback` in dev |

Migrations under `migrations/` are embedded and applied on startup, so there
is no separate migrate step.

## Endpoints

| Method | Path                | Response                                            |
| ------ | ------------------- | --------------------------------------------------- |
| GET    | `/api/health`       | `{"status":"ok"}`                                   |
| GET    | `/api/news`         | news posts, newest first (`[]` when none)           |
| GET    | `/api/raids/next`   | soonest upcoming raid, or `null` when none scheduled |
| GET    | `/api/streams/live` | live streamers, most viewers first (`[]` when none) |
| GET    | `/api/applications`  | officer only: applications newest first, `?status=pending\|accepted\|declined` optional (400 on unknown) |
| GET    | `/api/applications/{id}` | officer only: one application in full; 400 for a non-UUID id, 404 when unknown |
| PATCH  | `/api/applications/{id}` | officer only: `{status: "accepted"\|"declined", reviewNote?}`; stamps `reviewedBy` and `reviewedAt`, re-reviewing overwrites |
| GET    | `/api/auth/login`    | 302 to Discord's consent screen; sets a short-lived state cookie |
| GET    | `/api/auth/callback` | Discord lands here; verifies state, upserts the user, sets `fuzion_session`, 302 to `/` |
| POST   | `/api/auth/logout`   | deletes the session server-side, clears the cookie, 204 |
| GET    | `/api/auth/me`       | `{id, username, avatarUrl}` for the cookie's user, or 401 |

These are exactly what the frontend's `src/api/` modules call.

## Sample data

```
psql "$DATABASE_URL" -f seed/dev_seed.sql
```

Fills the three tables so the home page renders populated in local dev.

## Tests

Each feature package owns one sociable test file that drives the real router →
real service → real repo → a migrated Postgres container. `internal/testutil`
starts one container per package run (`testutil.Run` from `TestMain`) and
truncates the tables for each test (`testutil.DB`). The only thing faked is
Discord: `testutil.NewFakeDiscord` is an `httptest.Server` that plays Discord's
OAuth2 token and `/users/@me` endpoints, and `testutil.NewServer` points the
real `auth.Discord` provider at it. `srv.LoginAs(t, discordID, username)`
inserts a user and session directly and drops the cookie in the test client's
jar, so member-only tests do not have to walk the OAuth dance.

Live stream rows are read from Postgres, not from Twitch. A Twitch sync would
be the next provider boundary, built the same way.

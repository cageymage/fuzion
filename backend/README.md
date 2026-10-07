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
| `TWITCH_CLIENT_ID`      | optional | `cmd/sync` only; from a dev.twitch.tv app. The Twitch job skips itself unless both are set |
| `TWITCH_CLIENT_SECRET`  | optional | same place; never commit it        |

Background jobs run from `cmd/sync`, one job per invocation, once and exit
(Render runs `/sync twitch` every minute). Locally, with the API already run
once so migrations are applied:

```
npx dotenv -e .env -e .env.example -- go run -C backend ./cmd/sync twitch
```

The job reads `streams.twitch_login`, marks those channels live or offline, and
writes a row to `sync_log` with the outcome. It asks Twitch only for streams in
the retail `world-of-warcraft` category, so a guildie who is live in another
category is marked offline (spec section 5). It also stores each live stream's
title and a thumbnail URL with a fixed 440x248 size.

Migrations under `migrations/` are embedded and applied on startup, so there
is no separate migrate step.

## Endpoints

| Method | Path                | Response                                            |
| ------ | ------------------- | --------------------------------------------------- |
| GET    | `/api/health`       | `{"status":"ok"}`                                   |
| GET    | `/api/professions`  | crafting directory with each character embedded, sorted by profession, then skill; optional `?profession=` (case-insensitive, `[]` when nothing matches) |
| POST   | `/api/professions`  | officer only: `{characterId, profession, skillLevel}`; 201 with the same shape the list returns; 400 for an unknown `characterId`, a blank `profession` or a negative `skillLevel`; 409 when the character already has that profession |
| PATCH  | `/api/professions/{id}` | officer only: any of `profession`, `skillLevel`; 200 with the updated row; 404 for an unknown or malformed id; 409 when it would duplicate another of the character's professions |
| DELETE | `/api/professions/{id}` | officer only: 204; 404 for an unknown or malformed id |
| GET    | `/api/news`         | one page of news posts as HAL: `{total, _links: {self, first, last, prev?, next?}, _embedded: {news: [...]}}`, newest first (`news` is `[]` when none); `?limit=` 1 to 50 (default 10), `?offset=` 0 or more (default 0), `?category=raid-progress\|recruitment\|guild-news`, else 400; `total` counts the category matches, ignoring limit and offset; links keep the filters and leave out defaults (`offset=0` never appears, `limit` only when you sent it) |
| GET    | `/api/raids/next`   | soonest upcoming raid, or `null` when none scheduled |
| GET    | `/api/streams`      | every channel, live ones first, then offline by name; each has `title` (empty when offline) |
| GET    | `/api/streams/live` | live streamers, most viewers first (`[]` when none); each has `title` |
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

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

Migrations under `migrations/` are embedded and applied on startup, so there
is no separate migrate step.

## Endpoints

| Method | Path                | Response                                            |
| ------ | ------------------- | --------------------------------------------------- |
| GET    | `/api/health`       | `{"status":"ok"}`                                   |
| GET    | `/api/news`         | news posts, newest first (`[]` when none)           |
| GET    | `/api/raids/next`   | soonest upcoming raid, or `null` when none scheduled |
| GET    | `/api/streams/live` | live streamers, most viewers first (`[]` when none) |

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
truncates the tables for each test (`testutil.DB`). Nothing is mocked — there
is no external provider behind these three endpoints yet.

Live stream rows are read from Postgres, not from Twitch. When a Twitch sync
is added it becomes the first real provider boundary here: an interface next to
this package, faked with `httptest.Server` in tests.

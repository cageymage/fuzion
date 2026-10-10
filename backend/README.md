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
| `SYNC_BLIZZARD_ROSTER_ENABLED` | `false` | `cmd/sync` only; the Blizzard roster job does nothing unless this is `true` |
| `BLIZZARD_CLIENT_ID`    | when enabled | from a develop.battle.net client |
| `BLIZZARD_CLIENT_SECRET` | when enabled | same place; never commit it |
| `BLIZZARD_REGION`       | `us` | API host region |
| `BLIZZARD_NAMESPACE`    | `profile-<region>` | the game version's profile namespace, e.g. `profile-classic1x-us`; Forever's value is not known yet |
| `GUILD_REALM_SLUG`      | when enabled | the guild's realm in Blizzard's slug form, e.g. `area-52` |
| `GUILD_NAME_SLUG`       | when enabled | the guild name as a slug (lowercase, spaces to hyphens), e.g. `my-guild` |
| `BATTLENET_REDIRECT_URL` | optional | enables Battle.net account linking on `/dashboard`; must be registered on the Blizzard client, `http://localhost:5173/api/auth/battlenet/callback` in dev. Needs `BLIZZARD_CLIENT_ID` and `BLIZZARD_CLIENT_SECRET` on the API too |

Background jobs run from `cmd/sync`, one job per invocation, once and exit
(Render runs `/sync twitch` every minute). Locally, with the API already run
once so migrations are applied:

```
npx dotenv -e .env -e .env.example -- go run -C backend ./cmd/sync twitch
```

The job reads `streams.twitch_login`, marks those channels live or offline, and
writes a row to `sync_log` with the outcome. It asks Twitch for streams in any
game: a guildie live in retail WoW (game id 18122) is `isLive`, one live in
another game is `isLiveOtherGame` (never shown on the home page), and anyone
else has both flags false. It also stores each live stream's title and a
thumbnail URL with a fixed 440x248 size.

`sync blizzard-roster` pulls the guild roster from Blizzard once an hour and
is off until `SYNC_BLIZZARD_ROSTER_ENABLED=true`. For each member it adds the
character if the roster lacks it, or refreshes class, level, race and guild rank if
it exists (matched on name and realm, ignoring case). Race is set from the
roster when its id is known (Classic races only; Skyborne ids are not mapped yet, so
those members keep whatever race an officer set and show up as `unmapped race N` in
the `sync_log` message). New characters get the
secondary name `Unset` and role `dps` because Blizzard supplies neither; the
sync never touches role, spec, main/alt, raid team or owner after that. A
character the sync saw before and no longer sees gets `left_guild_at` and
drops off `GET /api/roster`; characters entered by hand that Blizzard never
listed are left alone. Members with a class the site lacks (Death Knight,
Monk, Demon Hunter, Evoker exist on retail only) are skipped and counted in
the `sync_log` message, and an empty or failed Blizzard response changes
nothing.

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
| GET    | `/api/streams`      | every channel, live ones first, then offline by name; each has `title` (empty when offline), `isLive` (WoW) and `isLiveOtherGame` |
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

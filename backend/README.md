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
| `DISCORD_GUILD_ID`      | optional | the guild Discord server's ID (Developer Mode, right-click the server, Copy Server ID). When set, login also asks for the `guilds` scope and records whether the user is in that server. Only users in the server count as guild members (`isGuildMember` on `/api/auth/me`), and only guild members can answer surveys; responses also carry the flag for officers. It is read at login and kept on the account, so it is as old as the person's last login (up to 30 days for someone who stays logged in; logging out and in refreshes it). Without it every logged-in user counts as a guild member, the stored flag is false for everyone, and the officer "only people in our Discord server" filter hides every response |
| `RAID_HELPER_WEBHOOK_SECRET` | optional | Raid-Helper's webhook key from `/webhooks show`; without it `POST /api/webhooks/raid-helper` is not registered (404) |
| `TWITCH_CLIENT_ID`      | optional | `cmd/sync` only; from a dev.twitch.tv app. The Twitch job skips itself unless both are set |
| `TWITCH_CLIENT_SECRET`  | optional | same place; never commit it        |
| `YOUTUBE_API_KEY`       | optional | YouTube Data API v3 key. The API uses it for the suggested video; `cmd/sync youtube` skips itself without it |
| `SYNC_BLIZZARD_ROSTER_ENABLED` | `false` | `cmd/sync` only; the Blizzard roster job does nothing unless this is `true` |
| `BLIZZARD_CLIENT_ID`    | when enabled | from a develop.battle.net client |
| `BLIZZARD_CLIENT_SECRET` | when enabled | same place; never commit it |
| `BLIZZARD_REGION`       | `us` | API host region |
| `BLIZZARD_NAMESPACE`    | `profile-<region>` | the game version's profile namespace, e.g. `profile-classic1x-us`; Forever's value is not known yet |
| `GUILD_REALM_SLUG`      | when enabled | the guild's realm in Blizzard's slug form, e.g. `area-52` |
| `GUILD_NAME_SLUG`       | when enabled | the guild name as a slug (lowercase, spaces to hyphens), e.g. `my-guild` |
| `BLIZZARD_ALT_RANKS`    | optional | comma-separated guild rank numbers that hold alts, e.g. `4,5` (0 is the guild master). When set, the roster sync marks members of those ranks as alts and every other rank as mains; unset, every new character is an alt |
| `SUBMISSIONS_ENABLED`   | `true` | emergency switch for art and screenshot submissions: `false` makes uploads return 503 and the submission page show a closed message, while officers keep reviewing, downloading and deleting. Unset means on. Needs a restart or redeploy to change |
| `SUBMISSIONS_STORAGE_LIMIT_MB` | `512` | total MiB all submission originals may hold; past it, new submissions are refused (503) and the page shows the closed message |
| `BATTLENET_REDIRECT_URL` | optional | enables Battle.net account linking on `/dashboard`; must be registered on the Blizzard client, `http://localhost:5173/api/auth/battlenet/callback` in dev. Needs `BLIZZARD_CLIENT_ID` and `BLIZZARD_CLIENT_SECRET` on the API too, and the API reads `BLIZZARD_REGION` and `BLIZZARD_NAMESPACE` to list the member's characters |

Background jobs run from `cmd/sync`, one job per invocation, once and exit
(Render runs `/sync twitch` every minute and `/sync youtube` every 5 minutes). Locally, with the API already run
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

`sync youtube` does the same for rows with a `youtube_channel_id` (the `UC...`
ID, not the `@handle`). A row has a Twitch login or a YouTube channel ID, never
both, so a member who streams on both gets two rows and each job only touches
its own. YouTube has no game category, so any live stream counts as `isLive`.
For each channel it reads the 10 most recent uploads from the channel's uploads
playlist and looks them up with `videos.list`: a live stream is one of the
uploads, though Shorts posted during it can push it below the newest. A channel
with no uploads playlist is skipped. Each run costs about one quota unit per
channel plus two; quota errors (`quotaExceeded`) land in `sync_log` like any
other failure.

To try it locally, add a row for a channel that is live now:

```
INSERT INTO streams (streamer_name, game_name, channel_url, youtube_channel_id)
VALUES ('Lofi Girl', '', 'https://www.youtube.com/@LofiGirl', 'UCSJ4gkVC6NrvII8umztf0Ow');
```

then run `npx dotenv -e .env -e .env.example -- go run -C backend ./cmd/sync youtube`.

`sync blizzard-roster` pulls the guild roster from Blizzard once an hour and
is off until `SYNC_BLIZZARD_ROSTER_ENABLED=true`. For each member it adds the
character if the roster lacks it, or refreshes class, level, race and guild rank if
it exists (matched on name and realm, ignoring case). Race is set from the
roster when its id is known (Classic races only; Skyborne ids are not mapped yet, so
those members keep whatever race an officer set and show up as `unmapped race N` in
the `sync_log` message). New characters get the
secondary name `Unset` and role `dps` because Blizzard supplies neither; the
sync never touches role, spec, raid team or owner after that. Main/alt follows
`BLIZZARD_ALT_RANKS`: with it unset, new characters are alts and the sync never
changes main/alt; with it set, members of those ranks are alts and all others are
mains, on every run, so a rank change in game moves the character. Once an officer
sets main/alt by hand (on a new character, or by changing it on an existing one),
`is_main_set_by_hand` is set and the sync stops changing it, whatever the rank. Rank
numbers for Forever are not known until the first real sync: read them from
`characters.guild_rank`, then set the variable. A
character the sync saw before and no longer sees gets `left_guild_at` and
drops off `GET /api/roster`; characters entered by hand that Blizzard never
listed are left alone. Members with a class the site lacks (Death Knight,
Monk, Demon Hunter, Evoker exist on retail only) are skipped and counted in
the `sync_log` message, and an empty or failed Blizzard response changes
nothing.

Raid-Helper events reach the `raids` table through its webhook (Premium only).
Point `/webhooks set` for `event.create`, `event.update` and `event.delete` at
`https://<site>/api/webhooks/raid-helper` (one URL serves all three: the type is
in the `X-Raidhelper-Event` header) and leave `comp.update` unset. The title
becomes `instance_name`, `startTime` becomes `starts_at`, and the difficulty is
`Normal`, `Heroic` or `Mythic` when the title names one, else `Unknown`. An
update whose `lastUpdated` is strictly older than the stored one is skipped. A
delete keeps the row with `deleted_at` set, which hides it from `/api/raids` and
stops a late create or update from bringing it back. Hand-entered raids (no
`raid_helper_event_id`) are never touched. Payload details and the captures the
tests use are in [plans/integrations/raid-helper.md](../plans/integrations/raid-helper.md).

Every `/api/submissions` response carries `X-Content-Type-Options: nosniff` and
`Cache-Control: private`. Submissions are never publicly readable, unlike
`/api/images`. Originals are kept byte for byte, EXIF data included.

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
| POST   | `/api/me/characters/{id}/professions` | login required, owner only: `{profession, skillLevel}` on a character the caller owns; 201 in the same shape as `GET /api/professions`. Same validation and 409 as the officer endpoint; 403 when someone else or nobody owns the character, 404 when it is unknown |
| PATCH  | `/api/me/professions/{id}` | login required, owner only: any of `profession`, `skillLevel` on a row of a character the caller owns; 200 with the updated row; 403 for someone else's or an unowned character, 404 when unknown. Officers and members edit the same rows and the last write wins |
| DELETE | `/api/me/professions/{id}` | login required, owner only: 204; 403 and 404 as for PATCH |
| GET    | `/api/news`         | one page of news posts as HAL: `{total, _links: {self, first, last, prev?, next?}, _embedded: {news: [...]}}`, newest first (`news` is `[]` when none); `?limit=` 1 to 50 (default 10), `?offset=` 0 or more (default 0), `?category=raid-progress\|recruitment\|guild-news`, else 400; `total` counts the category matches, ignoring limit and offset; links keep the filters and leave out defaults (`offset=0` never appears, `limit` only when you sent it) |
| GET    | `/api/raids/next`   | soonest upcoming raid, or `null` when none scheduled |
| POST   | `/api/webhooks/raid-helper` | Raid-Helper only: needs `Authorization: <webhook key>`, else 401. `X-Raidhelper-Event: webhookeventcreate\|webhookeventupdate` upserts the raid by event id, `webhookeventdelete` tombstones it; any other type is a 200 that writes nothing. 400 when the body has no `id` or `startTime` |
| GET    | `/api/streams`      | every channel, live ones first, then offline by name; each has `title` (empty when offline), `isLive` (WoW) and `isLiveOtherGame` |
| GET    | `/api/streams/live` | live streamers, most viewers first (`[]` when none); each has `title` |
| GET    | `/api/applications`  | officer only: applications newest first, `?status=pending\|accepted\|declined` optional (400 on unknown) |
| GET    | `/api/applications/{id}` | officer only: one application in full; 400 for a non-UUID id, 404 when unknown |
| PATCH  | `/api/applications/{id}` | officer only: `{status: "accepted"\|"declined", reviewNote?}`; stamps `reviewedBy` and `reviewedAt`, re-reviewing overwrites |
| GET    | `/api/auth/login`    | 302 to Discord's consent screen; sets a short-lived state cookie |
| GET    | `/api/auth/callback` | Discord lands here; verifies state, upserts the user, sets `fuzion_session`, 302 to `/` |
| POST   | `/api/auth/logout`   | deletes the session server-side, clears the cookie, 204 |
| GET    | `/api/auth/me`       | `{id, username, avatarUrl}` for the cookie's user, or 401 |
| GET    | `/api/auth/battlenet/callback` | Battle.net lands here after `/api/auth/battlenet/link`: stores the Battle.net ID and BattleTag, then reads the account's characters from `/profile/user/wow` and makes the member the owner of the roster characters with the same name and realm (ignoring case). Characters another member owns stay theirs, and ones the account no longer has are released. The token is not stored, so linking again refreshes the set. 302 to `/dashboard` |
| DELETE | `/api/auth/battlenet` | unlinks Battle.net and releases every character the member owns, 204 |
| GET    | `/api/me/characters` | login required: the caller's characters still in the guild, in the roster's shape, mains first (`[]` when none). Each profession carries its row `id`, for the `/api/me/professions` routes |
| PATCH  | `/api/me/characters/{id}` | login required, owner only: `{isMain}` and nothing else (raid team and the rest stay with officers); making a character a main turns the owner's other mains into alts. Both count as set by hand, so the roster sync's rank mapping leaves them alone. 403 when someone else or nobody owns it, 404 when unknown, 400 without `isMain` |
| GET    | `/api/submissions/status` | public: `{open, attestationVersion}`; `open` is false when `SUBMISSIONS_ENABLED=false` or storage is full |
| POST   | `/api/submissions`   | guild members only: multipart, a `metadata` JSON part (`{attestationAccepted, attestationVersion, descriptions}`) first, then up to 10 `files` parts (PNG, JPEG, WebP or GIF, sniffed from the bytes, at most 20 MiB each, 50 MiB in total, 50 megapixels). `descriptions[i]` belongs to the i-th file, at most 500 characters. 201 with `{results: [{filename, submission?, error?}]}` when any file was stored; 400 or 409 (all duplicates) when none were. 400 without the attestation or with an out-of-date `attestationVersion`; 413 over 50 MiB; 429 over 20 files or 100 MiB per member per rolling 24 hours; 503 when closed or storage is full |
| GET    | `/api/submissions`   | officer only: `{total, limit, offset, items}`, newest first, metadata and URLs only (never image bytes); `?limit=` 1 to 100 (default 48), `?offset=` |
| GET    | `/api/submissions/{id}/thumb` | officer only: an 800px WebP preview |
| GET    | `/api/submissions/{id}/full` | officer only: the exact original, as the type sniffed at upload |
| GET    | `/api/submissions/{id}/download` | officer only: the original as `Content-Disposition: attachment` with its sanitized original filename |
| DELETE | `/api/submissions/{id}` | officer only: hard-deletes the original and thumbnail, 204; 404 when unknown. Works while submissions are closed |
| POST   | `/api/submissions/bulk-delete` | officer only: `{ids}` (1 to 100), 200 with `{deleted}`, the ids that existed; 400 for a malformed id. Every delete is logged with the officer's id |
| GET    | `/api/surveys`       | every survey, latest deadline first: `[{id, title, closesAt, status}]`, `status` being `open` or `closed` |
| GET    | `/api/surveys/{id}`  | the survey with its questions in order (`{id, prompt, type: "single"\|"multi", required, options}`) and `myResponse: {guildName, answers: [{questionId, values}]}` for a logged-in caller who has answered, else `null`; 404 for an unknown or malformed id |
| PUT    | `/api/surveys/{id}/response` | login required: `{guildName?, answers: [{questionId, values}]}` creates or replaces the caller's response, 200 with what was saved; 400 for an unanswered required question, a value that is not an option, two values on a single-choice question, a question from another survey or a guild name over 64 characters; 409 after the deadline; 404 for an unknown survey. Saving over a voided response clears the void |
| GET    | `/api/surveys/{id}/results` | 403 until the deadline passes, then public: `{surveyId, title, turnout, questions: [{id, prompt, type, options: [{label, count}]}]}` with voided responses left out and unchosen options at 0 |
| POST   | `/api/surveys`       | officer only: `{title, description?, closesAt, questions: [{prompt, type, required, options \| preset}]}`, where `preset` is `wow-classes`, `wow-roles` or `weekdays`; 201 with the survey as respondents see it; 400 for a blank or over-long title, a deadline that is missing or not in the future, no questions, a bad `type`, an unknown preset, both `options` and `preset`, or fewer than two distinct options |
| PATCH  | `/api/surveys/{id}`  | officer only: any of `title`, `description`, `closesAt`, `questions`; a `closesAt` of now closes the survey; 409 when `questions` is sent after any response exists; 404 for an unknown survey |
| GET    | `/api/surveys/{id}/responses` | officer only: every response, oldest first, voided ones included, no paging: `{id, discordUsername, guildName, inDiscordServer, submittedAt, updatedAt, voidedAt, voidedBy, voidReason, answers}` |
| PATCH  | `/api/surveys/{id}/responses/{responseId}` | officer only: `{voided, reason?}` with a reason of at most 500 characters; voiding stamps the officer and time, unvoiding clears them and the reason; 404 when the response is not in that survey |
| GET    | `/api/surveys/{id}/responses.csv` | officer only: a CSV attachment with one column per question and multiple values joined by `; `; cells starting with `=`, `+`, `-`, `@`, tab or CR get a leading quote so spreadsheets do not run them as formulas |

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

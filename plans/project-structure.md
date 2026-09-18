# Fuzion — WoW Guild Site: Project & Test Structure

Status: **decisions confirmed below — ready to scaffold.**

Coding/testing rules derived from this plan live in [CLAUDE.md](../CLAUDE.md) at the repo root — read that alongside this doc when generating code.

## Stack summary

| Layer     | Choice                                             |
|-----------|-----------------------------------------------------|
| Frontend  | React 18 + TypeScript + Vite                        |
| FE tests  | Vitest + React Testing Library + MSW                |
| Backend   | Go (net/http + chi router)                          |
| DB access | sqlx (raw SQL + struct scanning — Dapper equivalent) |
| BE tests  | Go stdlib testing + httptest + testcontainers-go     |
| Database  | Postgres (migrations via golang-migrate)             |
| E2E       | Playwright (thin layer, critical paths only)         |
| Hosting   | Render (static site for FE, web service for BE, managed Postgres) |

Monorepo (single git repo, two top-level apps) — simplest for a solo/small project, avoids cross-repo versioning overhead.

## Prerequisites for a fresh clone

`npm install` is not sufficient on its own. Running the site and its full test suite requires:

- **Node.js + npm** — frontend dev/build, Vitest.
- **Go toolchain** — build/run the backend, `go test` on the host.
- **Docker or Podman** — required for two things: the `docker-compose.yml` local-dev stack, and `testcontainers-go` in the backend test suite (spins up a real ephemeral Postgres per test run — this is what lets tests hit a real DB instead of a fake, per [CLAUDE.md](../CLAUDE.md)). Podman works as a drop-in: it exposes a Docker-compatible socket; point `DOCKER_HOST` at it and set `TESTCONTAINERS_RYUK_DISABLED=true` (Podman doesn't need testcontainers' reaper container).
- **Playwright browser binaries** — auto-downloaded via `npx playwright install` the first time E2E tests run; not a separate manual install, just a first-run download.

This is a deliberate tradeoff, not an oversight: it's what makes "automated tests never depend on anything outside the dev environment" (see [CLAUDE.md](../CLAUDE.md)) compatible with testing against a real Postgres instead of a mock. README should state these prerequisites up front.

**GitHub repo: private.** Since it's private, secrets discipline is slightly more relaxed but still: never commit `.env` files or real Discord/DB credentials — use `.env.example` with placeholder values and real secrets only in Render's env var UI / local `.env` (gitignored).

## Hosting

All three pieces live on **Render**, under one dashboard/account, deployed from the same GitHub repo and declared in [render.yaml](../render.yaml) (a Render Blueprint):

- **Frontend** — Render Static Site, built from `frontend/` (`npm run build`, publish `dist/`). Auto-deploys on push to `main`, PR previews available. A rewrite rule proxies `/api/*` to the backend service, so the browser only ever talks to the frontend's origin — the same shape as the Vite dev proxy. No CORS in prod, and no cross-subdomain cookie handling (see decision 5).
- **Backend** — Render Web Service, built from `backend/`'s Dockerfile. Auto-deploys on push to `main`. Binds to Render's injected `PORT`.
- **Database** — Render managed Postgres instance, connected to the backend service via an internal connection string (env var). The free tier expires after 30 days, after which it needs a paid instance (~$6/mo) — worth budgeting for up front so nothing breaks unexpectedly later. The free web service also spins down after 15 minutes idle (~30–60s cold start), so a paid Starter instance (~$7/mo) is the realistic floor once people are actually using it.

CI/CD is just Render's native GitHub integration (build + deploy on push) — no separate GitHub Actions deploy step needed, keeping the "push and forget" workflow you liked from Netlify. GitHub Actions is still used, but only for running the test suites (Vitest, Go tests, Playwright) as a required check before merge — deploys themselves are handled by Render, not the Actions workflow.

Local dev still uses `docker-compose.yml` (Postgres + backend + Vite dev server) so nothing depends on Render being reachable while developing.

## Top-level layout

```
fuzion/
├── frontend/                # React app
├── backend/                 # Go app
├── e2e/                     # Playwright, cross-cutting FE+BE tests
├── plans/                   # planning docs (this file)
├── .github/
│   └── workflows/
│       └── test.yml         # CI: runs Vitest, Go tests, Playwright on PR/push
├── docker-compose.yml       # local Postgres + backend + frontend for dev
├── .env.example             # documents required env vars (DB, Discord OAuth, session secret) — no real values
├── .gitignore                # node_modules, dist/build output, .env, go build artifacts, docker volumes
└── README.md                 # what this is, local dev setup (docker-compose up), env vars needed
```

## Frontend structure

```
frontend/
├── src/
│   ├── app/                 # routing, top-level providers, App.tsx
│   ├── pages/                # one folder per route/page
│   │   └── Roster/
│   │       ├── Roster.tsx
│   │       ├── Roster.test.tsx      # sociable page test (real children, MSW-mocked API)
│   │       └── useRosterData.ts
│   ├── components/           # shared/reusable presentational components
│   │   └── RaidCard/
│   │       ├── RaidCard.tsx
│   │       └── RaidCard.test.tsx     # component-level render test
│   ├── api/                  # typed API client (fetch wrappers), one file per backend resource
│   │   ├── client.ts         # fetch wrapper: base URL, credentials: 'include' for session cookie
│   │   ├── roster.ts
│   │   └── raids.ts
│   ├── lib/                  # pure helper/utility functions (unit-testable)
│   ├── mocks/                 # MSW setup (automated tests only — see note below)
│   │   ├── handlers.ts       # request handlers per API resource
│   │   └── server.ts         # Node server used by Vitest
│   ├── types/                # shared TS types (mirrors backend DTOs)
│   ├── main.tsx
│   └── setupTests.ts         # Vitest setup: jest-dom matchers, MSW server lifecycle
├── vite.config.ts
├── vitest.config.ts
├── tsconfig.json
└── package.json
```

Routing: **React Router**. Data fetching/caching: **TanStack Query** (pairs with the `useRosterData.ts`-style hooks already sketched above — handles loading/error/caching state instead of hand-rolling it per page).

**Testing conventions (FE):**
- Co-locate `*.test.tsx` next to the component/page it tests.
- **Page tests** render the full page tree (real routing context, real child components, real state/query hooks) and only intercept HTTP at the network boundary via MSW — never mock hooks, contexts, or child components directly. This is the "sociable" equivalent of a bUnit full-page render.
- **Component tests** render a single component in isolation with props, asserting on rendered output/interactions via RTL queries (`getByRole`, `userEvent`), not implementation details.
- **Unit tests** for pure logic in `lib/` (e.g. loot priority calculations) — plain Vitest, no rendering.
- MSW handlers default to "happy path" in `mocks/handlers.ts`; individual tests override with `server.use(...)` for error/edge cases.
- MSW is wired for **automated tests only** (Vitest). Local dev always runs against the real Go backend via `docker-compose.yml` — no browser-mode MSW mock server, so local dev behaves the same as prod.

## Backend structure

```
backend/
├── cmd/
│   └── api/
│       └── main.go           # composition root: wires real DB, real services, real router
├── internal/
│   ├── server/
│   │   ├── server.go          # http.Server + router setup, takes dependencies via struct
│   │   └── routes.go          # also wires CORS middleware (only used when a frontend hits the API directly, not via the dev/prod proxy)
│   ├── roster/
│   │   ├── handler.go         # HTTP handlers
│   │   ├── service.go         # business logic
│   │   ├── repo.go            # Postgres queries via sqlx (raw SQL + struct scanning)
│   │   └── roster_test.go     # sociable test: real handler+service+repo, real test DB
│   ├── raids/
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── repo.go            # sqlx
│   │   └── raids_test.go
│   ├── blizzard/              # external API client (the true "edge")
│   │   ├── client.go
│   │   └── client_test.go     # mocked via httptest.Server standing in for Blizzard API
│   └── testutil/
│       ├── testserver.go      # spins up httptest.NewServer wrapping the real router
│       └── testdb.go          # testcontainers-go Postgres lifecycle + migrations
├── migrations/                # SQL migration files
├── go.mod
└── go.sum
```

**Testing conventions (BE):**
- Each feature package (`roster`, `raids`, ...) gets a **sociable test** that:
  1. Spins up a real Postgres via testcontainers-go (shared per test package run, migrated fresh).
  2. Constructs the real service and repo against that DB.
  3. Wraps the real router (with the real handler registered) in `httptest.NewServer`.
  4. Sends real HTTP requests via `http.Client` and asserts on real JSON responses and real DB state.
- Only genuine external edges (Blizzard/Warcraft Logs API, Discord webhooks) are swapped for fakes — via an interface + either a hand-written fake or an `httptest.Server` imitating the third-party API's contract.
- Pure logic (e.g. loot rules, raid comp validation) gets plain table-driven unit tests, no server/DB involved.
- `testutil` centralizes the boilerplate so every feature test file stays short: `srv := testutil.NewServer(t, db)`.
- Migrations are plain SQL files under `migrations/`, run via `golang-migrate` — both by `testutil.testdb.go` (fresh migrate on each test run) and by a startup step in `cmd/api/main.go` (or a one-off Render deploy job) in real environments.

## E2E (Playwright)

```
e2e/
├── tests/
│   └── raid-signup.spec.ts   # login -> view roster -> sign up for raid
├── playwright.config.ts
└── package.json
```

Runs against the full stack via `docker-compose.yml` (real Postgres, real Go backend, real built frontend). Kept small — a handful of critical user journeys only, not a substitute for the sociable test layers above.

## Decisions

1. **Router: `chi`.** Chosen over `gin` and stdlib `net/http`: chi stays a thin layer directly on top of `net/http`'s own types (`http.Handler`, `http.ResponseWriter`, `*http.Request`), so what you learn transfers to idiomatic Go rather than to a framework-specific API (gin's `*gin.Context` diverges from stdlib and is more of an ASP.NET-Core-style all-in-one framework). It's also one of the most widely used Go routers, so model support/training coverage is strong. Coming from C#, expect routing + middleware chaining (`r.Use(...)`) to feel roughly analogous to ASP.NET Core middleware pipelines, just far more minimal.
2. **Auth: Discord OAuth first, Battle.net OAuth added later if wanted.** The `internal/auth` package should define a small provider-agnostic interface (exchange code → get user identity) so a second provider is a new implementation, not a rewrite. Only Discord ships in the initial scaffold.
3. **MSW usage clarified:** MSW's Node server (`mocks/server.ts`) is used only inside the Vitest suite to intercept HTTP in sociable page tests. Local development (`npm run dev` / debugging) always talks to the real Go backend via `docker-compose.yml`, matching prod behavior — no browser-mode MSW mock server is scaffolded.
4. **DB access: `sqlx`.** Closest Go equivalent to Dapper — raw SQL, results scanned onto structs via `db:"..."` tags at runtime, no ORM query builder and no code-generation build step. Chosen over `sqlc` (build-time codegen, more type-safe but an extra tooling step) and `GORM` (full ORM, more magic, less idiomatic Go).
5. **Single origin via proxy; custom domain is optional.** The Render static site rewrites `/api/*` to the backend service, so the browser never sees a second origin. The session cookie is a plain `httpOnly; SameSite=Lax` cookie on the site's own host — no `Domain=` attribute, no CORS credentials dance. A custom domain (e.g. `fuzion.gg`) is still planned for vanity, but it's a DNS change, not a prerequisite for auth; the original cross-subdomain plan (`app.` / `api.` with `Domain=.fuzion.gg`) was dropped in favour of this because it removes a whole class of cookie/CORS problems for the cost of one proxy hop.
6. **GitHub repo: private.**

## Auth (Discord OAuth)

```
backend/
└── internal/
    └── auth/
        ├── provider.go     # Provider interface: AuthURL(state), Exchange(ctx, code) (Identity, error)
        ├── discord.go      # Discord OAuth implementation
        ├── handler.go      # /auth/login, /auth/callback, /auth/logout HTTP handlers
        ├── session.go      # issues httpOnly SameSite=Lax session cookie (same-origin via the proxy, no Domain= attribute)
        └── auth_test.go    # sociable test against httptest.Server standing in for Discord's OAuth endpoints
```

- Discord OAuth2 flow: redirect to Discord's authorize URL → callback with `code` → exchange for token → fetch Discord user (id, username, avatar, guild membership) → issue our own session.
- `Provider` interface keeps Discord swappable/extendable; a `battlenet.go` implementing the same interface can be added later without touching handlers.
- Guild membership check (is this Discord user in *our* guild's server) can gate access at the session layer — useful since guild rosters usually already live in a Discord server.

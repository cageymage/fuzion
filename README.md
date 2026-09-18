# Fuzion

WoW guild site: Go API (chi + sqlx + Postgres) under `backend/`, React/TS
client (Vite) under `frontend/`. Architecture and rationale live in
[plans/project-structure.md](plans/project-structure.md); coding and testing
rules in [CLAUDE.md](CLAUDE.md).

## Quick start

Once the [prerequisites](#prerequisites) below are installed:

```
git clone https://github.com/cageymage/fuzion.git
cd fuzion
npm install     # root tooling + frontend deps (postinstall)
npm run dev     # Postgres via docker compose, Go API on 127.0.0.1:8080, Vite on :5173
npm run db:seed # sample news/raid/stream rows so the home page has content
```

Open http://localhost:5173. Vite proxies `/api` to the Go API, so the
browser only ever talks to one origin. Both servers hot-reload on save.

The first `npm run dev` is slow: it pulls the `postgres:16-alpine` image and
Go downloads/compiles the backend's modules. Subsequent runs take a few
seconds.

## Running the tests

```
npm test            # everything: Go suite, then Vitest
npm run test:api    # Go only — Docker Desktop must be running
npm run test:web    # Vitest only — no Docker needed
```

The Go suite runs against a real Postgres: each test package starts its own
throwaway container via testcontainers, so Docker Desktop has to be running,
but you do **not** need `npm run db:up` and a running dev database is left
alone. Frontend tests intercept HTTP with MSW and need nothing running.

To narrow the run, call the tools directly:

```
go test -C backend ./internal/streams/...                        # one Go package
go test -C backend ./internal/streams/ -run TestListLiveStreams  # tests matching a regex
npm --prefix frontend run test:watch                             # Vitest in watch mode
npm --prefix frontend test -- src/pages/Home                     # Vitest, files matching a path
```

If the Go tests panic mentioning rootless Docker, see the
[gotcha below](#go-tests-panic-rootless-docker-is-not-supported-on-windows).

## Prerequisites

Three tools, all required. `npm install` alone is not enough — the backend is
Go, and both local dev and the backend test suite run a real Postgres in
Docker (see [CLAUDE.md](CLAUDE.md) for why nothing is mocked).

| Tool          | Version | Check                       |
| ------------- | ------- | --------------------------- |
| Node.js + npm | 20+     | `node --version`            |
| Go            | 1.25+   | `go version`                |
| Docker        | any recent, with Compose v2 | `docker compose version` |

### Node.js

- **Windows:** `winget install OpenJS.NodeJS.LTS`, or the installer from
  https://nodejs.org.
- **macOS:** `brew install node`.
- **Linux:** your distro's package, or https://github.com/nvm-sh/nvm.

Open a **new** terminal afterwards so `node` and `npm` are on `PATH`.

### Go

- **Windows:** `winget install GoLang.Go`, or the MSI from https://go.dev/dl.
- **macOS:** `brew install go`.
- **Linux:** tarball from https://go.dev/dl, or your distro's package.

Open a new terminal afterwards. If your installed Go is older than the
`go 1.25.0` line in [backend/go.mod](backend/go.mod), Go 1.21+ will
auto-download the right toolchain on first build — no manual action needed.

### Docker

- **Windows:** Docker Desktop from https://www.docker.com/products/docker-desktop.
  It needs WSL 2 — see the gotcha below, it bit us on a fresh machine.
- **macOS:** Docker Desktop, or `brew install --cask docker`.
- **Linux:** Docker Engine + the `docker-compose-plugin`
  (https://docs.docker.com/engine/install). Podman also works: point
  `DOCKER_HOST` at its socket and set `TESTCONTAINERS_RYUK_DISABLED=true`.

Docker Desktop must be **running** (whale icon says "running") before
`npm run dev` or `npm run test:api` — the CLI being installed isn't enough.

## Gotchas we actually hit

### Docker Desktop: "Virtualization support not detected" (Windows)

Almost certainly WSL 2 isn't installed, not a BIOS problem. Windows 11 ships
a stub `wsl.exe` whose only job is to print "not installed", so `wsl --status`
saying that is the real answer. From an **elevated** PowerShell:

```powershell
wsl --install --no-distribution
```

then reboot, start Docker Desktop, and confirm Settings → General →
"Use the WSL 2 based engine" is ticked. `--no-distribution` skips
installing Ubuntu; Docker Desktop brings its own.

If it still complains after that, check virtualization is actually on:
`Get-CimInstance Win32_Processor | Select VirtualizationFirmwareEnabled`
should say `True`; if not, enable AMD-V / Intel VT-x in the BIOS.

### Go tests panic: "rootless Docker is not supported on Windows"

This message is misleading. It means testcontainers couldn't reach the Docker
daemon, and there are two reasons that happens:

1. **Docker Desktop isn't running.** `docker version` will show
   `open //./pipe/docker_engine: The system cannot find the file specified`.
   Start Docker Desktop and re-run.
2. **It's running, but one package fails randomly while the others pass.**
   `go test ./...` starts several test binaries in parallel, each spinning up
   its own Postgres container; testcontainers' host auto-detection stats the
   Docker named pipe, which can report busy under that load, and detection
   falls through to a strategy that panics on Windows. Pin the host so it
   uses the real Docker client instead — create
   `C:\Users\<you>\.testcontainers.properties` containing:

   ```
   docker.host=npipe:////./pipe/docker_engine
   ```

   This is per-machine config, not something to commit.

### Windows Firewall keeps asking about `api.exe`

Windows only prompts for listeners bound to all interfaces, and `go run`
builds to a fresh temp path every restart, so a `:8080` bind means a new
prompt every time nodemon restarts the API. [.env.example](.env.example)
sets `ADDR=127.0.0.1:8080` to avoid this — loopback binds never prompt.
If you override `ADDR` in `.env`, keep the `127.0.0.1` prefix. Deployed
builds leave `ADDR` unset and bind all interfaces on Render's `PORT`.

### Port already in use

`npm run dev` needs 5173 (Vite), 8080 (API) and 5432 (Postgres) free. A local
Postgres service is the usual 5432 culprit — stop it, or change the host port
in [docker-compose.yml](docker-compose.yml) and `DATABASE_URL` in `.env`.

If 8080 is stuck after a dev session, an orphaned `api.exe` is still running:
`taskkill /F /IM api.exe` (Windows) or `pkill api` (macOS/Linux).

## Scripts

All run from the repo root.

| Script        | What it does                                                |
| ------------- | ----------------------------------------------------------- |
| `dev`         | `db:up`, then API and web dev servers side by side          |
| `dev:api`     | just the Go API, restarts on `.go`/`.sql` changes (nodemon)  |
| `dev:web`     | just the Vite dev server                                    |
| `db:up`       | start Postgres and wait until it accepts connections        |
| `db:down`     | stop Postgres; data persists in the `pgdata` volume         |
| `db:seed`     | load `backend/seed/dev_seed.sql` into the running Postgres  |
| `test`        | Go tests then Vitest — see [Running the tests](#running-the-tests) |
| `test:api`    | Go tests only — needs Docker running, not `db:up`           |
| `test:web`    | Vitest only — no Docker needed, HTTP is intercepted by MSW  |

## Environment

`npm run dev` reads `.env` and then [.env.example](.env.example), so a
fresh clone runs with the compose defaults and nothing to copy. Put
overrides (or real secrets, later) in `.env` — it is gitignored.

Migrations under `backend/migrations/` are embedded in the API binary and
applied on startup, so there is no separate migrate step. To start from a
clean database: `docker compose down -v` then `npm run dev` again.

## Deploying

Production is three Render services declared in [render.yaml](render.yaml):
a static site for the frontend, a Docker web service for the API, and a
managed Postgres. Render deploys `main` on push; GitHub Actions
([test.yml](.github/workflows/test.yml)) runs the test suites on PRs.

The static site rewrites `/api/*` to the API service, so the browser only
ever talks to one origin — the same as the Vite proxy locally. No CORS
config or `VITE_API_BASE_URL` is needed in prod.

First-time setup:

1. Render dashboard → **New → Blueprint**, pick this repo. Render creates
   all three services from `render.yaml` and wires `DATABASE_URL`.
2. Check the API's hostname in the dashboard. If it isn't
   `fuzion-api.onrender.com` (the name was taken), update the `/api/*`
   rewrite destination in `render.yaml` and push.
3. Open the static site's URL; `/api/health` should return `{"status":"ok"}`
   and the home page should load (empty until there's content).
4. The prod DB starts empty. Until there's an admin UI, load content with
   `psql "<external connection string from the dashboard>" -f backend/seed/dev_seed.sql`
   (swap in real content first).

Free-tier caveats: the Postgres expires after 30 days, and the free web
service sleeps after 15 minutes idle with a slow cold start. Budget for
paid instances of both (~$13/mo) once people are actually using the site.

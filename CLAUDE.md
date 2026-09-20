# Fuzion — Guidelines for Claude Code

WoW guild site. Go backend (chi + sqlx + Postgres), React/TS frontend (Vite), Discord OAuth.
Full architecture and rationale live in [plans/project-structure.md](plans/project-structure.md) — read that first for stack choices. This file is about *how to write code and tests* in this repo.

## Architecture rules

- **Governing rule: automated tests never depend on anything outside the local dev environment.** If `docker-compose up` provides it, tests can use the real thing. If it's a live external service a dev machine has no business calling automatically, it gets faked. This is the actual test for what counts as a "provider" below — not an arbitrary layer choice.
- **Sociable by default.** Handlers call real services, real services call a real repo, the repo hits a real Postgres (via testcontainers-go in tests, same Docker dependency `docker-compose.yml` already requires for local dev). Never introduce a mock/fake for internal code (services, repos, business logic) — only things outside the dev-env boundary get faked.
- **"Provider" boundary = the only thing that gets faked.** A provider is the narrowest interface around something that fails the governing rule above — it's a live external network service, or inherently non-deterministic:
  - Third-party APIs: Discord OAuth, Battle.net OAuth, Blizzard/Warcraft Logs API. These aren't reachable from `docker-compose`'s stack, so tests never hit them for real.
  - Non-determinism needed for reproducible tests: current time, UUID/ID generation.
  - Postgres is **not** a provider to fake — it's part of the dev-env stack (`docker-compose.yml`), so tests run against a real instance. The DB is inside the "sociable" boundary, not outside it.
- When a new external dependency is introduced, wrap it in the smallest interface that lets tests substitute a fake or `httptest.Server`, next to the package that uses it (see `internal/blizzard`, `internal/auth`). Don't build a generic "providers" abstraction layer speculatively — add one when a second real need for it exists.
- Add `internal/clock` (a `Now() time.Time` interface) and `internal/idgen` (a `New() uuid.UUID` interface) the first time a test needs to assert on a generated timestamp or ID. Production code uses the real implementation; tests inject a fixed value. Don't add these preemptively before something needs them.
- Frontend mirrors this: MSW intercepting `fetch` at the network boundary is the FE's "provider" fake. Never mock hooks, contexts, or child components to make a test pass — if a page test needs that, the page is doing too much or the API layer needs a seam, not a mock.

## Code generation

- No comments explaining *what* code does — names should carry that. A comment is only for a non-obvious *why* (a workaround, a constraint from an external API, a subtle invariant).
- No speculative abstractions, interfaces, or config options for hypothetical future requirements. Three similar lines beat a premature helper.
- Match existing package/file structure in [plans/project-structure.md](plans/project-structure.md) rather than inventing new top-level folders per feature.
- Go: idiomatic stdlib-first code. chi is a thin router on top of `net/http` — don't reach for framework-style patterns (global middleware registries, magic context injection) that fight that.
- Errors are real values, checked and wrapped with context (`fmt.Errorf("...: %w", err)`), not swallowed or logged-and-ignored.

## Test coverage requirements

Every new handler (backend) or page/component (frontend) ships with tests in the same PR — no endpoint or page merges without a corresponding spec.

### Backend (Go)

- One test file per feature package covering all its endpoints (`roster_test.go` for the `roster` package), or one file per endpoint under a subfolder if a single endpoint's test file is getting large — never mix unrelated endpoints' assertions into one test function.
- Structure: real `httptest.Server` wrapping the real router → real service → real repo → real test-container Postgres. Only `internal/auth`'s Discord client and `internal/blizzard`'s client are faked (via interface + `httptest.Server` standing in for the third-party API).
- Table-driven tests are fine for pure logic (loot rules, raid comp validation) but each case still needs a readable name — see naming rule below. Don't reach for a table just to save typing if the cases aren't genuinely cohesive; separate `Test...` functions are clearer when scenarios diverge.
- Given/When/Then structure inside each test via comments:
  ```go
  func TestAddRaidSignup_ReturnsBadRequest_WhenRaidIsFull(t *testing.T) {
      // given a raid that already has the max signups
      ...
      // when I sign up for the raid
      resp := srv.Post(t, "/raids/"+raidID+"/signups", body)
      // then I expect a 400 with the following error
      ...
  }
  ```
- Assertions can be verbose and explicit — inline the full expected struct/response rather than building a shared "assert helper" that hides what's actually being checked. Tests should be easy to read standalone (DAMP), not maximally deduplicated (DRY).

### Frontend (Vitest + RTL + MSW)

- **Page tests**: render the full page (real router context, real children, real hooks/TanStack Query), intercept only HTTP via MSW. Never mock a child component or hook to isolate a page test.
- **Component tests**: render a single component with props, assert via RTL queries (`getByRole`, `userEvent`), not implementation details.
- MSW handlers default to happy path in `mocks/handlers.ts`; individual tests override with `server.use(...)` only for the specific case they're testing.
- Test descriptions follow the same naming rule as backend tests (below).

### Test naming (both FE and BE)

Pattern: **`<feature/endpoint> should <expected behavior> when <discriminating criteria>`** (the "when" clause is optional if there's no discriminating condition).

- Exactly one "when I ___" action per test — if a scenario needs two distinct actions to set up, that's two "given" steps, not two "when"s.
- Be specific. Avoid catch-all names like "handles errors correctly" or "works" — a future reader (or Claude, next session) should know exactly what broke from the test name alone in a failure list, without opening the file.
- Avoid long "and" chains in the "when expected behavior" part — if a test is asserting several unrelated things, split it into separate tests.
- Examples:
  - Go: `TestAddRaidSignup_ReturnsBadRequest_WhenRaidIsFull`
  - Vitest: `it("should show a full-raid message when the raid has no open signup slots")`

## Don't

- Don't mock Postgres or any internal package to make a backend test pass — fix the seam (dependency injection into the service/repo) instead.
- Don't add auth/session logic ad hoc in handlers — route it through `internal/auth`.
- Don't hand-roll `.env` parsing per package — one config load in `cmd/api/main.go`, passed down via structs.

## Git workflow

- Never `git commit` unless explicitly told to in that message. Finishing a task, passing tests, or a plan that ends in "then commit" is not permission — leave the work staged/unstaged and say it's ready.
- Never `git push` or open a PR unless explicitly told to. "One PR is fine" describes the shape of the deliverable, not a go-ahead to create it; ask first.

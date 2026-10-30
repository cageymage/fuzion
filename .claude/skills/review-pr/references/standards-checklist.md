# Standards checklist

Turn each rule into a concrete check against the diff. CLAUDE.md is the source of truth; read it if a rule here is unclear. Flag only what the diff adds or changes.

## Comments and code shape
- A comment that explains *what* the code does is a `should`. Only a non-obvious *why* (workaround, external API constraint, subtle invariant) earns a comment.
- New interface, helper, or config option with a single caller or no current need is a `should` (speculative abstraction). Three similar lines beat a premature helper.
- New top-level folders or files that do not follow `plans/project-structure.md` are a `should`.

## Go
- Returned errors wrapped with `fmt.Errorf("...: %w", err)`. Swallowed or log-and-ignore errors are a `must`.
- Sentinel errors matched with `errors.Is`/`errors.As`, not string comparison.
- Handlers decode JSON and map errors to status codes; validation lives in `service.go`.
- Auth or session logic written inside a handler instead of `internal/auth` is a `must`.
- Config parsed in a package instead of the single load in `cmd/api/main.go` is a `should`.
- Code that is not gofmt-clean (visible tabs/spacing problems) is a `nit`.
- `internal/clock` or `internal/idgen` added before a test needs a fixed time or ID is a `should`.

## Migrations
- New `.up.sql` has a matching `.down.sql`.
- Number is above the highest on the base branch (`git ls-tree <base> backend/migrations/`). CI fails numbers at or below it.

## Backend tests
- Every new handler ships with tests in the same PR (`must` if missing).
- One test file per feature package; unrelated endpoints not mixed in one test function.
- Real `httptest.Server` over real router, service, repo, testcontainers Postgres. Mocking Postgres or any internal package is a `must`. Only Discord, Battle.net, and Blizzard clients may be faked.
- Name pattern `Test<Feature>_<Expected>_When<Criteria>`, for example `TestAddRaidSignup_ReturnsBadRequest_WhenRaidIsFull`. Vague names ("works", "handles errors") are a `should`.
- Given/When/Then comments present, and exactly one "when" action per test.
- Assertions compare the full expected value inline (a whole struct or slice, `cmp.Diff`). Spot checks such as `len(rows) != 1 || rows[0].X != ...` are a `should`. A field the test cannot predict (DB `now()`, generated ID) should be asserted as set, cleared, then the rest diffed.
- Table tests only where cases are cohesive and each case has a readable name.

## Frontend
- Every new page or component ships with tests (`must` if missing).
- Page tests render the full page with real router, children, hooks, and TanStack Query, intercepting only HTTP with MSW. `vi.mock` of a hook, context, or child component is a `must`.
- Component tests use RTL queries (`getByRole`, `userEvent`), not implementation details.
- MSW default handlers stay happy-path in `mocks/handlers.ts`; tests override with `server.use(...)` only for the case under test.
- Test names follow `<feature> should <expected behavior> when <criteria>` with one action per test, no long "and" chains.

## Legal page
`frontend/src/pages/Legal/Legal.tsx` (guild site) or `hub/src/pages/Legal/Legal.tsx` (hub) must change in the same PR, with a Legal test, when a change to that site: collects or stores new personal data (form fields, linked accounts, cookies), adds a third party that receives data or is embedded, adds a user-uploaded content type, or adds an asset needing attribution. Missing update is a `must`. The PR body should say what changed on the page.

## Git and scope
- Changes unrelated to the linked issue are a `should` (suggest a separate PR).
- Stacked PRs: base is the parent branch, body starts with `Stacked on #N`, migration numbers sit above the parent's.

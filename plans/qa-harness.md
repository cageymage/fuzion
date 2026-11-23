# Local QA harness

Status: **proposed, not built.** Phase 1 is three stacked PRs, listed under [Build order](#build-order). Phase 2 is described under [Phase 2: move to CI](#phase-2-move-to-ci).

One on-demand command, "qa this", reviews a change, runs every automated suite plus a Playwright browser layer, checks the changed pages visually, and offers to fix what it finds.

- **Phase 1 (this plan's build order):** everything runs on a developer's machine through Claude Code. Nothing new runs in GitHub Actions and no API keys or GitHub secrets are added.
- **Phase 2:** the same checks move into CI, so every PR gets them without anyone starting a local session.

This builds out the E2E layer already planned in [project-structure.md](project-structure.md#e2e-playwright) and keeps to the testing rules in [CLAUDE.md](../CLAUDE.md): only providers outside the dev environment are faked.

## Goals

- One command for review and QA: `review-pr`, Go tests, both Vitest suites, Playwright, and a visual check, ending in one verdict.
- Browser coverage of the paths that cross the API boundary, with the external integrations as the first focus because nothing exercises them end to end today.
- Fixes on request, verified by the same checks, with commits and pushes still gated on an explicit go-ahead.

## Non-goals for Phase 1

- Running Claude in GitHub Actions or on a schedule. In Phase 1, reviews happen when someone runs the command.
- Adding Playwright to `test.yml`. The suite runs locally until it has proven stable, then moves to CI in Phase 2.
- Testing against Render or the live site.

## 1. E2E harness (`e2e/`)

Matches the layout in project-structure.md:

```
e2e/
├── fakes/                  # one local server standing in for every external provider
├── tests/                  # *.spec.ts, one file per journey
├── global-setup.ts         # starts Postgres, builds the API and sync binaries and both sites
├── auth.setup.ts           # setup project: logs in the first (admin) account, then the other personas
├── playwright.config.ts
└── package.json
```

### The stack

`npm run test:e2e` from the root starts an isolated copy of the stack, so it can run while `npm run dev` is up and never touches the dev database:

| Piece | Dev | E2E |
|---|---|---|
| Postgres | compose project `fuzion`, port 5432, `pgdata` volume | `docker-compose.e2e.yml`, project `fuzion-e2e`, port 55432, tmpfs (empty every run) |
| API | `127.0.0.1:8080` | `127.0.0.1:18080` via the existing `ADDR` variable, provider URLs pointed at the fakes |
| Guild site | Vite dev on 5173 | `vite build` then `vite preview` on 15173 (the "real built frontend" from project-structure.md) |
| Hub | Vite dev on 5174 | `vite build` then `vite preview` on 15174 |
| Fakes | none | `e2e/fakes` on 18090 |

The proxy target in both `vite.config.ts` files is fixed at `http://127.0.0.1:8080`, so it reads an `API_PROXY_TARGET` variable and keeps 8080 as the default. `vite preview` reuses `server.proxy`, and the hub keeps proxying only `/api/streams`.

`global-setup.ts` starts Postgres with `docker compose -p fuzion-e2e`, builds the API and `cmd/sync` binaries once, and builds both sites with E2E values for build-time variables such as `VITE_TURNSTILE_SITE_KEY`. Playwright's `webServer` list then starts the API, both `vite preview` servers and the fakes, and waits until each one is ready. Building the binaries once avoids paying for `go run`'s compile on every sync.

`playwright.config.ts` follows the standard Playwright defaults for debugging failures: `trace: 'retain-on-failure'`, a screenshot and a video (`video: 'retain-on-failure'`) on failure, and the HTML report, which plays the video inline. Retries stay at 0 locally, so a flaky test shows up instead of being hidden.

The `postinstall` script gains `npm --prefix e2e install`. The first run downloads Chromium with `npx playwright install chromium`, as project-structure.md already describes.

### Faking the providers

The API and `cmd/sync` hard-code every provider's base URL in `main.go` (`auth.DiscordAPIBaseURL`, `auth.BattleNetAPIBaseURL`, `blizzard.APIBaseURL(region)`, `blizzard.AuthBaseURL`, `twitch.*`, `youtube.APIBaseURL`, `turnstile.VerifyBaseURL`). Each gets an optional environment variable read in the existing config load, defaulting to today's constant:

`DISCORD_API_BASE_URL`, `BATTLENET_AUTH_BASE_URL`, `BLIZZARD_API_BASE_URL`, `BLIZZARD_AUTH_BASE_URL`, `TWITCH_API_BASE_URL`, `TWITCH_AUTH_BASE_URL`, `YOUTUBE_API_BASE_URL`, `TURNSTILE_VERIFY_BASE_URL`.

The three Discord webhook URLs are already variables, so they point at the fakes as they are.

The Discord and Battle.net authorize URLs are built from the same base URLs (`discord.go`, `battlenet.go`), so the browser's login redirect also lands on the fake. The fake's authorize endpoint redirects straight back to `DISCORD_REDIRECT_URL` or `BATTLENET_REDIRECT_URL` with a code for the chosen persona, and those variables point at the E2E guild site on 15173.

The fakes are a small TypeScript server in `e2e/fakes`, not a reuse of `internal/testutil`. The Go fakes take a `*testing.T` and assert inside their handlers, so they cannot run as a standalone process without a refactor that would change every backend test. The E2E fakes:

- Serve the OAuth, profile, roster, professions, stream and siteverify endpoints the clients call. Where a Go package has a `testdata` folder (`blizzard` and `raidhelper`), they read those fixtures directly rather than keeping copies, so the Go and E2E fakes answer with the same data. Discord, Battle.net, YouTube and Turnstile responses live inline in `internal/testutil/fake*.go`, and Twitch has no Go fake at all, so PR 1 writes new fixtures for those under `e2e/fakes/fixtures`. Those are the fixtures that can drift from the Go fakes (see [Decision 1](#decisions-to-confirm-in-review)).
- Record every request, webhook body and webhook edit they receive.
- Expose a control API (`POST /_control/discord/users`, `GET /_control/webhooks`, and so on) so a test can set up a persona or assert on exactly what was sent to "Discord".

### Test isolation

Playwright runs test files in parallel workers against one database and one fakes server, so no test may depend on state another test can change:

- Each test creates its own data with unique names (a persona with its own Discord ID, an application from its own applicant), and asserts only on that data.
- The fakes keep recordings per persona or per test-generated ID, and tests filter by that ID. There is no global reset during a run.
- Tests that change guild-wide state (for example a Raid-Helper event replacing what the raids page shows) go in a serial `describe` block.

If isolation proves hard for a journey, that spec runs with `workers: 1` rather than relying on the order tests run in.

### Rules for what the browser can reach

- `playwright.config.ts` blocks every request that is not to localhost. That enforces the CLAUDE.md rule in the browser too, which matters because the pages load third-party content directly: Discord avatars, Twitch and YouTube embeds, and the Turnstile script.
- The Turnstile script request is answered with a stub that defines `window.turnstile` and passes a fixed token, following the shape of `frontend/src/mocks/fakeTurnstile.ts`. The API then verifies that token against the fake siteverify.
- Embeds and avatars are allowed to fail. Tests assert on our markup (the iframe's `src`, the fallback avatar), not on third-party content.

### Personas and the first-admin rule

`auth/repo.go` makes the first account ever created the admin. Every run starts from an empty database, so `auth.setup.ts` (a Playwright setup project the test projects depend on) logs in the admin persona first and saves its storage state, then does the same for the officer, member and not-in-guild personas. A setup project rather than `globalSetup` gets traces and the HTML report when login fails. Tests reuse the saved states without depending on test order.

### Sync jobs

The roster, professions, Twitch and YouTube syncs are `cmd/sync` jobs, not API endpoints. A test helper runs the sync binary built in global setup (`sync <job>`) with the E2E environment. Tests trigger syncs the way Render's cron does, without adding a test-only endpoint to the API.

### Journeys

project-structure.md keeps this layer to "a handful of critical user journeys". Each journey below is a single spec file and covers the main path plus its one most important failure. Everything else stays in the Go and Vitest layers.

Integrations first:

1. **Discord login**: each role lands with the right navigation, and a user outside the guild is refused.
2. **Battle.net link**: a member links an account, then after the roster sync their characters appear on "my characters".
3. **Application**: submitted through Turnstile, it shows on the officer review page, and the fake recruiting webhook receives the post.
4. **Raid-Helper**: a signed webhook for create, then update, then delete is reflected on the raids page each time. The payloads come from `backend/internal/raidhelper/testdata`.
5. **Rank request**: a member submits one, an officer sees it, and the rank-requests webhook receives it.
6. **Streams**: after a Twitch or YouTube sync, a live stream shows on the guild home page and on the hub.

Then the basics: the news post page (and its link-preview HTML from `/api/news/:id/page`), both Legal pages, and the 404 page on both sites.

Test names follow the CLAUDE.md pattern, for example `test("application should post to the recruiting webhook when an applicant submits the form")`.

## 2. `qa-this` skill

`.claude/skills/qa-this/SKILL.md` is triggered by "qa this", "qa PR 212", "qa my branch" or "QA these changes".

**Input:** a PR number, or nothing (the current branch or working tree).

1. **Workspace.** For a PR, check it out into a git worktree (`.claude/worktrees/pr-N`) so the user's working tree is never touched. For the current branch, use the working tree as it is.
2. **Review.** Run `review-pr` unchanged. Its four reviewers and its verdict rules stay as they are.
3. **Automated suites.** Run `npm test` (Go and both Vitest suites), then `npm run test:e2e`. Map the changed files to spec files (a frontend page to its journey, a backend package to the journeys that call it) and run those first. If they pass, run the full suite, which is small enough to always finish.
4. **Visual check.** Collect the routes the diff touches on the guild site and the hub, and load each one in the E2E stack as every relevant persona at 1280px and 390px wide. For each page:
   - Record a video of the page loading and of each interaction the change touches (opening a menu, submitting a form), for the person reading the report.
   - Save a full-page screenshot of each state the video shows, and read the screenshots. Claude can read images but not video, so the screenshots are what the visual findings are based on.
   - Run `@axe-core/playwright` and report serious and critical violations.
   - Report console errors and failed same-origin requests.
   - Check it against the type scale, colors and contrast rules in `plans/design.md`.
   Claude reading screenshots is exploratory QA, not a regression test. It produces review findings, never test failures. Pixel-baseline comparison (`toHaveScreenshot`) is left to Phase 2, because baselines taken on Windows don't match Linux CI rendering.
   Findings use `review-pr`'s `severity | file:line | problem | why | fix` shape. For a finding with no line to point to, use the route and persona instead.
5. **Report.** Merge everything into one report in `review-pr`'s format: verdict, findings, test results with counts, and the videos and screenshots saved under the run folder. The verdict rules gain one item: any failing test is a `must`.
6. **Offer to fix.** End with "Want me to fix these?" If the user says yes:
   - Fix every `must` and `should`, but not `nit`s.
   - Rerun steps 3 and 4 on what changed.
   - Stop after two passes and report what is left, matching the `issue-workflow.md` limit.
   - Ask before committing and again before pushing, as CLAUDE.md requires.
   - On someone else's PR from a fork, pushing works only if "Allow edits by maintainers" is ticked. Otherwise, offer the fixes as draft suggested-change comments, posted only when asked.

**Run record.** Each run saves its report and screenshots in a run folder under `.claude/qa-runs/`. Both `.claude/qa-runs/` and the `.claude/worktrees/` folder from step 1 are gitignored. The report notes the `HEAD` commit it checked and a hash of `git diff HEAD` plus the contents of untracked files (`git ls-files --others --exclude-standard`), because `git diff` alone leaves out new files. That way a later session can tell whether the current changes have already been checked.

**Changes to existing skills:**
- `issue-workflow.md` step 7 (verify) also runs steps 3 and 4 of `qa-this` for the pages and journeys the issue touches. Step 8 is unchanged and still runs `review-pr`, so issue work is not reviewed twice.
- `review-pr` is not changed.

### Prompt to run "qa this" before committing or pushing

A new rule in CLAUDE.md's Git workflow section: before Claude commits or pushes, it checks whether `qa-this` has run on the current changes. That means a run record matches the current `HEAD` and diff hash, or for a push, matches the commit being pushed. If no run matches, Claude asks the user whether to run "qa this" first, and says what has changed since the last run, if there was one.

- The rule prompts the user but doesn't block. If the user says to go ahead, Claude commits or pushes without running it.
- Docs-only changes (only `plans/`, `*.md` or Legal wording) skip the prompt, because the checks would test nothing.
- The prompt is part of the existing "ask before committing" and "ask before pushing" questions, so the user still answers one question, not two.
- On a fork PR Claude can't push to (no "Allow edits by maintainers"), Claude doesn't commit to or push that branch, so the prompt never fires. The fixes go out as suggested-change comments instead (step 6), and those are posted only when the user asks.
- Issue work through `next-issue` already runs the suites and the visual check in step 7, so its run record usually matches and the prompt doesn't fire.

## Build order

Phase 1. Each step is one stacked PR, per the stacked-PR rules in CLAUDE.md.

1. **`test(e2e): add Playwright harness with provider fakes`**
   - `docker-compose.e2e.yml`, `API_PROXY_TARGET`, the provider base-URL variables (with config tests for each default and override), `e2e/` with the fakes, global setup and journey 1.
   - The `test:e2e` script, and the postinstall change.
   - The README "Running the tests" and Scripts tables.
   - project-structure.md's E2E section updated to the layout above.
   - Global setup builds the `cmd/sync` binary, but no journey runs it until PR 2. PR 1 only proves the build works.
   - This is the largest PR. If it gets too big to review, the fakes server and its fixtures split into their own PR below it, and the rest stacks on top.
2. **`test(e2e): cover integration journeys`**: journeys 2 to 6 and the basics.
3. **`chore(skills): add qa-this skill`**: the skill and its run record, the `.gitignore` entries for `.claude/qa-runs/` and `.claude/worktrees/`, the `issue-workflow.md` step 7 change, and the CLAUDE.md rule that prompts the user to run "qa this" before committing or pushing.

The Legal pages do not change: no new data is collected and no new third party is involved.

## Phase 2: move to CI

Phase 2 starts once the Phase 1 suite has run reliably on local work. It brings the same checks into GitHub Actions, so guild devs' PRs get them automatically:

1. **Playwright in `test.yml`.** A job starts the same throwaway stack (`docker-compose.e2e.yml`, the TypeScript fakes, the built sites) and runs `npm run test:e2e` on every PR. It becomes a required check, which completes the CI plan in project-structure.md. This job has no Claude cost.
2. **Automatic Claude review.** A workflow using `anthropics/claude-code-action` runs `review-pr` when a PR is opened or marked ready for review, and posts its findings as PR comments. `review-pr` gains a CI mode that is allowed to post to GitHub.
3. **The visual check in CI.** The `qa-this` visual step runs against the CI stack, and its videos and screenshots are uploaded as workflow artifacts.
4. **Lint.** ESLint for both frontends and `golangci-lint` for the backend, as CI checks. Type checking already runs in CI through each site's `npm run build` (`tsc --noEmit`), and `go test` already runs `go vet`'s default checks.
5. **Visual regression baselines.** `toHaveScreenshot` baselines for the key pages, generated and compared only in CI's Linux container.
6. **Post-deploy smoke check.** A small read-only Playwright run against the live guild site and hub after Render deploys `main`, with no login and no writes.
7. **Fixes on request.** A `@claude fix` comment from someone with write access runs the fix step on the PR branch. Commits are added on top of the branch, never rebased, so stacked PRs keep working. CLAUDE.md gains a CI section saying that the comment counts as being told to commit.

Phase 2 needs decisions that Phase 1 doesn't: how Claude authenticates in Actions (an API key with a spend limit, or a subscription token), a cost cap per run, and how to handle PRs from forks, which get no secrets. It gets its own plan and issues when it starts.

## Decisions to confirm in review

1. **Fakes in TypeScript in `e2e/fakes`**, not a refactor of `internal/testutil` into a shared Go server. TypeScript keeps the backend untouched apart from the base-URL variables. The cost is that the Go and E2E fakes can drift apart. The `blizzard` and `raidhelper` fixtures are shared and can't drift. The Discord, Battle.net, YouTube and Turnstile fixtures are copies of the inline responses in `internal/testutil`, and Twitch's are new, so a change to one of those providers' responses has to be made in both places.
2. **Built frontend through `vite preview`** rather than the dev server, as project-structure.md describes. It is slower to start but tests what ships.
3. **Separate ports and a throwaway database**, so QA can run while `npm run dev` is up.
4. **Playwright stays out of CI until Phase 2.** Until then the repo doesn't match project-structure.md's plan to make it a required check.

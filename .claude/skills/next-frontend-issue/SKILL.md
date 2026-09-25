---
name: next-frontend-issue
description: Use when the user asks to find, pick up, or work on the next frontend issue in the Fuzion repo, or says "next frontend issue".
---

# Next frontend issue

Follow [../_shared/issue-workflow.md](../_shared/issue-workflow.md) end to end. This file supplies the frontend-specific parts.

## Pick filter (step 2)

Issues labeled `frontend`. Issues labeled both `backend` and `frontend` are fine only if the backend half is already merged. In addition to the shared blocker rules, skip any frontend issue whose API endpoint is not on `main` yet, and name the issue or PR that provides it (for example "skipped #12, needs GET /api/raids, PR #44 open"). Check `backend/internal/*/handler.go` and `backend/README.md` on `main` to confirm an endpoint exists.

## Implementation notes (step 6)

- React 18, TypeScript, Vite, TanStack Query, react-router-dom, Vitest, RTL, MSW. Structure: `plans/project-structure.md`. Look at existing pages and `src/api/` modules and match them.
- Before writing, read the `Home` page, its data hook (`useHomeData.ts`), and its test, plus the matching `src/api/` module and `mocks/handlers.ts`. Reuse existing components (`Chip`, `NewsCard`, `LiveStreamCard`, and so on) before creating new ones.
- React checklist for the self-review: server data goes through TanStack Query (no ad hoc `useEffect` fetching), types live in `src/types/`, no `any`, tests query by role or label (`getByRole`) and use `userEvent`, and loading, empty, and error states are all rendered and tested.
- Page tests render the full page with real router context, real children and real hooks, and intercept only HTTP with MSW. Never mock hooks, contexts or child components.
- MSW handlers default to the happy path in `mocks/handlers.ts`; individual tests override with `server.use(...)`.
- Test names follow: `<feature> should <expected behavior> when <criteria>`.

## Verify (step 7)

1. Automated, from `frontend/`: `npm run typecheck` then `npm test`. Run `npm install` first if `node_modules` is missing.
2. In the real app, when the page depends on a merged endpoint:
   - Start Postgres and the API as described in the backend skill's verify step (`docker compose up -d db`, run the API from `backend/`, seed with `seed/dev_seed.sql`).
   - Start `npm run dev` in `frontend/` (Vite proxies `/api` to the backend) and open the page at `http://localhost:5173`. Use the `run` skill or the browser tools if available to load the page and check that it renders real data, the filters and interactions in the issue's acceptance criteria work, and the empty and error states look right.
   - For officer-only UI, create an officer session as in the backend skill and set the `fuzion_session` cookie in the browser.
   - Stop the dev servers afterwards.
3. If no browser tooling is available, say the in-browser check was not done and give the URL and steps for the user to check.
4. Report anything that could not be checked. Do not claim the page works based on the test suite alone without saying the in-app check did not run.

Put a short summary of what was checked in the app (page, filters tried, states seen) in the PR body.

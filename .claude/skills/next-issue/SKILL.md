---
name: next-issue
description: Use when the user asks to find, pick up, or work on the next issue in the Fuzion repo, says "next issue", or picks an issue that needs both backend and frontend changes.
---

# Next issue

Follow [../_shared/issue-workflow.md](../_shared/issue-workflow.md) end to end. This file supplies the scope-dependent parts by reusing the sections of [next-backend-issue](../next-backend-issue/SKILL.md) and [next-frontend-issue](../next-frontend-issue/SKILL.md). Keep those headings ("Implementation notes", "Verify") in place; this skill points at them.

## Pick filter (step 2)

Issues labeled `backend` and/or `frontend`. Do not apply the frontend skill's "backend must already be on `main`" rule: for an issue that needs both, the backend half is built in this same branch. A frontend-only issue still needs its endpoint on `main`.

## Scope (part of step 3)

Decide `backend`, `frontend`, or `both` from the labels, then check the issue body (endpoints named, pages or components named). If they disagree, propose the body's scope and let the user decide. State the scope in the pick rationale and get it confirmed together with the pick, before claiming the issue in step 4.

## Implementation (step 6)

Apply the "Implementation notes" section of each skill that matches the scope.

- `backend` or `frontend`: that skill's notes only.
- `both`: one branch, one PR. Backend first (tests, then code), then frontend against the real endpoint. Do not start the frontend until the backend tests pass.
- Touch `hub/` only when the issue says so, using the frontend notes.

## Verify (step 7)

Run the "Verify" section of each matching skill.

- `both`: start Postgres and the API once, exercise the endpoints with curl, then start `npm run dev` in `frontend/` and check the page against that same API.
- If the backend check fails, stop and report. Do not run the frontend check against an endpoint that did not pass.
- Report each side's evidence separately in the PR body.

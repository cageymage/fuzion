# Issue workflow (shared by next-backend-issue and next-frontend-issue)

Follow these steps in order. The calling skill supplies the **pick filter** (step 2) and the **verify** step (step 6).

## Hard rules from CLAUDE.md

- Never `git commit` without asking first in this run. Never `git push` or open a PR without asking first.
- Commit subjects and PR titles use Conventional Commits prefixes (`feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `ci`), with an optional scope.
- Follow the test naming, given/when/then, and sociable-test rules in CLAUDE.md. Never mock Postgres or internal packages.
- No em dashes in generated text.

## 1. Gather

```
gh issue list --state open --limit 100 --json number,title,labels,body,assignees
gh pr list --state open --json number,title,headRefName,body
```

## 2. Filter and report blockers (always print this before picking)

Keep only issues matching the calling skill's pick filter. Skip and report each of these:

- **Blocked**: the body has a `Blocked by: #N, #M` line and any listed issue is still open. If the blocker has an open PR, say so ("waiting on #23, PR #43 open"). A blocker whose PR is open but unmerged still counts as blocked.
- **Taken**: has an assignee, or an open PR whose branch starts with the issue number or whose body says `Closes #N`.
- **Not work**: labeled `decision`, `documentation`, or `risk: legendary` (ask the user before touching legendary).

Print a compact list: `skipped #25 (blocked by #23, PR #43 open)`. Also mention soft dependencies, for example an issue that says to reuse a package from a blocked issue.

## 3. Pick

- Choose the best remaining issue. Prefer ones that set patterns later issues reuse, then `good first issue`, then lower `risk:` rating.
- If nothing is unblocked, stop and say so, listing what is blocking each. Do not pick a blocked issue to be helpful.
- Present the pick with a two-line rationale and ask the user to confirm before doing anything that is visible to others.

## 4. Claim (after confirmation)

```
gh issue edit N --add-assignee @me
```

## 5. Branch

- Read the issue body fully, plus `plans/project-structure.md` and any spec section it links.
- Start from up-to-date `main`: `git fetch origin && git switch main && git pull --ff-only`.
- If the working tree is dirty, stop and ask (or use a worktree with the user's OK).
- Create `N-short-slug` (existing examples: `21-middleware`, `11-raids-list-api`).
- If the issue adds a migration, check the highest number on `main` and in open PRs (`git ls-tree origin/main backend/migrations/`, `gh pr diff`). CI fails a PR numbered at or below main's highest. The number in the issue body is a suggestion; use the next free one and say if it changed.

## 6. Clarify, then implement

**Ask before writing any code** if anything is unclear or underspecified: ambiguous requirements, conflicting statements between the issue and the spec, an open question the issue itself flags (for example "ask in the issue"), a JSON shape or status code the issue does not pin down, or a soft dependency on unmerged work. Batch the questions into one message with a recommended answer for each. Do not guess and proceed. If everything is clear, say so in one line and continue.

Then read one existing, comparable feature (the calling skill names where) and match its file layout, naming, and idiom before writing anything new.

Use the issue's "What to build", "Acceptance criteria" and "Tests to write" sections as the spec. Write the tests first (test-driven), watch them fail, then implement. Use the test names the issue lists. Keep changes inside the scope of the issue.

## 7. Self-review, then verify

Reread your own diff against CLAUDE.md before verifying, and fix what you find:

- No comments that explain what the code does (only non-obvious why).
- No speculative helpers, interfaces, or config for hypothetical needs.
- No mocks of Postgres, hooks, or internal packages.
- Errors are wrapped with context and never swallowed.
- Test names follow `<feature> should <expected behavior> when <criteria>`, one "when" per test.
- Nothing outside the issue's scope changed.

Then run the calling skill's verify step. Report results faithfully, including failures. If anything fails, fix it or stop and report; do not proceed to commit on a failing or skipped check without saying so.

## 8. Ask before committing

Show `git status`, a diff summary, and the verification output. Ask: "Commit this?" Only on yes:

- Subject: `<type>(scope): <summary>`, for example `feat(roster): Add officer create/update/delete endpoints`.
- End the message with the attribution line from the session's system reminder (`Co-Authored-By: ...`).
- Do not skip hooks. Do not amend.

## 9. Ask before pushing and opening the PR

Ask: "Push and open the PR?" Only on yes:

- `git push -u origin <branch>`
- Title: `<type>(scope): Issue #N - <short summary>` (matches existing PRs, for example `feat(raids): Issue #11 - Add GET /api/raids to list upcoming raids`).
- Body: `Closes #N`, a short summary of what changed, what was verified (include the actual request and response evidence, or say the step was skipped and why), any migration number change, and any follow-ups or soft dependencies. End with the `Generated with Claude Code` line from the session's system reminder.
- Return the PR URL.

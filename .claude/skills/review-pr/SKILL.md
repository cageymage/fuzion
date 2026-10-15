---
name: review-pr
description: Use when the user asks to review a pull request, a branch, or a diff in the Fuzion repo, for example "review PR 123", "review my branch", "code review this", "check this before I push", or "is this ready to merge". Checks coding standards, security, performance, PR/commit naming, and whether the linked GitHub issue's requirements are met, then offers draft PR comments with suggested changes. Also applies when the user only asks to look over their changes.
---

# Review PR

Review a change against Fuzion's standards and report short, plain findings that a junior developer can act on. This skill is **read-only**: never commit, push, approve, or post a comment to GitHub. Posting is a separate step the user must ask for.

**Input:** a PR number, or nothing (the current branch).

## 1. Gather context once

Run `bash .claude/skills/review-pr/scripts/gather.sh [PR_NUMBER]`. It prints a folder containing `pr.json` (title, body, state, base, commits, files), `diff.patch`, `files.txt`, and `issue-N.json` for each linked issue (from `closingIssuesReferences` or a `Closes #N` line). With no PR number it diffs the current branch against its base, and says in `NOTE.txt` when only uncommitted changes exist (then there is no PR or commit metadata to check). Also read any issue named in a `Blocked by` line.

Pass the folder path to subagents instead of pasting the diff into four prompts. `CLAUDE.md` is already in context; read `plans/project-structure.md` only if file placement is in question.

## 2. Fan out four reviewers in parallel

Spawn all four in **one message** so they run concurrently. Give each the folder path and tell it to read its reference file first. Ask each to return only findings in this shape, nothing else:

`severity | file:line | problem (one sentence) | why it matters (one sentence) | suggested fix`

Severity is `must` (bug, security hole, broken requirement, rule in CLAUDE.md violated), `should` (likely to cause trouble or confuse readers), or `nit` (taste). Tell them to skip anything CI or a linter already enforces unless it is missing, and to say "no findings" rather than inventing some.

Tell them how to report lines: use the line in the **new version** of the file, open the file to confirm it, and write `unverified` if they could not. Wrong line numbers make the draft comments useless.

| Reviewer | Focus | Reference |
|---|---|---|
| Standards | CLAUDE.md rules, test style, Legal page, migrations | `references/standards-checklist.md` |
| Security | auth, injection, input validation, uploads, secrets, XSS | `references/security-performance.md` (Security section) |
| Performance | queries, indexes, unbounded lists, batch failures, FE refetching | `references/security-performance.md` (Performance section) |
| Requirements | each acceptance criterion and "Tests to write" item vs the diff | the `issue-N.json` files only |

The Requirements reviewer reports each criterion as `met`, `partial`, `missing`, or `unclear`, with the file that satisfies it, and separately lists changes that are **out of scope** for the issue. If the PR links no issue, that is one `should` finding.

## 3. Check naming and metadata yourself

This is quick, so do it inline rather than spawning an agent:

- **PR title**: `<type>(scope): Issue #N - <short summary>`, with type one of `feat fix docs chore refactor test ci`. Decision tickets use `docs: Issue #N - <summary>`. Chore/ci PRs with no issue may omit `Issue #N`.
- **Commit subjects**: each starts with a Conventional Commits type, scope optional.
- **Branch name**: `N-short-slug` for issue work.
- **PR body**: has `Closes #N`; starts with `Stacked on #M` when the base is not `main`; says what was verified; mentions a changed migration number or a Legal page change if relevant.
- **Trailers**: flag `Co-Authored-By` or Claude attribution lines in commits or the PR body as `should`, not `must`. The repo's own issue workflow still tells Claude to add them, so they are common in merged history. Also flag em dashes in the text.

## 4. Merge and report

Remove duplicates (two reviewers often flag the same line), drop `nit`s if there are more than five `must`/`should` items, and open the file at each `file:line` that a reviewer marked `unverified`. Never guess a line number.

Pick the verdict:

- **Ready to merge**: no `must` and at most two `should`.
- **Changes needed**: any `must`, or three or more `should`.
- **Blocking issues**: a security `must`, or a core requirement of the issue is missing.

Write the report in the layout from [references/report-format.md](references/report-format.md), including its rule for already-merged PRs.

## 5. Offer draft PR comments

End with one line: "Want draft PR comments for these?" Do not print them unprompted. If the user says yes, write them in the format from [references/report-format.md](references/report-format.md). Post to GitHub only if the user then asks.

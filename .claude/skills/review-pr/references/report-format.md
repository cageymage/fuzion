# Report and draft-comment format

Read this at the report step (step 5) and when the user asks for draft comments (step 6).

## Report layout

Use exactly this layout:

```
## Verdict: <Ready to merge | Changes needed | Blocking issues>
<one sentence on why>

| Area | Status | Must / Should / Nit |
|---|---|---|
| Standards | ... | 0 / 2 / 1 |
| Security | ... | ... |
| Performance | ... | ... |
| Requirements | 4 of 5 met | ... |
| PR and commit naming | ... | ... |

## Must fix
1. `path/file.go:42` Problem in plain words.
   Why: one sentence a junior dev would understand.
   Fix: what to change.

## Should fix
...

## Nits
...

## Requirements from #N
- [x] criterion (file)
- [ ] criterion: missing
Out of scope: ...

## Not checked
<e.g. "Tests were not run", "Did not start the API">
```

If the PR is already merged, rename the three headings to `Follow-ups (high)`, `Follow-ups (medium)`, and `Follow-ups (low)`, and say "already merged" in the verdict sentence. Nothing is left to block, so the verdict then reads as how the change would have fared.

Writing rules for the findings: short sentences, plain words, explain the *why* (the consequence if left alone), and name the exact fix. Say a section is clean when it is clean. Do not run tests or the app unless the user asked; list that under "Not checked".

## Draft PR comments

One draft per finding the user wants (default: all `must` and `should`):

````
**`path/file.go:42`**
Short, friendly comment that states the problem and why.

```suggestion
the replacement line(s)
```
````

Include a `suggestion` block only when the fix is a concrete replacement of the lines at that location. For a PR body, a commit message, or a change that spans several places, describe the change in words. Keep the tone constructive and written as one developer to another.

## Posting

Only when the user asks. Confirm first, because comments are visible to others. Submit them as a single **pending** review (`gh api` with `event` omitted) so the user presses submit themselves.

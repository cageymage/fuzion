#!/usr/bin/env bash
# Collect everything a review needs into one folder and print the folder path.
# Usage: gather.sh [PR_NUMBER] [OUT_DIR]
# Read-only: only runs gh/git commands that fetch data.
set -euo pipefail

pr="${1:-}"
out="${2:-$(mktemp -d)}"
mkdir -p "$out"

fields="number,title,body,state,baseRefName,headRefName,commits,files,closingIssuesReferences"

if [ -n "$pr" ]; then
  gh pr view "$pr" --json "$fields" > "$out/pr.json"
  gh pr diff "$pr" > "$out/diff.patch"
else
  if gh pr view --json "$fields" > "$out/pr.json" 2>/dev/null; then
    base=$(jq -r .baseRefName "$out/pr.json")
  else
    echo '{}' > "$out/pr.json"
    base=main
  fi
  git fetch origin "$base" --quiet || true
  git diff "origin/$base...HEAD" > "$out/diff.patch"
  git log "origin/$base..HEAD" --format=%s > "$out/commit-subjects.txt"
  if [ ! -s "$out/diff.patch" ]; then
    git diff HEAD > "$out/diff.patch"
    echo "no commits ahead of $base; diff is uncommitted changes only" > "$out/NOTE.txt"
  fi
fi

grep -E '^\+\+\+ b/' "$out/diff.patch" | sed 's#^+++ b/##' > "$out/files.txt" || true

body=$(jq -r '.body // ""' "$out/pr.json")
issues=$( { jq -r '.closingIssuesReferences[]?.number' "$out/pr.json"; \
            printf '%s' "$body" | grep -oiE '(closes|fixes|resolves) #[0-9]+' | grep -oE '[0-9]+'; } | sort -un )

for n in $issues; do
  gh issue view "$n" --json number,title,body,labels,comments > "$out/issue-$n.json"
done

echo "$out"

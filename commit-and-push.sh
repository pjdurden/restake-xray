#!/usr/bin/env bash
# Commit the restake-xray work and push to GitHub.
#
# On first run this creates the GitHub repo for you via the GitHub CLI (`gh`)
# and wires up `origin` — no manual repo creation needed. The commit is made
# with your local git identity / signing key.
#
# Usage:
#   ./commit-and-push.sh                 # commit, create repo via gh if needed, push
#   ./commit-and-push.sh --no-push       # commit only
#   ./commit-and-push.sh --private       # if the repo doesn't exist yet, create it private
#   ./commit-and-push.sh --public        # (default) create it public
#
# Env overrides:
#   REPO_NAME=foo                        repo name (default: this directory's name)
#   VISIBILITY=private                   public|private (default: public)
#   REMOTE_URL=git@github.com:you/x.git  use this exact remote instead of `gh repo create`

set -euo pipefail
cd "$(dirname "$0")"

COMMIT_MSG="Add restake-xray design spec

Open-source restaking/LRT exposure X-ray engine (Go). Exposure graph
(collateral -> LRT -> operator -> AVS), EigenLayer-deep v1 with
protocol-agnostic adapters, hybrid on-chain + off-chain data, CLI +
library + git-committed dataset + hosted API + static demo.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"

# --- args ---
DO_PUSH=1
VISIBILITY="${VISIBILITY:-public}"
for arg in "$@"; do
  case "$arg" in
    --no-push) DO_PUSH=0 ;;
    --private) VISIBILITY=private ;;
    --public)  VISIBILITY=public ;;
    *) echo ">> unknown arg: $arg" >&2; exit 2 ;;
  esac
done
REPO_NAME="${REPO_NAME:-$(basename "$PWD")}"

# --- ensure git repo + a commit ---
git rev-parse --git-dir >/dev/null 2>&1 || git init

git add -A
if git diff --cached --quiet; then
  echo ">> nothing staged to commit"
  git rev-parse HEAD >/dev/null 2>&1 || { echo ">> no commits and nothing to commit; aborting" >&2; exit 1; }
else
  echo ">> committing (signed with your key)..."
  git commit -m "$COMMIT_MSG"
fi

# normalize default branch to main
branch="$(git rev-parse --abbrev-ref HEAD)"
if [[ "$branch" == "master" ]]; then
  git branch -M main
  branch=main
fi

if [[ "$DO_PUSH" == 0 ]]; then
  echo ">> --no-push set; skipping push"
  exit 0
fi

# --- ensure an 'origin' remote (create the repo via gh if missing) ---
if ! git remote get-url origin >/dev/null 2>&1; then
  if [[ -n "${REMOTE_URL:-}" ]]; then
    echo ">> adding origin: $REMOTE_URL"
    git remote add origin "$REMOTE_URL"
  elif command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1; then
    owner="$(gh api user -q .login)"
    echo ">> creating GitHub repo $owner/$REPO_NAME ($VISIBILITY) via gh..."
    gh repo create "$REPO_NAME" "--$VISIBILITY" --source . --remote origin
  else
    echo ">> no 'origin', no REMOTE_URL, and gh is not authenticated." >&2
    echo "   Fix one of these, then re-run:" >&2
    echo "     gh auth login" >&2
    echo "     REMOTE_URL=git@github.com:<you>/$REPO_NAME.git ./commit-and-push.sh" >&2
    exit 1
  fi
fi

echo ">> pushing $branch to origin..."
git push -u origin "$branch"
echo ">> done: $(git remote get-url origin)"

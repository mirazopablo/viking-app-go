#!/usr/bin/env bash

# Exit immediately if a command exits with a non-zero status
set -e

PROTECTED_BRANCHES=("main" "master" "development" "develop" "staging" "release")
MAIN_BRANCH="main"

# Function to check if a branch is protected
is_protected_branch() {
  local branch="$1"
  for protected in "${PROTECTED_BRANCHES[@]}"; do
    if [ "$branch" == "$protected" ]; then
      return 0
    fi
  done
  return 1
}

# Usage helper
show_usage() {
  local exit_code="${1:-1}"
  echo "Usage:"
  echo "  Create new branch from the latest commit on main (preserves uncommitted changes):"
  echo "    ./create_branch.sh <new-branch-name>"
  echo ""
  echo "  Rename current branch:"
  echo "    ./create_branch.sh -m <new-branch-name>"
  echo "    ./create_branch.sh --rename <new-branch-name>"
  echo ""
  echo "  Show this help message:"
  echo "    ./create_branch.sh -h | --help"
  exit "$exit_code"
}

if [ -z "$1" ]; then
  show_usage 1
fi

# Handle Help Mode (-h, --help, help)
if [ "$1" == "-h" ] || [ "$1" == "--help" ] || [ "$1" == "help" ]; then
  show_usage 0
fi

# Handle Rename Mode (-m or --rename)
if [ "$1" == "-m" ] || [ "$1" == "--rename" ]; then
  NEW_BRANCH_NAME="$2"
  if [ -z "$NEW_BRANCH_NAME" ]; then
    echo "Error: New branch name is required for rename mode."
    show_usage 1
  fi

  CURRENT_BRANCH=$(git branch --show-current)

  if [ -z "$CURRENT_BRANCH" ]; then
    echo "Error: Not currently on any branch (detached HEAD)."
    exit 1
  fi

  if is_protected_branch "$CURRENT_BRANCH"; then
    echo "Error: Cannot rename protected branch '$CURRENT_BRANCH'."
    exit 1
  fi

  echo "Renaming branch '$CURRENT_BRANCH' to '$NEW_BRANCH_NAME'..."
  git branch -m "$NEW_BRANCH_NAME"

  echo "Successfully renamed local branch from '$CURRENT_BRANCH' to '$NEW_BRANCH_NAME'."
  echo "Tip: If you already pushed '$CURRENT_BRANCH' to remote, run:"
  echo "  git push origin -u $NEW_BRANCH_NAME"
  echo "  git push origin --delete $CURRENT_BRANCH"
  exit 0
fi

# Create Mode (default)
BRANCH_NAME="$1"

# Validate that branch name does not start with a dash (prevents accidental flags as names)
if [[ "$BRANCH_NAME" == -* ]]; then
  echo "Error: Unknown option or invalid branch name '$BRANCH_NAME'."
  show_usage 1
fi

STASHED=false
restore_stash_on_error() {
  if [ "$STASHED" = true ]; then
    echo "An error occurred. Restoring stashed changes to your original branch..."
    git stash pop || echo "Warning: Could not auto-pop stash. Run 'git stash pop' manually."
  fi
}
trap restore_stash_on_error ERR

# 1. Detect if working directory is dirty (tracked or untracked changes)
if [ -n "$(git status --porcelain)" ]; then
  echo "Detected uncommitted changes in current working tree."
  echo "Stashing local changes (including untracked files)..."
  git stash push --include-untracked -m "auto-stash-create-branch: $BRANCH_NAME"
  STASHED=true
fi

# 2. Fetch latest changes from origin/main
echo "Fetching latest changes from origin/${MAIN_BRANCH}..."
git fetch origin "$MAIN_BRANCH"

# 3. Always pick the newest commit between local main and origin/main
HAS_LOCAL_MAIN=false
HAS_REMOTE_MAIN=false

if git show-ref --verify --quiet "refs/heads/${MAIN_BRANCH}"; then
  HAS_LOCAL_MAIN=true
fi
if git show-ref --verify --quiet "refs/remotes/origin/${MAIN_BRANCH}"; then
  HAS_REMOTE_MAIN=true
fi

if [ "$HAS_LOCAL_MAIN" = true ] && [ "$HAS_REMOTE_MAIN" = true ]; then
  if git merge-base --is-ancestor "origin/${MAIN_BRANCH}" "$MAIN_BRANCH"; then
    # Local main is equal to or ahead of origin/main
    TARGET_BASE="$MAIN_BRANCH"
  elif git merge-base --is-ancestor "$MAIN_BRANCH" "origin/${MAIN_BRANCH}"; then
    # Remote origin/main is ahead of local main
    TARGET_BASE="origin/${MAIN_BRANCH}"
  else
    echo "Error: Local '${MAIN_BRANCH}' and 'origin/${MAIN_BRANCH}' have diverged."
    echo "Please reconcile '${MAIN_BRANCH}' (rebase or merge) before creating a new branch."
    false
  fi
elif [ "$HAS_REMOTE_MAIN" = true ]; then
  TARGET_BASE="origin/${MAIN_BRANCH}"
elif [ "$HAS_LOCAL_MAIN" = true ]; then
  TARGET_BASE="$MAIN_BRANCH"
else
  echo "Error: Neither '${MAIN_BRANCH}' nor 'origin/${MAIN_BRANCH}' exists."
  false
fi

BASE_COMMIT=$(git rev-parse --short "$TARGET_BASE")
echo "Creating and checking out branch '$BRANCH_NAME' from '$TARGET_BASE' ($BASE_COMMIT)..."
git checkout -b "$BRANCH_NAME" "$TARGET_BASE"

# Clear ERR trap before popping stash (conflicts are handled explicitly below)
trap - ERR

# 4. Restore stashed changes if any were saved
if [ "$STASHED" = true ]; then
  echo "Restoring your uncommitted changes into '$BRANCH_NAME'..."

  # Temporarily disable 'set -e' because git stash pop returns non-zero on merge conflict
  set +e
  git stash pop
  POP_STATUS=$?
  set -e

  if [ $POP_STATUS -ne 0 ]; then
    echo ""
    echo "====================================================================="
    echo "WARNING: Conflicts occurred while applying your local changes."
    echo "Your stash was preserved in Git's stash list."
    echo "Please resolve the conflict markers in your files, stage them, and commit."
    echo "====================================================================="
    exit 1
  fi

  echo "Changes restored cleanly. Ready to stage and commit."
fi

echo "Successfully created and switched to branch '$BRANCH_NAME' (base: $TARGET_BASE @ $BASE_COMMIT)."

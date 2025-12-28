#!/usr/bin/env bash

set -e

usage() {
    echo "Usage: $0 [--json] <branch-name>"
    echo ""
    echo "Creates a git worktree for the specified branch."
    echo ""
    echo "Options:"
    echo "  --json          Output in JSON format"
    echo "  --help, -h      Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 001-user-auth"
    echo "  $0 --json 002-feature-name"
    exit 0
}

JSON_MODE=false
BRANCH_NAME=""

while [ $# -gt 0 ]; do
    case "$1" in
        --json)
            JSON_MODE=true
            ;;
        --help|-h)
            usage
            ;;
        -*)
            echo "Error: Unknown option: $1" >&2
            exit 1
            ;;
        *)
            if [ -z "$BRANCH_NAME" ]; then
                BRANCH_NAME="$1"
            else
                echo "Error: Multiple branch names provided" >&2
                exit 1
            fi
            ;;
    esac
    shift
done

if [ -z "$BRANCH_NAME" ]; then
    echo "Error: Branch name required" >&2
    usage
fi

# Get repository root
if ! git rev-parse --show-toplevel >/dev/null 2>&1; then
    echo "Error: Not in a git repository" >&2
    exit 1
fi

REPO_ROOT=$(git rev-parse --show-toplevel)
REPO_NAME=$(basename "$REPO_ROOT")

# Worktree base directory
WORKTREE_BASE="$HOME/.worktrees/$REPO_NAME"
WORKTREE_PATH="$WORKTREE_BASE/$BRANCH_NAME"

# Check if branch exists
if ! git rev-parse --verify "$BRANCH_NAME" >/dev/null 2>&1; then
    echo "Error: Branch '$BRANCH_NAME' does not exist" >&2
    echo "Create it first with: git checkout -b $BRANCH_NAME" >&2
    exit 1
fi

# Check if worktree already exists
if [ -d "$WORKTREE_PATH" ]; then
    if $JSON_MODE; then
        printf '{"status":"exists","WORKTREE_PATH":"%s","BRANCH_NAME":"%s"}\n' "$WORKTREE_PATH" "$BRANCH_NAME"
    else
        echo "Worktree already exists at: $WORKTREE_PATH"
    fi
    exit 0
fi

# Create worktree base directory if needed
mkdir -p "$WORKTREE_BASE"

# Check if we're currently on the target branch
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" = "$BRANCH_NAME" ]; then
    # Switch to main first (worktree can't be created from the same branch)
    echo "Currently on $BRANCH_NAME, switching to main first..." >&2
    git checkout main
fi

# Create worktree
git worktree add "$WORKTREE_PATH" "$BRANCH_NAME"

if $JSON_MODE; then
    printf '{"status":"created","WORKTREE_PATH":"%s","BRANCH_NAME":"%s","command":"claude --cwd %s"}\n' "$WORKTREE_PATH" "$BRANCH_NAME" "$WORKTREE_PATH"
else
    echo ""
    echo "Worktree created successfully!"
    echo "  Path: $WORKTREE_PATH"
    echo "  Branch: $BRANCH_NAME"
    echo ""
    echo "To start working:"
    echo "  claude --cwd $WORKTREE_PATH"
    echo ""
    echo "Or open in terminal:"
    echo "  cd $WORKTREE_PATH"
fi

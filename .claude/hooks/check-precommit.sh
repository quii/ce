#!/bin/sh
set -e

hash_diff() {
	if command -v sha256sum >/dev/null 2>&1; then
		git diff --cached | sha256sum | awk '{print $1}'
	else
		git diff --cached | shasum -a 256 | awk '{print $1}'
	fi
}

MARKER=".claude/precommit-passed"

# The settings.json "if": "Bash(git commit*)" filter can't prove a compound
# command (pipes, subshells, ;-separated statements) doesn't contain a git
# commit, so the harness runs this hook for many unrelated commands too.
# Check the actual command ourselves so unrelated Bash calls aren't blocked.
COMMAND=$(cat | jq -r '.tool_input.command // empty')

if ! printf '%s' "$COMMAND" | grep -Eq '(^|[;&|(]|`) *git +commit\b'; then
	exit 0
fi

if [ ! -f "$MARKER" ]; then
	echo "No precommit check has been run yet. Run the /precommit skill before committing." >&2
	exit 2
fi

CURRENT_HASH=$(hash_diff)
MARKER_HASH=$(cat "$MARKER")

if [ "$CURRENT_HASH" != "$MARKER_HASH" ]; then
	echo "Staged changes have changed since the last precommit check. Run the /precommit skill again before committing." >&2
	exit 2
fi

exit 0

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

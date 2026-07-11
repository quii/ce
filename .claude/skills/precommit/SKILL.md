---
name: precommit
description: Runs every gate required before a commit - mechanical (mage test, mage lint, mage mutate) and the ADR check (one adr-checker subagent per relevant judgment-tier ADR) - and writes the marker a commit hook checks for. See docs/source-control.md.
---

Run before every commit, in this order:

## 1. Mechanical gates

Run `mage test`, `mage lint`, and `mage mutate` yourself, in the main conversation - these are deterministic and don't need a subagent. If any fails, stop here and fix it (or hand off to the `coder` agent to fix it) before going any further - there's no point running the ADR check against code that doesn't even pass the gates that exist to catch mistakes mechanically.

## 2. Determine which ADRs are relevant

Get the diff of what's about to be committed (staged changes against HEAD). Read every ADR's frontmatter in `docs/adr/`. Select the ones where:

- `enforcement` is `judgment` (skip `mechanical` - already covered by step 1 - and skip `process`, which isn't diff-checked at all)
- `scope` overlaps the changed files (skip ADRs whose scope doesn't touch anything in the diff)

## 3. Run the ADR check

Spawn one `adr-checker` subagent per ADR selected in step 2, in parallel, each given that ADR's path and the diff. Wait for all of them.

## 4. Handle findings

For each violation reported:

- if the fix is obvious, apply it yourself (or hand it to `coder` if it's substantial), then re-run from step 1 for whatever changed
- if the fix is not obvious, stop and describe it to the user - this is exactly the "warrants a conversation" case from `docs/source-control.md`, don't guess

## 5. Mark it clean

Once every gate passes and every applicable ADR check comes back clean, write the diff's hash to `.claude/precommit-passed` (the same hash the `check-precommit.sh` hook computes: `git diff --cached | shasum -a 256`) - this is what lets `git commit` through. Then tell the user it's ready to commit.

# Conversation Engine (CE)

See `brief.md` and `docs/` for what this is and why.

## Commands

`mage` and every tool it shells out to (`golangci-lint`, `gremlins`) are tracked via `go.mod`'s `tool` directive, not installed globally - run everything through `go tool mage <target>` so it works regardless of `PATH`/`GOBIN` setup:

- `go tool mage test` - the full test suite (`go test -race -count=3 -shuffle=on -tags=integration ./...`), including both the in-process and containerised specifications (`docs/adr/0022-specifications-and-drivers.md`) - this is the pre-commit gate
- `go tool mage testunit` - the same suite without Docker: the Postgres contract tests and container-driver specifications are excluded entirely (`docs/adr/0028-fast-and-full-test-tiers.md`) - use this while iterating, reach for `mage test` before committing
- `go tool mage lint` - `golangci-lint run --build-tags=integration ./...`
- `go tool mage mutate` - mutation testing scoped to the pending diff (`docs/adr/0020-mutation-testing.md`), only paying the Docker cost when the diff actually touches the Postgres adapter or container driver (`docs/adr/0028-fast-and-full-test-tiers.md`)

## Development flow

1. **`/story`** - facilitates an example-mapping conversation and writes the result to `stories/backlog/<name>.md` (`docs/story-process.md`). Doesn't write code.
2. **Ask for the `coder` agent** on that story once you're happy with it (e.g. "use the coder agent on stories/backlog/<name>.md"). It implements the story end-to-end via outside-in TDD, running `go tool mage testunit`/`lint` while iterating and `go tool mage test` before handing back, moves the story to `stories/completed/` once it has a real specification behind it, and hands back uncommitted work - it never commits.
3. **`/precommit`** before every commit - also enforced by a git hook, which refuses `git commit` until this has passed for the exact diff being committed. Runs the mechanical gates (`test`, `lint`, `mutate`), then spawns one `adr-checker` subagent per judgment-tier ADR whose `scope` overlaps the diff (`docs/source-control.md`). A flagged violation is either fixed and re-checked, or - if the fix isn't obvious - stops for a conversation rather than a guess.
4. If `mage mutate` turns up survivors mid-development (outside `/precommit`), hand them to the **`mutation-gap-closer`** agent - it writes the test that kills each one, or deletes the code if there's no real behaviour worth testing.
5. `git commit` once `/precommit` has passed.

Advisory, not part of the above sequence: **`story-drift-checker`** runs when a completed story's linked specification changes, flagging Gherkin that's drifted from what the specification actually verifies now.

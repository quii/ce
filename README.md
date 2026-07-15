# Conversation Engine (CE)

See `brief.md` and `docs/` for what this is and why.

## Commands

`mage` and every tool it shells out to (`golangci-lint`, `gremlins`) are tracked via `go.mod`'s `tool` directive, not installed globally - run everything through `go tool mage <target>` so it works regardless of `PATH`/`GOBIN` setup:

- `go tool mage test` - the full test suite (`go test -race -count=3 -shuffle=on ./...`), including both the in-process and containerised specifications (`docs/adr/0022-specifications-and-drivers.md`)
- `go tool mage lint` - `golangci-lint run ./...`
- `go tool mage mutate` - mutation testing scoped to the pending diff (`docs/adr/0020-mutation-testing.md`)

## Development flow

1. **`/story`** - facilitates an example-mapping conversation and writes the result to `stories/backlog/<name>.md` (`docs/story-process.md`). Doesn't write code.
2. **Ask for the `coder` agent** on that story once you're happy with it (e.g. "use the coder agent on stories/backlog/<name>.md"). It implements the story end-to-end via outside-in TDD, runs `go tool mage test`/`lint` itself, moves the story to `stories/completed/` once it has a real specification behind it, and hands back uncommitted work - it never commits.
3. **`/precommit`** before every commit - also enforced by a git hook, which refuses `git commit` until this has passed for the exact diff being committed. Runs the mechanical gates (`test`, `lint`, `mutate`), then spawns one `adr-checker` subagent per judgment-tier ADR whose `scope` overlaps the diff (`docs/source-control.md`). A flagged violation is either fixed and re-checked, or - if the fix isn't obvious - stops for a conversation rather than a guess.
4. If `mage mutate` turns up survivors mid-development (outside `/precommit`), hand them to the **`mutation-gap-closer`** agent - it writes the test that kills each one, or deletes the code if there's no real behaviour worth testing.
5. `git commit` once `/precommit` has passed.

Advisory, not part of the above sequence: **`story-drift-checker`** runs when a completed story's linked specification changes, flagging Gherkin that's drifted from what the specification actually verifies now.

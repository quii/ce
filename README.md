# Conversation Engine (CE)

See `brief.md` and `docs/` for what this is and why.

## Commands

`mage` and every tool it shells out to (`golangci-lint`, `gremlins`) are tracked via `go.mod`'s `tool` directive, not installed globally - run everything through `go tool mage <target>` so it works regardless of `PATH`/`GOBIN` setup:

- `go tool mage test` - the full test suite (`go test -race -count=3 -shuffle=on ./...`), including both the in-process and containerised specifications (`docs/adr/0022-specifications-and-drivers.md`)
- `go tool mage lint` - `golangci-lint run ./...`
- `go tool mage mutate` - mutation testing scoped to the pending diff (`docs/adr/0020-mutation-testing.md`)

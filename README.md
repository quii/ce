# Conversation Engine (CE)

See `brief.md` and `docs/` for what this is and why.

## Commands

- `mage test` - the full test suite (`go test -race -count=3 -shuffle=on ./...`), including both the in-process and containerised specifications (`docs/adr/0022-specifications-and-drivers.md`)
- `mage lint` - `golangci-lint run ./...`
- `mage mutate` - mutation testing scoped to the pending diff (`docs/adr/0020-mutation-testing.md`)

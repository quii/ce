# Source control

True trunk-based development: no feature branches, no long-lived branches, commits land directly on `main`.

That's a deliberate constraint, not a shortcut. With no branch to tidy things up on before they land, every commit has to be small and good enough to stand on its own - `main` doesn't get a "settling" period where it's allowed to be broken.

## What "small and frequent" looks like

A commit is the size of one coherent step, not a whole feature - closer to a single red-green-refactor cycle (or a small, related group of them) than to "the whole story." Watch for the failure mode this constraint exists to prevent: a large commit bundling several unrelated changes because they'd been sitting uncommitted for a while. A change that's been sitting in the working tree for a while unstaged is a sign it should have already been split into smaller pieces, not evidence it's finally ready to land as one big commit.

## Before every commit

A commit passes the same gates as everything else in this project - there's no separate, looser bar for "just a small commit":

- `go test -race -count=3 -shuffle=on ./...` is green (see `docs/development-practice.md` - this already gives full confidence on its own, nothing extra needed for CI)
- `golangci-lint run ./...` is clean (this also covers file length - see `docs/adr/0004-file-length.md`)
- mutation testing on the diff shows no unexplained survivors (see `docs/adr/0020-mutation-testing.md`)

This isn't a separate ceremony bolted onto committing - it's the same checks already being run constantly while working outside-in. Committing just means running them one more time, right before, on a diff small enough that doing so is cheap every single time.

## The ADR check

On top of the mechanical gates above, an automatic pre-commit hook checks the diff against the ADRs in `docs/adr/`. It skips three categories: ADRs whose `enforcement` is `mechanical` (already caught by the gates above, so re-checking them would be pure waste), ADRs whose `scope` doesn't overlap the changed files, and - when every changed file in the diff is confined to `cmd/web/**` or `internal/adapters/webui/**` - the check entirely. That web layer is deliberately kept outside the story process and outside the rest of the codebase's rigor bar for now (see `brief.md`'s developer experience note); running eight ADR subagents against a template-file move there was pure overhead. Revisit this exemption once that layer stops being demoware. What's left - `judgment`-tier ADRs relevant to a diff that isn't entirely web-only - each get one subagent checking the diff against that ADR specifically.

A flagged violation blocks the commit. If the fix is obvious, it gets fixed and the commit retried without involving anyone. If the fix isn't obvious, that's a stop, not a guess - it's worth a conversation before proceeding, not a judgment call for the agent to make alone.

//go:build mage

package main

import (
	"os"
	"os/exec"
)

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Test runs the full test suite: race detector, three runs, shuffled order.
func Test() error {
	return run("go", "test", "-race", "-count=3", "-shuffle=on", "./...")
}

// Lint runs golangci-lint over the whole module.
func Lint() error {
	return run("golangci-lint", "run", "./...")
}

// Mutate runs mutation testing scoped to the pending diff (working tree vs
// the last commit), failing if any mutant escapes and logging escapes in a
// format meant for an agent to consume - see docs/adr/0020-mutation-testing.md.
func Mutate() error {
	return run("go-mutesting",
		"--git-diff-lines", "--git-diff-base=HEAD",
		"--fail-on-escaped",
		"--logger-agentic-json",
		"--ignore-msi-with-no-mutations",
		"./...",
	)
}

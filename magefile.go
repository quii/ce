//go:build mage

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Run runs the application via go run, inheriting the current environment (set CE_ROLE to change the startup role).
func Run() error {
	cmd := exec.Command("go", "run", "./cmd/ce")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = append(os.Environ())
	return cmd.Run()
}

// Up builds and starts all services via docker compose.
func Up() error {
	return run("docker", "compose", "up", "--build")
}

// Test runs the full test suite: race detector, three runs, shuffled order.
func Test() error {
	return run("go", "test", "-race", "-count=3", "-shuffle=on", "./...")
}

// Lint runs golangci-lint over the whole module.
func Lint() error {
	return run("go", "tool", "golangci-lint", "run", "./...")
}

const mutateReportPath = "gremlins-report.json"

// Mutate runs mutation testing scoped to the pending diff (working tree vs
// the last commit), failing if any mutant survives - see
// docs/adr/0020-mutation-testing.md. gremlins treats an empty diff as
// "test everything" rather than "nothing to do", so a clean pending diff
// (e.g. a docs-only commit) skips the run entirely instead of gating on
// unrelated, pre-existing debt elsewhere in the repo.
//
// --integration is required: by default gremlins only runs `go test` on
// the package containing the mutated file, not the whole module. Most of
// this project's domain/use-case code has no _test.go files of its own -
// it's only exercised through the separate specifications package (outside-
// in TDD) - so without --integration every mutation there trivially
// "survives" (nothing in that package's own test run can fail). Confirmed
// by hand: two real mutants gremlins reported as LIVED were proven to be
// genuinely killed by manually applying them and running `go test ./...`
// directly; adding --integration (which switches gremlins' own test
// target to ./... per mutant) made both resolve to KILLED.
//
// The gate is enforced here off the JSON report's mutants_lived count
// rather than gremlins' own --threshold-efficacy/--threshold-mcover flags:
// those are silently inert in v0.6.0 (a viper/pflag float64 casting gap -
// viper.go has no case for a plain float64 flag, so it falls through to
// returning the raw string; gremlins' Get[float64] then does a failed type
// assertion against that string and quietly defaults to 0) - confirmed by
// hand, the threshold never fired at any value before this workaround.
func Mutate() error {
	if err := exec.Command("git", "diff", "--quiet", "HEAD", "--", "*.go").Run(); err == nil {
		return nil
	}

	if err := run("go", "tool", "gremlins", "unleash",
		"--diff=HEAD",
		"--coverpkg=./...",
		"--timeout-coefficient=30",
		"--integration",
		"-o", mutateReportPath,
		".",
	); err != nil {
		return err
	}

	return checkNoSurvivors(mutateReportPath)
}

func checkNoSurvivors(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("could not read %s: %w", path, err)
	}

	var report struct {
		MutantsLived int `json:"mutants_lived"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return fmt.Errorf("could not parse %s: %w", path, err)
	}
	if report.MutantsLived > 0 {
		return fmt.Errorf("%d mutant(s) survived - see %s", report.MutantsLived, path)
	}

	return nil
}

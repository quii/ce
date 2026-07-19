//go:build mage

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// portOwner returns a human-readable description of whatever process is
// already listening on port, or "" if the port is free (including if
// lsof itself isn't available - this check degrades to a no-op rather
// than blocking Up/Run on environments without it).
//
// This exists because a stale process left over from an earlier `go
// run`/`mage run` can win a port silently: Docker/OrbStack don't
// reliably surface that a port is already held by a native process as a
// bind error, so `mage up` can report success while every request
// actually reaches the wrong, long-dead process over the interface
// Docker didn't grab (observed first-hand - a `go run ./cmd/api` from
// two days earlier had been quietly answering on :8080 over IPv6 the
// entire time).
func portOwner(port int) string {
	out, err := exec.Command("lsof", "-nP", fmt.Sprintf("-iTCP:%d", port), "-sTCP:LISTEN", "-t").Output()
	if err != nil || strings.TrimSpace(string(out)) == "" {
		return ""
	}

	pid := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	cmdOut, err := exec.Command("ps", "-p", pid, "-o", "command=").Output()
	if err != nil {
		return "pid " + pid
	}
	return fmt.Sprintf("pid %s (%s)", pid, strings.TrimSpace(string(cmdOut)))
}

func checkPortsFree(ports ...int) error {
	for _, port := range ports {
		if owner := portOwner(port); owner != "" {
			return fmt.Errorf("port %d is already in use by %s - stop it first, then retry", port, owner)
		}
	}
	return nil
}

// Run runs the given app (api, relay, or web) via go run, inheriting the
// current environment - e.g. `go tool mage run api`.
func Run(service string) error {
	switch service {
	case "api", "web":
		if err := checkPortsFree(8080); err != nil {
			return err
		}
	case "relay":
	default:
		return fmt.Errorf("unknown service %q - want one of: api, relay, web", service)
	}

	cmd := exec.Command("go", "run", "./cmd/"+service)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = append(os.Environ())
	return cmd.Run()
}

// Up builds and starts all services via docker compose.
func Up() error {
	if err := checkPortsFree(8080, 8090, 5432); err != nil {
		return err
	}

	cmd := exec.Command("docker", "compose", "up", "--build")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "DOCKER_BUILDKIT=0")
	return cmd.Run()
}

// Test runs the full test suite: race detector, three runs, shuffled order,
// including the Docker-backed Postgres and container-driver specifications
// (the "integration" build tag - see docs/adr/0028-fast-and-full-test-tiers.md).
// This is the pre-commit gate: docs/source-control.md relies on it giving
// full confidence with no separate CI to catch what it misses.
func Test() error {
	return run("go", "test", "-race", "-count=3", "-shuffle=on", "-tags=integration", "./...")
}

// TestUnit runs the same suite as Test but without the "integration" build
// tag, so the Docker-backed Postgres contract tests and the container
// driver's specifications are excluded entirely - no containers started.
// This is the fast, in-memory-only path for the inner dev loop; it does not
// replace Test as the pre-commit gate (docs/adr/0028-fast-and-full-test-tiers.md).
func TestUnit() error {
	return run("go", "test", "-race", "-count=3", "-shuffle=on", "./...")
}

// Lint runs golangci-lint over the whole module, including files behind the
// "integration" build tag - without --build-tags, golangci-lint would
// silently stop analysing them.
func Lint() error {
	return run("go", "tool", "golangci-lint", "run", "--build-tags=integration", "./...")
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
//
// --exclude-files skips *.gen.go: golangci-lint already excludes generated
// files by convention (the "Code generated ... DO NOT EDIT" header), but
// gremlins has no equivalent built-in behaviour, so it's done explicitly
// here. Generated code (docs/adr/0024-openapi-spec-first-with-oapi-codegen.md)
// supports OpenAPI features this project doesn't use yet (e.g. array-style
// query params), which surfaced as LIVED mutants with no missing test to
// write - the fix for those is regenerating from a smaller spec, not adding
// a test that pins down someone else's generated branch.
//
// The "integration" build tag (--tags, see docs/adr/0028-fast-and-full-test-tiers.md)
// is only added when the diff actually touches a path that needs Docker to
// verify - internal/adapters/postgres or specifications/container. --integration
// itself stays on unconditionally regardless: it's what makes gremlins run
// go test against the whole module per mutant rather than just the mutated
// file's own package, which is what lets domain/use-case mutations be killed
// via the specifications package at all (see above). Without the build tag,
// those go test reruns simply don't compile the Docker-backed test files in,
// so an unrelated diff (domain, use-case, HTTP handler) never starts a
// container while still being checked against every other in-memory
// specification.
func Mutate() error {
	if err := exec.Command("git", "diff", "--quiet", "HEAD", "--", "*.go").Run(); err == nil {
		return nil
	}

	args := []string{
		"unleash",
		"--diff=HEAD",
		"--coverpkg=./...",
		"--timeout-coefficient=30",
		"--integration",
		"--exclude-files=.*\\.gen\\.go$",
		"-o", mutateReportPath,
	}
	if needsIntegrationTag() {
		args = append(args, "--tags=integration")
	}
	args = append(args, ".")

	if err := run("go", append([]string{"tool", "gremlins"}, args...)...); err != nil {
		return err
	}

	return checkNoSurvivors(mutateReportPath)
}

// needsIntegrationTag reports whether the pending diff touches a path that
// can only be verified against a real Postgres/container instance, in which
// case gremlins needs the "integration" build tag to see those tests at all.
func needsIntegrationTag() bool {
	out, err := exec.Command("git", "diff", "--name-only", "HEAD", "--", "*.go").Output()
	if err != nil {
		return true
	}

	for _, path := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.HasPrefix(path, "internal/adapters/postgres/") || strings.HasPrefix(path, "specifications/container/") {
			return true
		}
	}
	return false
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

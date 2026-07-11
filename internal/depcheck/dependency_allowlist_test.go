package depcheck_test

import (
	"os"
	"strings"
	"testing"
)

// Every direct (non-indirect) dependency must be flagged here before it's
// added - see docs/adr/0005-no-new-dependencies.md.
var allowlist = map[string]bool{
	"github.com/testcontainers/testcontainers-go": true,
}

func TestNoUnapprovedDirectDependencies(t *testing.T) {
	data, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatalf("failed to read go.mod: %v", err)
	}

	inRequireBlock := false
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)

		switch {
		case trimmed == "require (":
			inRequireBlock = true
			continue
		case inRequireBlock && trimmed == ")":
			inRequireBlock = false
			continue
		case strings.HasPrefix(trimmed, "require ") && !strings.HasSuffix(trimmed, "("):
			trimmed = strings.TrimPrefix(trimmed, "require ")
		case !inRequireBlock:
			continue
		}

		if trimmed == "" || strings.Contains(trimmed, "// indirect") {
			continue
		}

		fields := strings.Fields(trimmed)
		if len(fields) == 0 {
			continue
		}
		modulePath := fields[0]

		if !allowlist[modulePath] {
			t.Errorf("unapproved direct dependency %q in go.mod - flag it first, see docs/adr/0005-no-new-dependencies.md", modulePath)
		}
	}
}

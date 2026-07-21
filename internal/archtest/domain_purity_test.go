package archtest_test

import (
	"go/build"
	"strings"
	"testing"

	"github.com/quii/ce/internal/assert"
)

const modulePath = "github.com/quii/ce"

func TestDomainPurity(t *testing.T) {
	pkg, err := build.ImportDir("../domain", 0)
	assert.NoErr(t, err, "inspect domain package")

	for _, imp := range pkg.Imports {
		if imp == "log" || imp == "log/slog" {
			t.Errorf("domain package must not import %q - logging is not allowed in the domain, see docs/adr/0001-domain-purity.md", imp)
			continue
		}

		if strings.HasPrefix(imp, modulePath) {
			t.Errorf("domain package must not import %q - only the domain package itself, see docs/adr/0001-domain-purity.md", imp)
		}
	}
}

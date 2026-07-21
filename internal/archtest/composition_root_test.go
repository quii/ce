package archtest_test

import (
	"go/build"
	"strings"
	"testing"

	"github.com/quii/ce/internal/assert"
)

func TestPortsDoNotImportAdapters(t *testing.T) {
	adaptersPrefix := modulePath + "/internal/adapters"

	for _, dir := range []string{"../ports/in", "../ports/out"} {
		pkg, err := build.ImportDir(dir, 0)
		assert.NoErr(t, err, "inspect %s package", dir)

		for _, imp := range pkg.Imports {
			if imp == adaptersPrefix || strings.HasPrefix(imp, adaptersPrefix+"/") {
				t.Errorf("%s must not import %q - only cmd/** constructs adapters, see docs/adr/0025-composition-root.md", dir, imp)
			}
		}
	}
}

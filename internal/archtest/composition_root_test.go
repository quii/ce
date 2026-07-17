package archtest_test

import (
	"go/build"
	"strings"
	"testing"
)

func TestPortsDoNotImportAdapters(t *testing.T) {
	adaptersPrefix := modulePath + "/internal/adapters"

	for _, dir := range []string{"../ports/in", "../ports/out"} {
		pkg, err := build.ImportDir(dir, 0)
		if err != nil {
			t.Fatalf("failed to inspect %s package: %v", dir, err)
		}

		for _, imp := range pkg.Imports {
			if imp == adaptersPrefix || strings.HasPrefix(imp, adaptersPrefix+"/") {
				t.Errorf("%s must not import %q - only cmd/** constructs adapters, see docs/adr/0025-composition-root.md", dir, imp)
			}
		}
	}
}

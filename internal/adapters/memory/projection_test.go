package memory_test

import (
	"testing"

	"github.com/quii/ce/internal/adapters/contracttest"
	"github.com/quii/ce/internal/adapters/memory"
	"github.com/quii/ce/internal/ports/out"
)

func TestProjection_Contract(t *testing.T) {
	contracttest.Projection(t, func() out.Projection {
		return memory.NewProjection()
	})
}

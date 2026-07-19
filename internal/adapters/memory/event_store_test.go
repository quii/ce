package memory_test

import (
	"testing"

	"github.com/quii/ce/internal/adapters/contracttest"
	"github.com/quii/ce/internal/adapters/memory"
	"github.com/quii/ce/internal/ports/out"
)

func TestEventStore_Contract(t *testing.T) {
	contracttest.EventStore(t, func() out.EventStore {
		return memory.NewEventStore()
	})
}

func TestOutbox_Contract(t *testing.T) {
	contracttest.Outbox(t, func() out.Outbox {
		return memory.NewEventStore()
	})
}

func TestEventStoreOutbox_Contract(t *testing.T) {
	contracttest.EventStoreEnqueuesViaAppend(t, func() contracttest.EventStoreOutbox {
		return memory.NewEventStore()
	})
}

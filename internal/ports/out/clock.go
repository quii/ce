package out

import "time"

// Clock is the one legitimate way a use case gets the current time - see
// docs/adr/0015-utc-always.md.
type Clock interface {
	Now() time.Time
}

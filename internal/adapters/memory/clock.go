package memory

import "time"

type Clock struct{}

func NewClock() *Clock {
	return &Clock{}
}

func (*Clock) Now() time.Time {
	return time.Now().UTC() //nolint:forbidigo // this is the out.Clock adapter - the one place time.Now is meant to be called, see docs/adr/0015-utc-always.md
}

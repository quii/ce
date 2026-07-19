package out

// IDGenerator mints a new, unique, opaque identifier. Generating an ID is
// a real external need (randomness) that a use case can't own itself -
// see docs/adr/0001-domain-purity.md.
type IDGenerator interface {
	NewID() string
}

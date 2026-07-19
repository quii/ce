package memory

import (
	"crypto/rand"
	"encoding/hex"
)

type IDGenerator struct{}

func NewIDGenerator() *IDGenerator {
	return &IDGenerator{}
}

func (*IDGenerator) NewID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil { //nolint:forbidigo // this is the out.IDGenerator adapter - the one place randomness is meant to be sourced, see docs/adr/0001-domain-purity.md
		panic(err)
	}
	return hex.EncodeToString(buf)
}

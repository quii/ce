package httpapi

import (
	"errors"

	"github.com/quii/ce/internal/domain"
)

// domainErrorKind is the small, closed set of HTTP-relevant categories a
// domain error crossing out of a use case can fall into. Translating a
// domain error into a status code is still each handler's own job (see
// toConversation and docs/adr/0007-thin-http-handlers.md - a handler
// builds its own generated response type from the kind/message pair
// classifyDomainError returns), but every write handler in this package
// shares the same errors.As/errors.Is chain to arrive at that kind, so
// it's centralized here rather than repeated per handler - see
// docs/adr/0011-domain-errors-stay-domain-errors.md.
type domainErrorKind int

const (
	noDomainErrorKind domainErrorKind = iota
	validationErrorKind
	notFoundErrorKind
	forbiddenErrorKind
)

// classifyDomainError classifies err into the kind of response it should
// become, plus the message to carry: a domain.ValidationError is always
// validationErrorKind; domain.ErrConversationNotFound is always
// notFoundErrorKind, joined by additionalNotFoundErr when a handler has a
// second not-found sentinel of its own (e.g. ReplyToThread's
// domain.ErrThreadNotFound - pass nil when there isn't one);
// domain.ErrReplyForbidden is always forbiddenErrorKind. noDomainErrorKind
// means err doesn't match anything this package knows how to translate -
// the caller should return it as-is, and the generated strict handler's
// default maps it to 500.
func classifyDomainError(err error, additionalNotFoundErr error) (kind domainErrorKind, message string) {
	var validationErr domain.ValidationError
	if errors.As(err, &validationErr) {
		return validationErrorKind, validationErr.Error()
	}
	if errors.Is(err, domain.ErrConversationNotFound) || (additionalNotFoundErr != nil && errors.Is(err, additionalNotFoundErr)) {
		return notFoundErrorKind, err.Error()
	}
	if errors.Is(err, domain.ErrReplyForbidden) {
		return forbiddenErrorKind, err.Error()
	}

	return noDomainErrorKind, ""
}

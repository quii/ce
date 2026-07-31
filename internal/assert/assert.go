// Package assert holds this project's own small, generics-based test
// assertion helpers - see docs/adr/0027-structural-diffs-via-go-cmp.md.
// Every helper calls t.Helper() so a failure reports the caller's line,
// and every failure message states which operation or field was under
// test, not just "not equal" - the bar docs/adr/0012-clear-assertion-messages.md
// sets.
package assert

import (
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// Equal fails the test if got and want aren't structurally equal,
// reporting a cmp.Diff-based diff rather than a %#v dump of both values -
// one helper for scalars and structs/slices alike, since cmp.Diff
// handles scalars exactly as well as structural values. A nil slice/map
// and an empty one of the same type compare equal (cmpopts.EquateEmpty) -
// which one a value happens to be is usually an implementation detail
// (e.g. two different out-port adapters representing "no recipients"
// differently), not something a test should have to know to get right.
//
// T needs either no unexported fields or a public Equal method cmp
// recognises (time.Time has one; time.Location doesn't) - cmp.Diff
// panics on a struct with unexported fields it can't get at otherwise.
// A plain "if got != want" is still the right tool for a comparable type
// like *time.Location that trips this.
func Equal[T any](t *testing.T, got, want T, context string, args ...any) {
	t.Helper()
	if diff := cmp.Diff(want, got, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("%s: mismatch (-want +got):\n%s", fmt.Sprintf(context, args...), diff)
	}
}

func True(t *testing.T, got bool, context string, args ...any) {
	t.Helper()
	if !got {
		t.Errorf("%s: got false, want true", fmt.Sprintf(context, args...))
	}
}

func False(t *testing.T, got bool, context string, args ...any) {
	t.Helper()
	if got {
		t.Errorf("%s: got true, want false", fmt.Sprintf(context, args...))
	}
}

// NoErr fails and stops the test if err is non-nil. Unlike the comparison
// helpers above, this calls t.Fatalf rather than t.Errorf - almost every
// "unexpected error" check in this codebase halts the test, since
// there's nothing meaningful left to assert on once a call that was
// supposed to succeed didn't.
func NoErr(t *testing.T, err error, context string, args ...any) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s returned an unexpected error: %v", fmt.Sprintf(context, args...), err)
	}
}

func ErrorIs(t *testing.T, err, target error, context string, args ...any) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Errorf("%s returned err = %v, want %v", fmt.Sprintf(context, args...), err, target)
	}
}

// ErrorAs fails and stops the test if err doesn't unwrap to a T
// (errors.As), otherwise returning the extracted value so the caller can
// assert further on it (its message, say). Like NoErr, this halts the
// test rather than continuing to assert against a meaningless zero-value T.
func ErrorAs[T error](t *testing.T, err error, context string, args ...any) T {
	t.Helper()
	var target T
	if !errors.As(err, &target) {
		t.Fatalf("%s returned err = %v, want a %T", fmt.Sprintf(context, args...), err, target)
	}
	return target
}

// Len fails and stops the test if len(got) != want. Halts like NoErr and
// ErrorAs - a caller almost always indexes into got right after (got[0],
// say), which would panic on a shorter-than-expected slice if this only
// recorded a non-fatal failure and let the test carry on.
func Len[T any](t *testing.T, got []T, want int, context string, args ...any) {
	t.Helper()
	if len(got) != want {
		t.Fatalf("%s: len = %d, want %d (%#v)", fmt.Sprintf(context, args...), len(got), want, got)
	}
}

// Contains fails the test if want is not an element of haystack -
// equality checked via ==, which comparable guarantees is defined for T.
func Contains[T comparable](t *testing.T, haystack []T, want T, context string, args ...any) {
	t.Helper()
	for _, v := range haystack {
		if v == want {
			return
		}
	}
	t.Errorf("%s: %v does not contain %v", fmt.Sprintf(context, args...), haystack, want)
}

// NotContains fails the test if unwanted is an element of haystack -
// equality checked via ==, which comparable guarantees is defined for T.
func NotContains[T comparable](t *testing.T, haystack []T, unwanted T, context string, args ...any) {
	t.Helper()
	for _, v := range haystack {
		if v == unwanted {
			t.Errorf("%s: %v should not contain %v but it does", fmt.Sprintf(context, args...), haystack, unwanted)
			return
		}
	}
}

package assert_test

import (
	"errors"
	"testing"

	"github.com/quii/ce/internal/assert"
)

// fatalSafe runs f in its own goroutine and waits for it to finish,
// including via runtime.Goexit() - what t.Fatalf does after logging -
// so a helper under test that calls Fatalf can be exercised without
// unwinding the goroutine actually running this test.
func fatalSafe(f func()) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		f()
	}()
	<-done
}

func TestEqual(t *testing.T) {
	t.Run("passes when structurally equal", func(t *testing.T) {
		sub := &testing.T{}
		assert.Equal(sub, []string{"a", "b"}, []string{"a", "b"}, "recipients")
		if sub.Failed() {
			t.Errorf("Equal(equal slices) failed the test, want it to pass")
		}
	})

	t.Run("fails when structurally different", func(t *testing.T) {
		sub := &testing.T{}
		assert.Equal(sub, []string{"a", "c"}, []string{"a", "b"}, "recipients")
		if !sub.Failed() {
			t.Errorf("Equal(different slices) passed, want it to fail")
		}
	})

	t.Run("treats a nil slice and an empty slice as equal", func(t *testing.T) {
		sub := &testing.T{}
		var got []string
		assert.Equal(sub, got, []string{}, "recipients")
		if sub.Failed() {
			t.Errorf("Equal(nil, []string{}) failed the test, want it to pass - which one a value happens to be is usually an implementation detail")
		}
	})
}

func TestTrue(t *testing.T) {
	t.Run("passes when true", func(t *testing.T) {
		sub := &testing.T{}
		assert.True(sub, true, "condition")
		if sub.Failed() {
			t.Errorf("True(true) failed the test, want it to pass")
		}
	})

	t.Run("fails when false", func(t *testing.T) {
		sub := &testing.T{}
		assert.True(sub, false, "condition")
		if !sub.Failed() {
			t.Errorf("True(false) passed, want it to fail")
		}
	})
}

func TestFalse(t *testing.T) {
	t.Run("passes when false", func(t *testing.T) {
		sub := &testing.T{}
		assert.False(sub, false, "condition")
		if sub.Failed() {
			t.Errorf("False(false) failed the test, want it to pass")
		}
	})

	t.Run("fails when true", func(t *testing.T) {
		sub := &testing.T{}
		assert.False(sub, true, "condition")
		if !sub.Failed() {
			t.Errorf("False(true) passed, want it to fail")
		}
	})
}

func TestNoErr(t *testing.T) {
	t.Run("passes when err is nil", func(t *testing.T) {
		sub := &testing.T{}
		fatalSafe(func() { assert.NoErr(sub, nil, "Operation") })
		if sub.Failed() {
			t.Errorf("NoErr(nil) failed the test, want it to pass")
		}
	})

	t.Run("fails when err is non-nil", func(t *testing.T) {
		sub := &testing.T{}
		fatalSafe(func() { assert.NoErr(sub, errors.New("boom"), "Operation") })
		if !sub.Failed() {
			t.Errorf("NoErr(err) passed, want it to fail")
		}
	})
}

func TestErrorIs(t *testing.T) {
	target := errors.New("target")

	t.Run("passes when err wraps target", func(t *testing.T) {
		sub := &testing.T{}
		assert.ErrorIs(sub, target, target, "Operation")
		if sub.Failed() {
			t.Errorf("ErrorIs(target, target) failed the test, want it to pass")
		}
	})

	t.Run("fails when err doesn't wrap target", func(t *testing.T) {
		sub := &testing.T{}
		assert.ErrorIs(sub, errors.New("other"), target, "Operation")
		if !sub.Failed() {
			t.Errorf("ErrorIs(other, target) passed, want it to fail")
		}
	})
}

type testError struct{ msg string }

func (e testError) Error() string { return e.msg }

func TestErrorAs(t *testing.T) {
	t.Run("passes and returns the extracted value", func(t *testing.T) {
		sub := &testing.T{}
		var got testError
		fatalSafe(func() {
			got = assert.ErrorAs[testError](sub, testError{msg: "boom"}, "Operation")
		})
		if sub.Failed() {
			t.Errorf("ErrorAs(testError) failed the test, want it to pass")
		}
		if got.msg != "boom" {
			t.Errorf("ErrorAs(testError) returned %+v, want the extracted testError{msg: \"boom\"}", got)
		}
	})

	t.Run("fails and halts when err doesn't unwrap to T", func(t *testing.T) {
		sub := &testing.T{}
		fatalSafe(func() { _ = assert.ErrorAs[testError](sub, errors.New("plain"), "Operation") })
		if !sub.Failed() {
			t.Errorf("ErrorAs(plain error) passed, want it to fail")
		}
	})
}

func TestLen(t *testing.T) {
	t.Run("passes when the length matches", func(t *testing.T) {
		sub := &testing.T{}
		fatalSafe(func() { assert.Len(sub, []string{"a", "b"}, 2, "Thread.Messages") })
		if sub.Failed() {
			t.Errorf("Len(len=2, want=2) failed the test, want it to pass")
		}
	})

	t.Run("fails and halts when the length doesn't match", func(t *testing.T) {
		sub := &testing.T{}
		fatalSafe(func() { assert.Len(sub, []string{"a"}, 2, "Thread.Messages") })
		if !sub.Failed() {
			t.Errorf("Len(len=1, want=2) passed, want it to fail")
		}
	})
}

func TestContains(t *testing.T) {
	t.Run("passes when the element is present", func(t *testing.T) {
		sub := &testing.T{}
		assert.Contains(sub, []string{"user-1", "user-2"}, "user-2", "recipients")
		if sub.Failed() {
			t.Errorf("Contains(present) failed the test, want it to pass")
		}
	})

	t.Run("fails when the element is absent", func(t *testing.T) {
		sub := &testing.T{}
		assert.Contains(sub, []string{"user-1", "user-2"}, "user-3", "recipients")
		if !sub.Failed() {
			t.Errorf("Contains(absent) passed, want it to fail")
		}
	})
}

package specifications

import (
	"context"
	"testing"

	"github.com/quii/ce/internal/ports/in"
)

func GreetingSpecification(t *testing.T, greeter in.Greeter) {
	t.Helper()

	t.Run("no name supplied defaults to a generic greeting", func(t *testing.T) {
		assertGreeting(t, greeter, "Hello, World!", "")
	})

	t.Run("a name is supplied", func(t *testing.T) {
		assertGreeting(t, greeter, "Hello, Chris!", "Chris")
	})

	t.Run("a name with surrounding whitespace is trimmed", func(t *testing.T) {
		assertGreeting(t, greeter, "Hello, Chris!", " Chris ")
	})

	t.Run("an empty name is treated as no name", func(t *testing.T) {
		assertGreeting(t, greeter, "Hello, World!", "")
	})

	t.Run("a whitespace-only name is treated as no name", func(t *testing.T) {
		assertGreeting(t, greeter, "Hello, World!", "  ")
	})

	t.Run("a name with unrestricted characters is used as-is", func(t *testing.T) {
		assertGreeting(t, greeter, "Hello, 世界!", "世界")
	})
}

func assertGreeting(t *testing.T, greeter in.Greeter, want, name string) {
	t.Helper()

	got, err := greeter.Greet(context.Background(), in.GetGreetingCommand{Name: name})
	if err != nil {
		t.Fatalf("Greet(%q) returned an unexpected error: %v", name, err)
	}

	if string(got) != want {
		t.Errorf("Greet(%q) = %q, want %q", name, got, want)
	}
}

package specifications

import (
	"context"
	"testing"
)

func GreetingSpecification(t *testing.T, driver Driver) {
	t.Helper()

	t.Run("no name supplied defaults to a generic greeting", func(t *testing.T) {
		assertGreeting(t, driver, "Hello, World!", "")
	})

	t.Run("a name is supplied", func(t *testing.T) {
		assertGreeting(t, driver, "Hello, Chris!", "Chris")
	})

	t.Run("a name with surrounding whitespace is trimmed", func(t *testing.T) {
		assertGreeting(t, driver, "Hello, Chris!", " Chris ")
	})

	t.Run("an empty name is treated as no name", func(t *testing.T) {
		assertGreeting(t, driver, "Hello, World!", "")
	})

	t.Run("a whitespace-only name is treated as no name", func(t *testing.T) {
		assertGreeting(t, driver, "Hello, World!", "  ")
	})

	t.Run("a name with unrestricted characters is used as-is", func(t *testing.T) {
		assertGreeting(t, driver, "Hello, 世界!", "世界")
	})
}

func assertGreeting(t *testing.T, driver Driver, want, name string) {
	t.Helper()

	got, err := driver.Greeting(context.Background(), name)
	if err != nil {
		t.Fatalf("Greeting(%q) returned an unexpected error: %v", name, err)
	}

	if got != want {
		t.Errorf("Greeting(%q) = %q, want %q", name, got, want)
	}
}

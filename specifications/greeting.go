package specifications

import (
	"context"
	"testing"
)

func GreetingSpecification(t *testing.T, driver Driver) {
	t.Helper()

	got, err := driver.Greeting(context.Background())
	if err != nil {
		t.Fatalf("Greeting() returned an unexpected error: %v", err)
	}

	want := "Hello, World!"
	if got != want {
		t.Errorf("Greeting() = %q, want %q", got, want)
	}
}

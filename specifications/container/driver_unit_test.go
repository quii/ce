package container_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/in"
	"github.com/quii/ce/specifications/container"
)

// TestStartConversation_ValidationMessageComesFromResponseBody pins down
// that a 400's JSON body is what supplies the ValidationError's message -
// not the driver's own "request rejected" fallback - whenever the server
// actually sends one, as internal/adapters/httpapi/conversation_handler.go
// always does for a rejection it produced itself.
func TestStartConversation_ValidationMessageComesFromResponseBody(t *testing.T) {
	const wantMessage = "resourceUrl is required"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"` + wantMessage + `"}`))
	}))
	t.Cleanup(server.Close)

	driver := container.New(server.URL)

	_, err := driver.StartConversation(context.Background(), in.StartConversationCommand{})
	if err == nil {
		t.Fatalf("StartConversation against a stub 400 response returned no error, want a domain.ValidationError with message %q", wantMessage)
	}

	var validationErr domain.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("StartConversation returned err = %v (%T), want it to be a domain.ValidationError", err, err)
	}

	if got := validationErr.Error(); got != wantMessage {
		t.Errorf("ValidationError message = %q, want %q (the message from the response body's JSON400.Message, not the driver's fallback)", got, wantMessage)
	}
}

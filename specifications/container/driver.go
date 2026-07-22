package container

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/quii/ce/internal/adapters/apiclient"
	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/in"
)

type Driver struct {
	client *apiclient.ClientWithResponses
}

func New(baseURL string) *Driver {
	client, err := apiclient.NewClientWithResponses(baseURL)
	if err != nil {
		panic(err)
	}
	return &Driver{client: client}
}

// Drain is a no-op: this driver talks to a real, separate relay
// container that drains the outbox on its own ticker (docs/write-path.md),
// with no HTTP surface to trigger it on demand. Waiting for a specific
// write to be reflected is ConversationProjectionSpecification's job, via
// polling GetConversation - see specifications/conversation_projection.go.
func (d *Driver) Drain(_ context.Context) error {
	return nil
}

func (d *Driver) Greet(ctx context.Context, cmd in.GetGreetingCommand) (domain.Greeting, error) {
	var params apiclient.GetGreetingParams
	if cmd.Name != "" {
		params.Name = &cmd.Name
	}

	resp, err := d.client.GetGreetingWithResponse(ctx, &params)
	if err != nil {
		return "", err
	}

	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return "", fmt.Errorf("unexpected status code %d", resp.StatusCode())
	}

	return domain.Greeting(resp.JSON200.Greeting), nil
}

func (d *Driver) StartConversation(ctx context.Context, cmd in.StartConversationCommand) (in.StartConversationResult, error) {
	resp, err := d.client.StartConversationWithResponse(ctx, apiclient.StartConversationJSONRequestBody{
		ResourceUrl: cmd.ResourceURL,
		ThreadTitle: cmd.ThreadTitle,
		Author:      cmd.Author,
		Recipients:  cmd.Recipients,
		Message:     cmd.Message,
	})
	if err != nil {
		return in.StartConversationResult{}, err
	}

	if resp.StatusCode() == http.StatusBadRequest {
		message := "request rejected"
		if resp.JSON400 != nil {
			message = resp.JSON400.Message
		}
		return in.StartConversationResult{}, domain.NewValidationError(message)
	}

	if resp.StatusCode() != http.StatusAccepted {
		return in.StartConversationResult{}, fmt.Errorf("unexpected status code %d", resp.StatusCode())
	}

	id, seq, err := parseConversationLocation(resp.HTTPResponse.Header.Get("Location"))
	if err != nil {
		return in.StartConversationResult{}, err
	}

	return in.StartConversationResult{ConversationID: id, Sequence: seq}, nil
}

func (d *Driver) ReplyToThread(ctx context.Context, cmd in.ReplyToThreadCommand) (in.ReplyToThreadResult, error) {
	resp, err := d.client.ReplyToThreadWithResponse(ctx, cmd.ConversationID, cmd.ThreadID, apiclient.ReplyToThreadJSONRequestBody{
		Author: cmd.Author,
		Text:   cmd.Message,
	})
	if err != nil {
		return in.ReplyToThreadResult{}, err
	}

	switch resp.StatusCode() {
	case http.StatusBadRequest:
		message := "request rejected"
		if resp.JSON400 != nil {
			message = resp.JSON400.Message
		}
		return in.ReplyToThreadResult{}, domain.NewValidationError(message)
	case http.StatusForbidden:
		return in.ReplyToThreadResult{}, domain.ErrReplyForbidden
	case http.StatusNotFound:
		return in.ReplyToThreadResult{}, domain.ErrConversationNotFound
	case http.StatusAccepted:
	default:
		return in.ReplyToThreadResult{}, fmt.Errorf("unexpected status code %d", resp.StatusCode())
	}

	id, seq, err := parseConversationLocation(resp.HTTPResponse.Header.Get("Location"))
	if err != nil {
		return in.ReplyToThreadResult{}, err
	}

	return in.ReplyToThreadResult{ConversationID: id, Sequence: seq}, nil
}

func (d *Driver) GetConversation(ctx context.Context, cmd in.GetConversationCommand) (domain.ConversationView, error) {
	var params apiclient.GetConversationParams
	if cmd.After != nil {
		params.After = cmd.After
	}

	resp, err := d.client.GetConversationWithResponse(ctx, cmd.ConversationID, &params)
	if err != nil {
		return domain.ConversationView{}, err
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		if resp.JSON200 == nil {
			return domain.ConversationView{}, fmt.Errorf("200 response had no body")
		}
		return toConversationView(*resp.JSON200), nil
	case http.StatusAccepted:
		return domain.ConversationView{}, domain.ErrProjectionNotCaughtUp
	case http.StatusNotFound:
		return domain.ConversationView{}, domain.ErrConversationNotFound
	default:
		return domain.ConversationView{}, fmt.Errorf("unexpected status code %d", resp.StatusCode())
	}
}

func parseConversationLocation(location string) (domain.ConversationID, domain.Sequence, error) {
	u, err := url.Parse(location)
	if err != nil {
		return "", 0, fmt.Errorf("could not parse Location header %q: %w", location, err)
	}

	// The Location header this driver parses is always shaped like
	// /conversations/{id}?after=N - see
	// internal/adapters/httpapi/conversation_handler.go - so trimming the
	// known prefix is all that's needed to recover the id.
	id := domain.ConversationID(strings.TrimPrefix(u.Path, "/conversations/"))

	after, err := strconv.ParseInt(u.Query().Get("after"), 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("could not parse after= from Location header %q: %w", location, err)
	}

	return id, domain.Sequence(after), nil
}

func toConversationView(c apiclient.Conversation) domain.ConversationView {
	participants := make(domain.Recipients, len(c.Thread.Participants))
	for i, p := range c.Thread.Participants {
		participants[i] = domain.ParticipantID(p)
	}

	messages := make([]domain.MessageView, len(c.Thread.Messages))
	for i, m := range c.Thread.Messages {
		messages[i] = domain.MessageView{
			Author:   domain.ParticipantID(m.Author),
			Text:     domain.MessageText(m.Text),
			PostedAt: m.PostedAt,
		}
	}

	return domain.ConversationView{
		ID:          domain.ConversationID(c.Id),
		ResourceURL: domain.ResourceURL(c.ResourceUrl),
		Thread: domain.ThreadView{
			ID:           domain.ThreadID(c.Thread.Id),
			Title:        domain.ThreadTitle(c.Thread.Title),
			Participants: participants,
			Messages:     messages,
		},
	}
}

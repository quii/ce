package container

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/quii/ce/internal/adapters/apiclient"
	"github.com/quii/ce/internal/domain"
)

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
	threads := make([]domain.ThreadView, len(c.Threads))
	for i, t := range c.Threads {
		threads[i] = toThreadView(t)
	}

	return domain.ConversationView{
		ID:          domain.ConversationID(c.Id),
		ResourceURL: domain.ResourceURL(c.ResourceUrl),
		Threads:     threads,
	}
}

func toThreadView(t apiclient.Thread) domain.ThreadView {
	participants := make(domain.Recipients, len(t.Participants))
	for i, p := range t.Participants {
		participants[i] = domain.ParticipantID(p)
	}

	messages := make([]domain.MessageView, len(t.Messages))
	for i, m := range t.Messages {
		messages[i] = domain.MessageView{
			Author:   domain.ParticipantID(m.Author),
			Text:     domain.MessageText(m.Text),
			PostedAt: m.PostedAt,
		}
	}

	return domain.ThreadView{
		ID:           domain.ThreadID(t.Id),
		Title:        domain.ThreadTitle(t.Title),
		Participants: participants,
		Messages:     messages,
	}
}

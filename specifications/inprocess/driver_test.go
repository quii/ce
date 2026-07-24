package inprocess_test

import (
	"testing"

	"github.com/quii/ce/internal/adapters/memory"
	"github.com/quii/ce/internal/ports/in"
	"github.com/quii/ce/specifications"
)

func TestGreeting(t *testing.T) {
	greetings := memory.NewGreetingFinder()
	useCase := in.NewGetGreetingUseCase(greetings)

	specifications.GreetingSpecification(t, useCase)
}

type conversationDriver struct {
	in.ConversationStarter
	in.ThreadAdder
	in.ThreadReplier
	in.ConversationGetter
	in.Relay
}

func newConversationDriver() conversationDriver {
	events := memory.NewEventStore()
	projection := memory.NewProjection()

	return conversationDriver{
		ConversationStarter: in.NewStartConversationUseCase(in.StartConversationDependencies{
			IDs:    memory.NewIDGenerator(),
			Clock:  memory.NewClock(),
			Events: events,
		}),
		ThreadAdder: in.NewAddThreadUseCase(in.AddThreadDependencies{
			IDs:        memory.NewIDGenerator(),
			Clock:      memory.NewClock(),
			Events:     events,
			Projection: projection,
		}),
		ThreadReplier: in.NewReplyToThreadUseCase(in.ReplyToThreadDependencies{
			IDs:        memory.NewIDGenerator(),
			Clock:      memory.NewClock(),
			Events:     events,
			Projection: projection,
		}),
		ConversationGetter: in.NewGetConversationUseCase(projection),
		Relay:              in.NewRelay(events, projection),
	}
}

func TestStartConversation(t *testing.T) {
	specifications.ConversationSpecification(t, newConversationDriver())
}

func TestStartConversation_Projection(t *testing.T) {
	specifications.ConversationProjectionSpecification(t, newConversationDriver())
}

func TestReplyToThread(t *testing.T) {
	specifications.ReplyToThreadSpecification(t, newConversationDriver())
}

func TestAddThread(t *testing.T) {
	specifications.AddThreadSpecification(t, newConversationDriver())
}

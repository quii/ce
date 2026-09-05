package in

import (
	"context"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

// ManageThreadParticipantCommand is raw, not-yet-validated input for
// either an add or a remove - see docs/adr/0003-commands-not-parameter-lists.md
// and docs/adr/0010-tiny-types.md. Which of the two the caller wants is
// modelled by which ThreadParticipantManager method they call, not by a
// field on the command.
type ManageThreadParticipantCommand struct {
	ConversationID string
	ThreadID       string
	ParticipantID  string
}

// ManageThreadParticipantResult reports the sequence a real membership
// event was appended at (rule 5 of "manage thread participants"), or
// Changed=false when the request was already the current state and no
// event was appended (rule 6).
type ManageThreadParticipantResult struct {
	ConversationID domain.ConversationID
	Sequence       domain.Sequence
	Changed        bool
}

// ThreadParticipantManager is a single in-port covering both add and
// remove: no caller in the codebase needs one without the other (the HTTP
// handler routes both, the specification exercises both, the composition
// root wires both), so splitting them into two interfaces bought nothing
// but a double-construction at every wiring site.
type ThreadParticipantManager interface {
	AddThreadParticipant(context.Context, ManageThreadParticipantCommand) (ManageThreadParticipantResult, error)
	RemoveThreadParticipant(context.Context, ManageThreadParticipantCommand) (ManageThreadParticipantResult, error)
}

// ManageThreadParticipantDependencies deliberately does not include
// out.Projection: participant membership is a write invariant that has to
// be checked against the durable event log (rehydrated Conversation), not
// the projection, because the projection may lag - a stale membership
// read would surface as bogus 202s on repeats that should be immediate
// 204 no-ops (rule 6).
type ManageThreadParticipantDependencies struct {
	Clock  out.Clock
	Events out.EventStore
}

type manageThreadParticipantUseCase struct {
	deps ManageThreadParticipantDependencies
}

func NewManageThreadParticipantUseCase(deps ManageThreadParticipantDependencies) ThreadParticipantManager {
	return &manageThreadParticipantUseCase{deps: deps}
}

func (uc *manageThreadParticipantUseCase) AddThreadParticipant(ctx context.Context, cmd ManageThreadParticipantCommand) (ManageThreadParticipantResult, error) {
	return uc.manage(ctx, manageParticipantOp{Cmd: cmd, Action: domain.Conversation.AddParticipant})
}

func (uc *manageThreadParticipantUseCase) RemoveThreadParticipant(ctx context.Context, cmd ManageThreadParticipantCommand) (ManageThreadParticipantResult, error) {
	return uc.manage(ctx, manageParticipantOp{Cmd: cmd, Action: domain.Conversation.RemoveParticipant})
}

// manageParticipantOp bundles a single manage call's inputs into one
// value so the shared pipeline (uc.manage) stays within the
// docs/adr/0003-commands-not-parameter-lists.md limit of ctx + one
// parameter. Action is the aggregate method that decides whether an
// event should be raised - a method value like
// domain.Conversation.AddParticipant is an ordinary function of the
// aggregate + params by Go's method-expression rules, so no closure or
// flag field is needed to pick between add and remove.
type manageParticipantOp struct {
	Cmd    ManageThreadParticipantCommand
	Action func(domain.Conversation, domain.ManageThreadParticipantParams) (domain.Event, bool, error)
}

func (uc *manageThreadParticipantUseCase) manage(ctx context.Context, op manageParticipantOp) (ManageThreadParticipantResult, error) {
	conversationID := domain.ConversationID(op.Cmd.ConversationID)
	params := domain.ManageThreadParticipantParams{
		ConversationID: conversationID,
		ThreadID:       domain.ThreadID(op.Cmd.ThreadID),
		ParticipantID:  domain.ParticipantID(op.Cmd.ParticipantID),
	}
	if err := domain.ValidateManageThreadParticipant(params); err != nil {
		return ManageThreadParticipantResult{}, err
	}
	params.OccurredAt = uc.deps.Clock.Now()

	records, err := uc.deps.Events.ListByConversation(ctx, conversationID)
	if err != nil {
		return ManageThreadParticipantResult{}, err
	}
	conversation, err := domain.RehydrateConversation(records)
	if err != nil {
		return ManageThreadParticipantResult{}, err
	}

	event, changed, err := op.Action(conversation, params)
	if err != nil {
		return ManageThreadParticipantResult{}, err
	}
	if !changed {
		return ManageThreadParticipantResult{ConversationID: conversationID}, nil
	}

	seq, err := uc.deps.Events.Append(ctx, event)
	if err != nil {
		return ManageThreadParticipantResult{}, err
	}
	return ManageThreadParticipantResult{ConversationID: conversationID, Sequence: seq, Changed: true}, nil
}

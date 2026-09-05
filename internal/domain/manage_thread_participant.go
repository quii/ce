package domain

import "time"

type ParticipantAdded struct {
	ConversationID ConversationID
	ThreadID       ThreadID
	ParticipantID  ParticipantID
	OccurredAt     time.Time
}

type ParticipantRemoved struct {
	ConversationID ConversationID
	ThreadID       ThreadID
	ParticipantID  ParticipantID
	OccurredAt     time.Time
}

// Conversation is the event-sourced aggregate that owns thread membership.
// ConversationView is deliberately not used for write decisions because it
// may lag behind the durable event log.
type Conversation struct {
	ID      ConversationID
	threads map[ThreadID]Recipients
}

func RehydrateConversation(records []EventRecord) (Conversation, error) {
	if len(records) == 0 {
		return Conversation{}, ErrConversationNotFound
	}

	conversation := Conversation{threads: make(map[ThreadID]Recipients)}
	for _, record := range records {
		conversation.apply(record.Event)
	}

	return conversation, nil
}

func (c *Conversation) apply(event Event) {
	switch e := event.(type) {
	case ConversationCreated:
		c.ID = e.ConversationID
	case ThreadStarted:
		c.threads[e.ThreadID] = e.Participants()
	case ParticipantAdded:
		c.threads[e.ThreadID] = append(c.threads[e.ThreadID], e.ParticipantID)
	case ParticipantRemoved:
		participants := c.threads[e.ThreadID]
		for i, participant := range participants {
			if participant == e.ParticipantID {
				c.threads[e.ThreadID] = append(participants[:i:i], participants[i+1:]...)
				break
			}
		}
	}
}

// ManageThreadParticipantParams bundles the fields both AddParticipant
// and RemoveParticipant need from an already-validated command - see
// docs/adr/0003-commands-not-parameter-lists.md.
type ManageThreadParticipantParams struct {
	ConversationID ConversationID
	ThreadID       ThreadID
	ParticipantID  ParticipantID
	OccurredAt     time.Time
}

func ValidateManageThreadParticipant(params ManageThreadParticipantParams) error {
	if params.ParticipantID == "" {
		return ErrParticipantIDRequired
	}
	return nil
}

// AddParticipant returns the ParticipantAdded event to append, or reports
// changed=false when the participant is already a member of the thread
// (rule 6 of "manage thread participants" - the repeat is a no-op).
// ErrThreadNotFound covers both "no such thread" and "thread belongs to a
// different conversation" (rule 4).
func (c Conversation) AddParticipant(params ManageThreadParticipantParams) (Event, bool, error) {
	participants, ok := c.threads[params.ThreadID]
	if !ok {
		return nil, false, ErrThreadNotFound
	}
	if participants.Contains(params.ParticipantID) {
		return nil, false, nil
	}
	return ParticipantAdded{
		ConversationID: c.ID,
		ThreadID:       params.ThreadID,
		ParticipantID:  params.ParticipantID,
		OccurredAt:     params.OccurredAt,
	}, true, nil
}

// RemoveParticipant is AddParticipant's mirror: ParticipantRemoved when
// the participant is currently a member, no-op when they're already
// absent (rule 6).
func (c Conversation) RemoveParticipant(params ManageThreadParticipantParams) (Event, bool, error) {
	participants, ok := c.threads[params.ThreadID]
	if !ok {
		return nil, false, ErrThreadNotFound
	}
	if !participants.Contains(params.ParticipantID) {
		return nil, false, nil
	}
	return ParticipantRemoved{
		ConversationID: c.ID,
		ThreadID:       params.ThreadID,
		ParticipantID:  params.ParticipantID,
		OccurredAt:     params.OccurredAt,
	}, true, nil
}

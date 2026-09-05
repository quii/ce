package container

import (
	"fmt"
	"net/http"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/in"
)

func (d *Driver) manageThreadParticipantResponse(status int, location string) (in.ManageThreadParticipantResult, error) {
	switch status {
	case http.StatusNoContent:
		return in.ManageThreadParticipantResult{}, nil
	case http.StatusNotFound:
		return in.ManageThreadParticipantResult{}, domain.ErrConversationNotFound
	case http.StatusAccepted:
		id, seq, err := parseConversationLocation(location)
		if err != nil {
			return in.ManageThreadParticipantResult{}, err
		}
		return in.ManageThreadParticipantResult{ConversationID: id, Sequence: seq, Changed: true}, nil
	default:
		return in.ManageThreadParticipantResult{}, fmt.Errorf("unexpected status code %d", status)
	}
}

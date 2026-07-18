package usecase

import (
	"context"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/ports"
)

// AcceptRejectFlagUseCase handles flag state transitions (accept/reject).
type AcceptRejectFlagUseCase struct {
	sessionRepo ports.SessionRepository
}

// NewAcceptRejectFlagUseCase creates an AcceptRejectFlagUseCase.
func NewAcceptRejectFlagUseCase(sessionRepo ports.SessionRepository) *AcceptRejectFlagUseCase {
	return &AcceptRejectFlagUseCase{sessionRepo: sessionRepo}
}

// Execute accepts or rejects a flag in a session.
func (uc *AcceptRejectFlagUseCase) Execute(ctx context.Context, sessionID model.SessionID, flagID model.FlagID, accept bool) error {
	session, err := uc.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return &Error{Op: "AcceptRejectFlag", Message: "session not found", Err: err}
	}

	var stateErr error
	if accept {
		stateErr = session.AcceptFlag(flagID)
	} else {
		stateErr = session.RejectFlag(flagID)
	}
	if stateErr != nil {
		return &Error{Op: "AcceptRejectFlag", Message: "flag state transition", Err: stateErr}
	}

	return uc.sessionRepo.Save(ctx, session)
}

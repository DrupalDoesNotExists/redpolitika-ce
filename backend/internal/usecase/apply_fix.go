package usecase

import (
	"context"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/ports"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/service"
)

// ApplyFixUseCase applies flag fixes to session text.
type ApplyFixUseCase struct {
	sessionRepo ports.SessionRepository
	applier     *service.FixApplier
}

// NewApplyFixUseCase creates an ApplyFixUseCase.
func NewApplyFixUseCase(sessionRepo ports.SessionRepository, applier *service.FixApplier) *ApplyFixUseCase {
	return &ApplyFixUseCase{sessionRepo: sessionRepo, applier: applier}
}

// Execute applies a single flag fix and updates the session text.
func (uc *ApplyFixUseCase) Execute(ctx context.Context, sessionID model.SessionID, flagID model.FlagID) (*model.Suggestion, error) {
	session, err := uc.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, &Error{Op: "ApplyFix", Message: "session not found", Err: err}
	}

	// Validate and transition flag state
	suggestion, err := session.ApplyFlag(flagID)
	if err != nil {
		return nil, &Error{Op: "ApplyFix", Message: "apply flag", Err: err}
	}

	// Find the target flag for FixApplier
	var target *model.Flag
	for _, f := range session.Flags() {
		if f.ID() == flagID {
			target = f
			break
		}
	}
	if target == nil {
		return nil, &Error{Op: "ApplyFix", Message: "flag not found"}
	}

	// Apply fix to text — replaces span with suggestion, adjusts remaining flags
	result := uc.applier.Apply(session.Text(), target, session.Flags())
	session.SetText(result.Text)
	session.ReplaceFlags(result.RemainingFlags)

	if err := uc.sessionRepo.Save(ctx, session); err != nil {
		return nil, &Error{Op: "ApplyFix", Message: "save session", Err: err}
	}

	return &suggestion, nil
}

// ApplyAll applies all pending and accepted flag fixes in priority order.
func (uc *ApplyFixUseCase) ApplyAll(ctx context.Context, sessionID model.SessionID) (int, error) {
	session, err := uc.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return 0, &Error{Op: "ApplyAll", Message: "session not found", Err: err}
	}

	active := session.ActiveFlags()
	if len(active) == 0 {
		return 0, nil
	}

	result := uc.applier.ApplyAll(session.Text(), active)
	session.SetText(result.Text)
	session.ReplaceFlags(result.RemainingFlags)

	if err := uc.sessionRepo.Save(ctx, session); err != nil {
		return 0, &Error{Op: "ApplyAll", Message: "save session", Err: err}
	}

	return len(active) - len(result.RemainingFlags), nil
}

package book

import (
	"context"
	"hkorpo/book/pkg/errorwrapper"

	"github.com/google/uuid"
)

// RecordFailure persists a pipeline-stage failure for a book so the admin UI
// can surface it and offer a retry from that stage, then returns cause
// (wrapped) so the caller's existing error-logging path still fires.
func RecordFailure(ctx context.Context, repo Repository, bookID, stage string, cause error) error {
	id, err := uuid.Parse(bookID)
	if err != nil {
		return errorwrapper.Wrap(cause)
	}

	if err := repo.MarkBookFailed(ctx, id, stage, cause.Error()); err != nil {
		return errorwrapper.Wrap(err)
	}

	return errorwrapper.Wrap(cause)
}

// RecordPermanentFailure is like RecordFailure but marks the failure as
// permanent (see Repository.MarkBookFailedPermanently) — used when retrying
// would just repeat work that's already known to be a bad idea, e.g. AI
// spend for the stage exceeded its budget (pricing.BudgetExceededError).
func RecordPermanentFailure(ctx context.Context, repo Repository, bookID, stage string, cause error) error {
	id, err := uuid.Parse(bookID)
	if err != nil {
		return errorwrapper.Wrap(cause)
	}

	if err := repo.MarkBookFailedPermanently(ctx, id, stage, cause.Error()); err != nil {
		return errorwrapper.Wrap(err)
	}

	return errorwrapper.Wrap(cause)
}

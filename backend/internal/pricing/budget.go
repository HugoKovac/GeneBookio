package pricing

import (
	"context"
	"errors"
	"fmt"
	"hkorpo/book/internal/primitive"
)

// BudgetExceededError means a book's AI spend for one pipeline stage went
// over that stage's allotted EUR budget. Pipeline main.go's consumers treat
// this as a permanent failure (see book.RecordPermanentFailure) — retrying
// would just spend more money repeating work that's already over budget.
type BudgetExceededError struct {
	Stage    string
	Model    string
	CostEUR  float64
	LimitEUR float64
}

func (e *BudgetExceededError) Error() string {
	return fmt.Sprintf("%s stage cost €%.4f (model %s) exceeds the €%.2f budget for this book", e.Stage, e.CostEUR, e.Model, e.LimitEUR)
}

// CheckBudget returns a *BudgetExceededError if usage's cost for model,
// converted to EUR, exceeds limitEUR — nil otherwise (including when model
// isn't in the pricing table, e.g. the test-mode substitution clients,
// which always cost $0 and so never trip a budget).
//
// If the current USD->EUR exchange rate can't be determined, the USD cost
// is compared against limitEUR directly as a conservative fallback (rather
// than skip the check) — EUR and USD have stayed within a few percent of
// each other in practice.
func (c *Calculator) CheckBudget(ctx context.Context, stage, model string, usage primitive.ModelUsage, limitEUR float64) error {
	usd := CostUSD(primitive.TokenUsage{model: usage})
	if usd <= 0 {
		return nil
	}

	cost, err := c.CostEUR(ctx, usd)
	if err != nil {
		cost = usd
	}

	if cost <= limitEUR {
		return nil
	}

	return &BudgetExceededError{Stage: stage, Model: model, CostEUR: cost, LimitEUR: limitEUR}
}

// IsBudgetExceeded reports whether err is (or wraps) a *BudgetExceededError
// — the signal each cmd/*/main.go queue consumer uses to decide between
// book.RecordFailure (retryable) and book.RecordPermanentFailure.
func IsBudgetExceeded(err error) bool {
	var budgetErr *BudgetExceededError
	return errors.As(err, &budgetErr)
}

package pricing

import (
	"context"
	"errors"
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	got := EstimateTokens("12345678") // 8 chars
	if got != 2 {
		t.Errorf("EstimateTokens(8 chars) = %v, want 2", got)
	}
}

// TestCapOutputTokens_InputAloneExceedsBudget hits the real exchange-rate
// API (see budget_test.go's TestCheckBudget_OverBudget).
func TestCapOutputTokens_InputAloneExceedsBudget(t *testing.T) {
	c := NewCalculator(NewExchangeRateClient())

	// gpt-5.2 input is $1.75/1M tokens: 1M input tokens alone is ~$1.75,
	// already past a €1 budget under any plausible USD->EUR rate.
	_, err := c.CapOutputTokens(context.Background(), "prepare", "gpt-5.2", 1_000_000, 1.0)
	if err == nil {
		t.Fatal("expected CapOutputTokens to refuse before any request is made")
	}
	if !IsBudgetExceeded(err) {
		t.Errorf("expected a budget-exceeded error, got %T: %v", err, err)
	}
}

func TestCapOutputTokens_LeavesRoomForOutput(t *testing.T) {
	c := NewCalculator(NewExchangeRateClient())

	// A handful of input tokens leaves almost the entire €1 budget for
	// output, so the cap should come back well above zero.
	max, err := c.CapOutputTokens(context.Background(), "prepare", "gpt-5-mini", 100, 1.0)
	if err != nil {
		t.Fatalf("CapOutputTokens returned an error: %v", err)
	}
	if max <= 0 {
		t.Errorf("expected a positive output cap, got %v", max)
	}
}

func TestCapOutputTokens_UnpricedModelIsUncapped(t *testing.T) {
	c := NewCalculator(NewExchangeRateClient())

	max, err := c.CapOutputTokens(context.Background(), "prepare", "test-mode", 10_000_000, 1.0)
	if err != nil {
		t.Errorf("an unpriced model should never be refused pre-flight, got %v", err)
	}
	if max != 0 {
		t.Errorf("expected an unpriced model to come back uncapped (0), got %v", max)
	}
}

func TestCapOutputTokens_ReturnsBudgetExceededError(t *testing.T) {
	c := NewCalculator(NewExchangeRateClient())

	_, err := c.CapOutputTokens(context.Background(), "prepare", "gpt-5.2", 1_000_000, 1.0)
	var budgetErr *BudgetExceededError
	if !errors.As(err, &budgetErr) {
		t.Fatalf("expected a *BudgetExceededError, got %T: %v", err, err)
	}
	if budgetErr.Stage != "prepare" || budgetErr.Model != "gpt-5.2" {
		t.Errorf("unexpected error fields: %+v", budgetErr)
	}
}

package pricing

import (
	"context"
	"errors"
	"hkorpo/book/internal/primitive"
	"testing"
)

func TestCostUSD(t *testing.T) {
	usage := primitive.TokenUsage{
		"gpt-5-mini": {InputTokens: 500_000, OutputTokens: 200_000},
		"tts-1":      {InputTokens: 300_000},
		"unknown":    {InputTokens: 1_000_000, OutputTokens: 1_000_000},
	}

	got := CostUSD(usage)
	want := 500_000.0/1_000_000*0.25 + 200_000.0/1_000_000*2.00 + 300_000.0/1_000_000*15.00
	if got != want {
		t.Errorf("CostUSD(%v) = %v, want %v", usage, got, want)
	}
}

func TestCheckBudget_ZeroCostNeverExceeds(t *testing.T) {
	c := NewCalculator(NewExchangeRateClient())

	// "test-mode" isn't in the pricing table, so this must short-circuit
	// without ever calling the exchange-rate API.
	usage := primitive.ModelUsage{InputTokens: 10_000_000, OutputTokens: 10_000_000}
	if err := c.CheckBudget(context.Background(), "prepare", "test-mode", usage, 1.0); err != nil {
		t.Errorf("CheckBudget with an unpriced model should never exceed budget, got %v", err)
	}
}

// TestCheckBudget_OverBudget hits the real Frankfurter exchange-rate API
// (like internal/library's OpenLibrary test hits a real API) — expect it to
// be slow/flaky offline.
func TestCheckBudget_OverBudget(t *testing.T) {
	c := NewCalculator(NewExchangeRateClient())

	// gpt-5.2 output is $14/1M tokens, so 1M output tokens is way past a
	// €1 budget under any plausible USD->EUR rate.
	usage := primitive.ModelUsage{OutputTokens: 1_000_000}
	err := c.CheckBudget(context.Background(), "generate_script", "gpt-5.2", usage, 1.0)
	if err == nil {
		t.Fatal("expected CheckBudget to report the budget exceeded")
	}

	var budgetErr *BudgetExceededError
	if !errors.As(err, &budgetErr) {
		t.Fatalf("expected a *BudgetExceededError, got %T: %v", err, err)
	}
	if !IsBudgetExceeded(err) {
		t.Error("IsBudgetExceeded should report true for this error")
	}
	if budgetErr.LimitEUR != 1.0 {
		t.Errorf("LimitEUR = %v, want 1.0", budgetErr.LimitEUR)
	}
}

func TestCheckBudget_UnderBudget(t *testing.T) {
	c := NewCalculator(NewExchangeRateClient())

	usage := primitive.ModelUsage{InputTokens: 1_000, OutputTokens: 100}
	err := c.CheckBudget(context.Background(), "generate_script", "gpt-5.2", usage, 1.0)
	if err != nil {
		t.Errorf("small usage should stay within budget, got %v", err)
	}
}

package pricing

import "context"

// EstimateTokens approximates the number of tokens in text using the
// common ~4-characters-per-token rule of thumb for English/French prose.
// It's only used to size a preventive request cap before the real call is
// made — the display cost and the post-call budget check (CheckBudget)
// always use the API's own reported token counts, never this estimate.
func EstimateTokens(text string) int64 {
	return int64(len([]rune(text))) / 4
}

// limitToUSD converts a EUR budget to USD using the current exchange rate.
// If the rate can't be determined, it falls back to treating the EUR figure
// as USD directly — the same conservative fallback CheckBudget uses.
func (c *Calculator) limitToUSD(ctx context.Context, limitEUR float64) float64 {
	rate, err := c.exchangeRates.USDToEURRate(ctx)
	if err != nil || rate <= 0 {
		return limitEUR
	}
	return limitEUR / rate
}

// CapOutputTokens returns the largest number of output tokens model may
// generate without limitEUR being exceeded, given inputTokens (an estimate
// is fine — see EstimateTokens) that are already committed to being spent
// once the request goes out. The caller is expected to pass the result as
// the request's max-output-tokens parameter, bounding real spend *before*
// the request is made rather than only detecting overspend after (compare
// CheckBudget, which only runs once real usage is already known).
//
// Returns a *BudgetExceededError — without the request ever being made — if
// inputTokens alone already exceeds the budget. Returns (0, nil) if model
// isn't in the pricing table (nothing to cap against) or has no per-output
// price (e.g. tts-1, which CheckInputBudget covers instead).
func (c *Calculator) CapOutputTokens(ctx context.Context, stage, model string, inputTokens int64, limitEUR float64) (int64, error) {
	rate, priced := usdRatesPerMillion[model]
	if !priced {
		return 0, nil
	}

	limitUSD := c.limitToUSD(ctx, limitEUR)
	inputCost := float64(inputTokens) / 1_000_000 * rate.InputPerMillion
	remaining := limitUSD - inputCost
	if remaining <= 0 {
		cost, err := c.CostEUR(ctx, inputCost)
		if err != nil {
			cost = inputCost
		}
		return 0, &BudgetExceededError{Stage: stage, Model: model, CostEUR: cost, LimitEUR: limitEUR}
	}

	if rate.OutputPerMillion <= 0 {
		return 0, nil
	}
	return int64(remaining / rate.OutputPerMillion * 1_000_000), nil
}

package pricing

import (
	"context"
	"hkorpo/book/internal/primitive"
)

// Calculator turns a book's recorded TokenUsage into a display cost, in USD
// and (best-effort) EUR.
type Calculator struct {
	exchangeRates *ExchangeRateClient
}

func NewCalculator(exchangeRates *ExchangeRateClient) *Calculator {
	return &Calculator{exchangeRates: exchangeRates}
}

func (c *Calculator) CostUSD(usage primitive.TokenUsage) float64 {
	return CostUSD(usage)
}

// CostEUR converts usdCost to EUR using the current USD->EUR exchange rate.
// Returns an error only if no exchange rate (fresh or cached) is available;
// callers should treat that as "EUR cost unavailable right now" rather than
// fail the whole request.
func (c *Calculator) CostEUR(ctx context.Context, usdCost float64) (float64, error) {
	rate, err := c.exchangeRates.USDToEURRate(ctx)
	if err != nil {
		return 0, err
	}
	return usdCost * rate, nil
}

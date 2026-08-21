package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"hkorpo/book/pkg/errorwrapper"
	"net/http"
	"sync"
	"time"
)

// frankfurterURL is Frankfurter (https://frankfurter.dev), a free public
// exchange-rate API backed by the European Central Bank's daily reference
// rates. No API key required.
const frankfurterURL = "https://api.frankfurter.dev/v1/latest?base=USD&symbols=EUR"

// ExchangeRateClient fetches and caches the USD->EUR exchange rate. The rate
// only needs to be roughly current (it's used for a display-only cost
// estimate), so results are cached for rateTTL to avoid hitting the public
// API on every catalog request.
type ExchangeRateClient struct {
	httpClient *http.Client
	rateTTL    time.Duration

	mu        sync.Mutex
	rate      float64
	fetchedAt time.Time
}

func NewExchangeRateClient() *ExchangeRateClient {
	return &ExchangeRateClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		rateTTL:    time.Hour,
	}
}

type frankfurterResponse struct {
	Rates struct {
		EUR float64 `json:"EUR"`
	} `json:"rates"`
}

// USDToEURRate returns the current USD->EUR exchange rate, serving a cached
// value when it's still fresh. If a fetch fails but a stale cached rate is
// available, that stale rate is returned rather than an error.
func (c *ExchangeRateClient) USDToEURRate(ctx context.Context) (float64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.rate > 0 && time.Since(c.fetchedAt) < c.rateTTL {
		return c.rate, nil
	}

	rate, err := c.fetchRate(ctx)
	if err != nil {
		if c.rate > 0 {
			return c.rate, nil
		}
		return 0, err
	}

	c.rate = rate
	c.fetchedAt = time.Now()
	return c.rate, nil
}

func (c *ExchangeRateClient) fetchRate(ctx context.Context) (float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, frankfurterURL, nil)
	if err != nil {
		return 0, errorwrapper.Wrap(err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, errorwrapper.Wrap(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, errorwrapper.Wrap(fmt.Errorf("exchange rate API error (%d)", resp.StatusCode))
	}

	var parsed frankfurterResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return 0, errorwrapper.Wrap(err)
	}
	if parsed.Rates.EUR <= 0 {
		return 0, errorwrapper.Wrap(fmt.Errorf("exchange rate API returned no USD->EUR rate"))
	}

	return parsed.Rates.EUR, nil
}

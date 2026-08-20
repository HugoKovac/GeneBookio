package ttsapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hkorpo/book/pkg/errorwrapper"
	"net/http"
	"time"
)

func Init(cfg *ConfigTTSAPI) *Client {
	return &Client{
		client: &http.Client{
			Timeout: time.Minute * 5,
		},
		host: cfg.TTS_API_HOST,
	}
}

func (c *Client) Request(ctx context.Context, endpoint Endpoint, payload any) (*http.Response, error) {
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}

	req, err := http.NewRequest("POST", c.host+string(endpoint), bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return resp, errorwrapper.Wrap(err)

}

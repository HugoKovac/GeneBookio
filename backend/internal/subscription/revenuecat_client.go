package subscription

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"hkorpo/book/pkg/errorwrapper"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// https://www.revenuecat.com/docs/integrations/webhooks
// https://www.revenuecat.com/docs/api-v2

type ConfigRevenueCat struct {
	WebhookAuthHeader string `envconfig:"REVENUECAT_WEBHOOK_AUTH_HEADER"`
	SecretAPIKey      string `envconfig:"REVENUECAT_SECRET_API_KEY"`
	ProjectID         string `envconfig:"REVENUECAT_PROJECT_ID"`
	// EntitlementID is the single RevenueCat Entitlement identifier both the
	// App Store and Play Store products are attached to in the RevenueCat
	// dashboard — the flat single-plan gate, e.g. "premium".
	EntitlementID string `envconfig:"REVENUECAT_ENTITLEMENT_ID"`
}

type RevenueCatClient struct {
	client *http.Client
	cfg    *ConfigRevenueCat
}

func NewRevenueCatClient(cfg *ConfigRevenueCat) *RevenueCatClient {
	return &RevenueCatClient{
		client: &http.Client{Timeout: 15 * time.Second},
		cfg:    cfg,
	}
}

// entitlementRelevantEventTypes are the RevenueCat event types that affect
// whether a user is currently entitled — everything else (paywall views,
// experiment enrollments, virtual currency, TEST) is ignored.
var entitlementRelevantEventTypes = map[string]bool{
	"INITIAL_PURCHASE":      true,
	"RENEWAL":               true,
	"CANCELLATION":          true,
	"UNCANCELLATION":        true,
	"NON_RENEWING_PURCHASE": true,
	"SUBSCRIPTION_PAUSED":   true,
	"EXPIRATION":            true,
	"BILLING_ISSUE":         true,
	"PRODUCT_CHANGE":        true,
	"SUBSCRIPTION_EXTENDED": true,
	"TRANSFER":              true,
}

type webhookEnvelope struct {
	APIVersion string `json:"api_version"`
	Event      struct {
		Type                  string   `json:"type"`
		AppUserID             string   `json:"app_user_id"`
		EntitlementIDs        []string `json:"entitlement_ids"`
		ExpirationAtMs        int64    `json:"expiration_at_ms"`
		Store                 string   `json:"store"`
		OriginalTransactionID string   `json:"original_transaction_id"`
	} `json:"event"`
}

// ConstructWebhookEvent verifies the Authorization header against the
// configured shared secret and, for an entitlement-relevant event type,
// builds a RevenueCatSnapshot. Active is derived from "now < expiry", not
// from the event type itself — a CANCELLATION still leaves the user active
// until expiration_at_ms, and an EXPIRATION's own expiration_at_ms is
// already in the past, so this needs no per-event special-casing.
func (c *RevenueCatClient) ConstructWebhookEvent(rawBody []byte, authorizationHeader string) (string, *RevenueCatSnapshot, error) {
	if c.cfg.WebhookAuthHeader == "" || subtle.ConstantTimeCompare([]byte(authorizationHeader), []byte(c.cfg.WebhookAuthHeader)) != 1 {
		return "", nil, errorwrapper.Wrap(ErrInvalidRevenueCatWebhookAuth)
	}

	var envelope webhookEnvelope
	if err := json.Unmarshal(rawBody, &envelope); err != nil {
		return "", nil, errorwrapper.Wrap(err)
	}
	event := envelope.Event

	if !entitlementRelevantEventTypes[event.Type] {
		return event.Type, nil, nil
	}

	userID, err := uuid.Parse(event.AppUserID)
	if err != nil {
		return event.Type, nil, errorwrapper.Wrap(err)
	}

	entitlementID := c.cfg.EntitlementID
	if len(event.EntitlementIDs) > 0 {
		entitlementID = event.EntitlementIDs[0]
	}

	var expiresAt *time.Time
	if event.ExpirationAtMs > 0 {
		t := time.UnixMilli(event.ExpirationAtMs)
		expiresAt = &t
	}

	return event.Type, &RevenueCatSnapshot{
		UserID:                userID,
		Active:                expiresAt != nil && expiresAt.After(time.Now()),
		ExpiresAt:             expiresAt,
		Store:                 strings.ToLower(event.Store),
		EntitlementID:         entitlementID,
		OriginalTransactionID: event.OriginalTransactionID,
	}, nil
}

// customerActiveEntitlementsResponse mirrors the v2 API's
// CustomerActiveEntitlementList/CustomerEntitlement schemas (see
// list-customer-active-entitlements in
// https://www.revenuecat.com/docs/redocusaurus/openapi-v2.yaml) — a
// CustomerEntitlement only ever carries entitlement_id and expires_at
// (ms since epoch); store/original_transaction_id live on the separate
// Subscription resource, not here.
type customerActiveEntitlementsResponse struct {
	Items []struct {
		EntitlementID string `json:"entitlement_id"`
		ExpiresAt     *int64 `json:"expires_at"`
	} `json:"items"`
}

// GetEntitlementSnapshot fetches a subscriber's current entitlements
// directly from RevenueCat's REST API v2 — the fast path used right after a
// native purchase completes, so the UI doesn't have to wait for webhook
// delivery. Store/OriginalTransactionID are left unset here (Repository
// only overwrites them when non-empty, see UpsertRevenueCat) — they get
// filled in from the webhook payload, which carries them, shortly after.
func (c *RevenueCatClient) GetEntitlementSnapshot(ctx context.Context, appUserID string) (*RevenueCatSnapshot, error) {
	userID, err := uuid.Parse(appUserID)
	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}

	url := fmt.Sprintf("https://api.revenuecat.com/v2/projects/%s/customers/%s/active_entitlements", c.cfg.ProjectID, appUserID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.SecretAPIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}
	if resp.StatusCode == http.StatusNotFound {
		// RevenueCat has no customer record for this app_user_id yet (SDK
		// hasn't configured/purchased for them) — same as "never
		// subscribed", not an error worth failing the reconcile call for.
		return &RevenueCatSnapshot{UserID: userID, Active: false, EntitlementID: c.cfg.EntitlementID}, nil
	}
	if resp.StatusCode >= 300 {
		return nil, errorwrapper.Wrap(fmt.Errorf("revenuecat api error: %d: %s", resp.StatusCode, string(body)))
	}

	var parsed customerActiveEntitlementsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, errorwrapper.Wrap(err)
	}

	for _, item := range parsed.Items {
		if item.EntitlementID != c.cfg.EntitlementID {
			continue
		}
		var expiresAt *time.Time
		if item.ExpiresAt != nil {
			t := time.UnixMilli(*item.ExpiresAt)
			expiresAt = &t
		}
		return &RevenueCatSnapshot{
			UserID:        userID,
			Active:        expiresAt != nil && expiresAt.After(time.Now()),
			ExpiresAt:     expiresAt,
			EntitlementID: item.EntitlementID,
		}, nil
	}

	// No matching entitlement in the response — user has never purchased,
	// or it's expired and no longer listed. Either way, not active.
	return &RevenueCatSnapshot{UserID: userID, Active: false, EntitlementID: c.cfg.EntitlementID}, nil
}

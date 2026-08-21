package subscription

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func testRevenueCatClient() *RevenueCatClient {
	return NewRevenueCatClient(&ConfigRevenueCat{
		WebhookAuthHeader: "test-secret",
		EntitlementID:     "premium",
	})
}

func revenueCatWebhookBody(t *testing.T, eventType string, userID uuid.UUID, expirationAtMs int64) []byte {
	t.Helper()
	payload := map[string]any{
		"api_version": "1.0",
		"event": map[string]any{
			"type":                    eventType,
			"app_user_id":             userID.String(),
			"entitlement_ids":         []string{"premium"},
			"expiration_at_ms":        expirationAtMs,
			"store":                   "APP_STORE",
			"original_transaction_id": "txn_123",
		},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal webhook payload: %v", err)
	}
	return b
}

func TestRevenueCatConstructWebhookEvent_RejectsBadAuth(t *testing.T) {
	c := testRevenueCatClient()
	body := revenueCatWebhookBody(t, "INITIAL_PURCHASE", uuid.New(), time.Now().Add(time.Hour).UnixMilli())

	if _, _, err := c.ConstructWebhookEvent(body, "wrong-secret"); err == nil {
		t.Fatal("expected an error for a mismatched Authorization header")
	}
	if _, _, err := c.ConstructWebhookEvent(body, ""); err == nil {
		t.Fatal("expected an error for a missing Authorization header")
	}
}

func TestRevenueCatConstructWebhookEvent_InitialPurchaseIsActive(t *testing.T) {
	c := testRevenueCatClient()
	userID := uuid.New()
	expiresMs := time.Now().Add(30 * 24 * time.Hour).UnixMilli()
	body := revenueCatWebhookBody(t, "INITIAL_PURCHASE", userID, expiresMs)

	eventType, snapshot, err := c.ConstructWebhookEvent(body, "test-secret")
	if err != nil {
		t.Fatalf("ConstructWebhookEvent() error = %v", err)
	}
	if eventType != "INITIAL_PURCHASE" {
		t.Errorf("eventType = %q, want INITIAL_PURCHASE", eventType)
	}
	if snapshot == nil {
		t.Fatal("expected a non-nil snapshot")
	}
	if snapshot.UserID != userID {
		t.Errorf("UserID = %v, want %v", snapshot.UserID, userID)
	}
	if !snapshot.Active {
		t.Error("Active = false, want true (expiry is in the future)")
	}
	if snapshot.Store != "app_store" {
		t.Errorf("Store = %q, want app_store (lowercased)", snapshot.Store)
	}
	if snapshot.EntitlementID != "premium" {
		t.Errorf("EntitlementID = %q, want premium", snapshot.EntitlementID)
	}
}

// TestRevenueCatConstructWebhookEvent_CancellationStillActive verifies the
// key semantic difference from Stripe: RevenueCat's CANCELLATION means
// auto-renew was turned off, not that access ended — Active must still be
// true as long as expiration_at_ms is in the future.
func TestRevenueCatConstructWebhookEvent_CancellationStillActive(t *testing.T) {
	c := testRevenueCatClient()
	body := revenueCatWebhookBody(t, "CANCELLATION", uuid.New(), time.Now().Add(5*24*time.Hour).UnixMilli())

	_, snapshot, err := c.ConstructWebhookEvent(body, "test-secret")
	if err != nil {
		t.Fatalf("ConstructWebhookEvent() error = %v", err)
	}
	if !snapshot.Active {
		t.Error("a CANCELLATION event with a future expiry must still be Active")
	}
}

func TestRevenueCatConstructWebhookEvent_ExpirationIsInactive(t *testing.T) {
	c := testRevenueCatClient()
	body := revenueCatWebhookBody(t, "EXPIRATION", uuid.New(), time.Now().Add(-time.Hour).UnixMilli())

	_, snapshot, err := c.ConstructWebhookEvent(body, "test-secret")
	if err != nil {
		t.Fatalf("ConstructWebhookEvent() error = %v", err)
	}
	if snapshot.Active {
		t.Error("an EXPIRATION event with a past expiry must be Inactive")
	}
}

func TestRevenueCatConstructWebhookEvent_IrrelevantEventTypeIsNil(t *testing.T) {
	c := testRevenueCatClient()
	body := revenueCatWebhookBody(t, "TEST", uuid.New(), 0)

	eventType, snapshot, err := c.ConstructWebhookEvent(body, "test-secret")
	if err != nil {
		t.Fatalf("ConstructWebhookEvent() error = %v", err)
	}
	if eventType != "TEST" {
		t.Errorf("eventType = %q, want TEST", eventType)
	}
	if snapshot != nil {
		t.Errorf("snapshot = %+v, want nil for an event type we don't act on", snapshot)
	}
}

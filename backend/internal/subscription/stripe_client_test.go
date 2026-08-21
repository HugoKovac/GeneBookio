package subscription

import (
	"hkorpo/book/internal/primitive"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v86"
)

func TestSubscriptionToSnapshot_MapsFieldsFromMetadata(t *testing.T) {
	userID := uuid.New()
	periodEnd := time.Now().Add(30 * 24 * time.Hour).Unix()

	sub := &stripe.Subscription{
		ID:                "sub_123",
		Status:            stripe.SubscriptionStatusActive,
		CancelAtPeriodEnd: true,
		Customer:          &stripe.Customer{ID: "cus_123"},
		Metadata:          map[string]string{userIDMetadataKey: userID.String()},
		Items: &stripe.SubscriptionItemList{
			Data: []*stripe.SubscriptionItem{{CurrentPeriodEnd: periodEnd}},
		},
	}

	snapshot, err := subscriptionToSnapshot(sub)
	if err != nil {
		t.Fatalf("subscriptionToSnapshot() error = %v", err)
	}

	if snapshot.UserID != userID {
		t.Errorf("UserID = %v, want %v", snapshot.UserID, userID)
	}
	if snapshot.StripeCustomerID != "cus_123" {
		t.Errorf("StripeCustomerID = %q, want cus_123", snapshot.StripeCustomerID)
	}
	if snapshot.StripeSubscriptionID != "sub_123" {
		t.Errorf("StripeSubscriptionID = %q, want sub_123", snapshot.StripeSubscriptionID)
	}
	if snapshot.Status != primitive.SubscriptionActive {
		t.Errorf("Status = %v, want active", snapshot.Status)
	}
	if !snapshot.CancelAtPeriodEnd {
		t.Error("CancelAtPeriodEnd = false, want true")
	}
	if snapshot.CurrentPeriodEnd == nil || snapshot.CurrentPeriodEnd.Unix() != periodEnd {
		t.Errorf("CurrentPeriodEnd = %v, want unix %d", snapshot.CurrentPeriodEnd, periodEnd)
	}
}

func TestSubscriptionToSnapshot_MissingMetadataErrors(t *testing.T) {
	sub := &stripe.Subscription{ID: "sub_123", Status: stripe.SubscriptionStatusActive}

	if _, err := subscriptionToSnapshot(sub); err == nil {
		t.Fatal("expected an error when the subscription has no user_id metadata")
	}
}

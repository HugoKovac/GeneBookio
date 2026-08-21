package subscription

import (
	"context"
	"hkorpo/book/internal/primitive"
	"hkorpo/book/pkg/ent"
	"testing"
	"time"

	"github.com/google/uuid"
)

// fakeRepository is a minimal in-memory Repository used to unit test
// Service's business logic without a real database.
type fakeRepository struct {
	byUser map[uuid.UUID]*Subscription
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{byUser: map[uuid.UUID]*Subscription{}}
}

func (r *fakeRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*Subscription, error) {
	sub, ok := r.byUser[userID]
	if !ok {
		return nil, &ent.NotFoundError{}
	}
	cp := *sub
	return &cp, nil
}

func (r *fakeRepository) Upsert(ctx context.Context, snapshot *SubscriptionSnapshot) (*Subscription, error) {
	sub, ok := r.byUser[snapshot.UserID]
	if !ok {
		sub = &Subscription{ID: uuid.New(), UserID: snapshot.UserID}
		r.byUser[snapshot.UserID] = sub
	}
	if snapshot.StripeCustomerID != "" {
		sub.StripeCustomerID = snapshot.StripeCustomerID
	}
	if snapshot.StripeSubscriptionID != "" {
		sub.StripeSubscriptionID = snapshot.StripeSubscriptionID
	}
	sub.Status = snapshot.Status
	sub.CurrentPeriodEnd = snapshot.CurrentPeriodEnd
	sub.CancelAtPeriodEnd = snapshot.CancelAtPeriodEnd
	cp := *sub
	return &cp, nil
}

func (r *fakeRepository) UpsertRevenueCat(ctx context.Context, snapshot *RevenueCatSnapshot) (*Subscription, error) {
	sub, ok := r.byUser[snapshot.UserID]
	if !ok {
		sub = &Subscription{ID: uuid.New(), UserID: snapshot.UserID}
		r.byUser[snapshot.UserID] = sub
	}
	sub.RevenueCatActive = snapshot.Active
	sub.RevenueCatExpiresAt = snapshot.ExpiresAt
	if snapshot.Store != "" {
		sub.RevenueCatStore = snapshot.Store
	}
	if snapshot.EntitlementID != "" {
		sub.RevenueCatEntitlementID = snapshot.EntitlementID
	}
	if snapshot.OriginalTransactionID != "" {
		sub.RevenueCatOriginalTransactionID = snapshot.OriginalTransactionID
	}
	cp := *sub
	return &cp, nil
}

// fakeStripeAPI stubs the Stripe port for tests that drive Service directly
// via HandleWebhookEvent/ReconcileFromCheckoutSession rather than through
// the real client — HMAC signature mechanics are covered separately in
// stripe_client_test.go.
type fakeStripeAPI struct {
	webhookEventType string
	webhookSnapshot  *SubscriptionSnapshot

	checkoutUserID   uuid.UUID
	checkoutSubID    string
	checkoutComplete bool

	subscriptionSnapshot *SubscriptionSnapshot
}

func (f *fakeStripeAPI) CreateCheckoutSession(ctx context.Context, req CreateCheckoutSessionRequest) (string, error) {
	return "https://checkout.stripe.com/fake", nil
}
func (f *fakeStripeAPI) CreateBillingPortalSession(ctx context.Context, stripeCustomerID, returnURL string) (string, error) {
	return "https://billing.stripe.com/fake", nil
}
func (f *fakeStripeAPI) GetCheckoutSessionUserID(ctx context.Context, sessionID string) (uuid.UUID, string, bool, error) {
	return f.checkoutUserID, f.checkoutSubID, f.checkoutComplete, nil
}
func (f *fakeStripeAPI) GetSubscriptionSnapshot(ctx context.Context, stripeSubscriptionID string) (*SubscriptionSnapshot, error) {
	return f.subscriptionSnapshot, nil
}
func (f *fakeStripeAPI) ConstructWebhookEvent(rawBody []byte, signatureHeader string) (string, *SubscriptionSnapshot, error) {
	return f.webhookEventType, f.webhookSnapshot, nil
}

// fakeRevenueCatAPI stubs the RevenueCat port for tests that drive Service
// directly via HandleRevenueCatWebhookEvent/ReconcileFromRevenueCat.
type fakeRevenueCatAPI struct {
	webhookEventType string
	webhookSnapshot  *RevenueCatSnapshot

	entitlementSnapshot *RevenueCatSnapshot
}

func (f *fakeRevenueCatAPI) ConstructWebhookEvent(rawBody []byte, authorizationHeader string) (string, *RevenueCatSnapshot, error) {
	return f.webhookEventType, f.webhookSnapshot, nil
}
func (f *fakeRevenueCatAPI) GetEntitlementSnapshot(ctx context.Context, appUserID string) (*RevenueCatSnapshot, error) {
	return f.entitlementSnapshot, nil
}

func testPlanConfig() PlanConfig {
	return PlanConfig{
		PriceID:         "price_fake",
		SuccessURL:      "http://localhost/return",
		CancelURL:       "http://localhost/subscribe",
		PortalReturnURL: "http://localhost/dashboard",
	}
}

func TestHandleWebhookEvent_ActivatesFromSubscriptionSnapshot(t *testing.T) {
	repo := newFakeRepository()
	userID := uuid.New()
	periodEnd := time.Now().Add(30 * 24 * time.Hour)

	stripeAPI := &fakeStripeAPI{
		webhookEventType: "customer.subscription.created",
		webhookSnapshot: &SubscriptionSnapshot{
			UserID:               userID,
			StripeCustomerID:     "cus_1",
			StripeSubscriptionID: "sub_1",
			Status:               primitive.SubscriptionActive,
			CurrentPeriodEnd:     &periodEnd,
		},
	}
	svc := NewService(repo, stripeAPI, &fakeRevenueCatAPI{}, testPlanConfig())

	if err := svc.HandleWebhookEvent(context.Background(), []byte("{}"), "sig"); err != nil {
		t.Fatalf("HandleWebhookEvent() error = %v", err)
	}

	got, err := svc.GetStatus(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}
	if got == nil || got.Status != primitive.SubscriptionActive {
		t.Fatalf("GetStatus() = %+v, want active", got)
	}
	if !got.IsActive() {
		t.Error("an active subscription must report IsActive() == true")
	}
}

func TestHandleWebhookEvent_IgnoredEventTypeIsNoop(t *testing.T) {
	repo := newFakeRepository()
	stripeAPI := &fakeStripeAPI{webhookEventType: "invoice.paid", webhookSnapshot: nil}
	svc := NewService(repo, stripeAPI, &fakeRevenueCatAPI{}, testPlanConfig())

	if err := svc.HandleWebhookEvent(context.Background(), []byte("{}"), "sig"); err != nil {
		t.Fatalf("HandleWebhookEvent() error = %v", err)
	}
	if len(repo.byUser) != 0 {
		t.Errorf("expected no subscription row to be written, got %d", len(repo.byUser))
	}
}

func TestHandleWebhookEvent_PastDueStillGrantsAccess(t *testing.T) {
	repo := newFakeRepository()
	userID := uuid.New()
	stripeAPI := &fakeStripeAPI{
		webhookEventType: "customer.subscription.updated",
		webhookSnapshot: &SubscriptionSnapshot{
			UserID:               userID,
			StripeSubscriptionID: "sub_1",
			Status:               primitive.SubscriptionPastDue,
		},
	}
	svc := NewService(repo, stripeAPI, &fakeRevenueCatAPI{}, testPlanConfig())

	if err := svc.HandleWebhookEvent(context.Background(), []byte("{}"), "sig"); err != nil {
		t.Fatalf("HandleWebhookEvent() error = %v", err)
	}

	got, _ := svc.GetStatus(context.Background(), userID)
	if !got.IsActive() {
		t.Error("past_due must still grant access during Stripe's retry grace window")
	}
}

func TestHandleWebhookEvent_CanceledRevokesAccess(t *testing.T) {
	repo := newFakeRepository()
	userID := uuid.New()
	svc := NewService(repo, &fakeStripeAPI{
		webhookEventType: "customer.subscription.deleted",
		webhookSnapshot: &SubscriptionSnapshot{
			UserID:               userID,
			StripeSubscriptionID: "sub_1",
			Status:               primitive.SubscriptionCanceled,
		},
	}, &fakeRevenueCatAPI{}, testPlanConfig())

	if err := svc.HandleWebhookEvent(context.Background(), []byte("{}"), "sig"); err != nil {
		t.Fatalf("HandleWebhookEvent() error = %v", err)
	}

	got, _ := svc.GetStatus(context.Background(), userID)
	if got.IsActive() {
		t.Error("a canceled subscription must not grant access")
	}
}

func TestReconcileFromCheckoutSession_ActivatesWhenComplete(t *testing.T) {
	repo := newFakeRepository()
	userID := uuid.New()
	stripeAPI := &fakeStripeAPI{
		checkoutUserID:   userID,
		checkoutSubID:    "sub_1",
		checkoutComplete: true,
		subscriptionSnapshot: &SubscriptionSnapshot{
			UserID:               userID,
			StripeCustomerID:     "cus_1",
			StripeSubscriptionID: "sub_1",
			Status:               primitive.SubscriptionActive,
		},
	}
	svc := NewService(repo, stripeAPI, &fakeRevenueCatAPI{}, testPlanConfig())

	if err := svc.ReconcileFromCheckoutSession(context.Background(), userID, "cs_1"); err != nil {
		t.Fatalf("ReconcileFromCheckoutSession() error = %v", err)
	}

	got, _ := svc.GetStatus(context.Background(), userID)
	if !got.IsActive() {
		t.Error("expected the subscription to be active after reconciling a complete checkout session")
	}
}

func TestReconcileFromCheckoutSession_RejectsMismatchedUser(t *testing.T) {
	repo := newFakeRepository()
	userID := uuid.New()
	otherUserID := uuid.New()
	stripeAPI := &fakeStripeAPI{checkoutUserID: otherUserID, checkoutComplete: true, checkoutSubID: "sub_1"}
	svc := NewService(repo, stripeAPI, &fakeRevenueCatAPI{}, testPlanConfig())

	if err := svc.ReconcileFromCheckoutSession(context.Background(), userID, "cs_1"); err == nil {
		t.Fatal("expected an error when the checkout session belongs to a different user")
	}
}

// TestGetStatus_NoSubscriptionRowIsNotAnError guards the rollout-critical
// behavior: every existing user has zero Subscription rows today, and
// MiddlewareRequireActiveSubscription must treat that as "inactive" (via
// Subscription.IsActive's nil receiver), not as a request-failing error.
func TestGetStatus_NoSubscriptionRowIsNotAnError(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo, &fakeStripeAPI{}, &fakeRevenueCatAPI{}, testPlanConfig())

	sub, err := svc.GetStatus(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("GetStatus() error = %v, want nil", err)
	}
	if sub != nil {
		t.Errorf("GetStatus() = %v, want nil", sub)
	}
	if sub.IsActive() {
		t.Error("a nil Subscription must report IsActive() == false")
	}
}

// TestSubscription_StripeAndRevenueCatDoNotClobberEachOther is the single
// most important behavioral contract for this feature: a Stripe
// cancellation must not silently revoke access for a user who is
// independently active via RevenueCat (native purchase), and vice versa.
// This exercises the Service methods end-to-end against fakeRepository,
// which mirrors the real RepositoryImpl's namespace-isolated Upsert/
// UpsertRevenueCat split (see repository.go) — that ent-level split was
// also verified live against a running MySQL instance during development,
// since this repo has no DB-integration test harness to encode that here.
func TestSubscription_StripeAndRevenueCatDoNotClobberEachOther(t *testing.T) {
	repo := newFakeRepository()
	userID := uuid.New()
	expiresAt := time.Now().Add(30 * 24 * time.Hour)

	svc := NewService(repo, &fakeStripeAPI{}, &fakeRevenueCatAPI{
		webhookEventType: "INITIAL_PURCHASE",
		webhookSnapshot: &RevenueCatSnapshot{
			UserID: userID, Active: true, ExpiresAt: &expiresAt, Store: "app_store", EntitlementID: "premium",
		},
	}, testPlanConfig())

	if err := svc.HandleRevenueCatWebhookEvent(context.Background(), []byte("{}"), "auth"); err != nil {
		t.Fatalf("HandleRevenueCatWebhookEvent() error = %v", err)
	}

	// A Stripe cancellation for the same user arrives next (e.g. they also
	// once had, and canceled, an old web subscription).
	if _, err := repo.Upsert(context.Background(), &SubscriptionSnapshot{
		UserID: userID, Status: primitive.SubscriptionCanceled,
	}); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	got, err := svc.GetStatus(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}
	if !got.RevenueCatActive || got.RevenueCatExpiresAt == nil {
		t.Errorf("a Stripe cancellation must not clobber RevenueCat's independently-active state, got RevenueCatActive=%v RevenueCatExpiresAt=%v", got.RevenueCatActive, got.RevenueCatExpiresAt)
	}
	if !got.IsActive() {
		t.Error("the user should still be active via RevenueCat even though Stripe reports canceled")
	}
}

func TestHandleRevenueCatWebhookEvent_CancellationStillActiveUntilExpiry(t *testing.T) {
	repo := newFakeRepository()
	userID := uuid.New()
	expiresAt := time.Now().Add(5 * 24 * time.Hour) // still in the future

	svc := NewService(repo, &fakeStripeAPI{}, &fakeRevenueCatAPI{
		webhookEventType: "CANCELLATION",
		webhookSnapshot: &RevenueCatSnapshot{
			UserID: userID, Active: true, ExpiresAt: &expiresAt, Store: "play_store", EntitlementID: "premium",
		},
	}, testPlanConfig())

	if err := svc.HandleRevenueCatWebhookEvent(context.Background(), []byte("{}"), "auth"); err != nil {
		t.Fatalf("HandleRevenueCatWebhookEvent() error = %v", err)
	}

	got, _ := svc.GetStatus(context.Background(), userID)
	if !got.IsActive() {
		t.Error("a CANCELLATION event (auto-renew off) must still grant access until the expiry date")
	}
}

func TestHandleRevenueCatWebhookEvent_ExpirationRevokesAccess(t *testing.T) {
	repo := newFakeRepository()
	userID := uuid.New()
	expiresAt := time.Now().Add(-1 * time.Hour) // already in the past

	svc := NewService(repo, &fakeStripeAPI{}, &fakeRevenueCatAPI{
		webhookEventType: "EXPIRATION",
		webhookSnapshot: &RevenueCatSnapshot{
			UserID: userID, Active: false, ExpiresAt: &expiresAt, Store: "app_store", EntitlementID: "premium",
		},
	}, testPlanConfig())

	if err := svc.HandleRevenueCatWebhookEvent(context.Background(), []byte("{}"), "auth"); err != nil {
		t.Fatalf("HandleRevenueCatWebhookEvent() error = %v", err)
	}

	got, _ := svc.GetStatus(context.Background(), userID)
	if got.IsActive() {
		t.Error("an EXPIRATION event must revoke access")
	}
}

func TestHandleRevenueCatWebhookEvent_IgnoredEventTypeIsNoop(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo, &fakeStripeAPI{}, &fakeRevenueCatAPI{webhookEventType: "TEST", webhookSnapshot: nil}, testPlanConfig())

	if err := svc.HandleRevenueCatWebhookEvent(context.Background(), []byte("{}"), "auth"); err != nil {
		t.Fatalf("HandleRevenueCatWebhookEvent() error = %v", err)
	}
	if len(repo.byUser) != 0 {
		t.Errorf("expected no subscription row to be written, got %d", len(repo.byUser))
	}
}

func TestReconcileFromRevenueCat_SyncsEntitlement(t *testing.T) {
	repo := newFakeRepository()
	userID := uuid.New()
	expiresAt := time.Now().Add(30 * 24 * time.Hour)

	svc := NewService(repo, &fakeStripeAPI{}, &fakeRevenueCatAPI{
		entitlementSnapshot: &RevenueCatSnapshot{
			UserID: userID, Active: true, ExpiresAt: &expiresAt, Store: "app_store", EntitlementID: "premium",
		},
	}, testPlanConfig())

	if err := svc.ReconcileFromRevenueCat(context.Background(), userID); err != nil {
		t.Fatalf("ReconcileFromRevenueCat() error = %v", err)
	}

	got, _ := svc.GetStatus(context.Background(), userID)
	if !got.IsActive() {
		t.Error("expected the subscription to be active after reconciling from RevenueCat")
	}
}

package subscription

import (
	"context"
	"encoding/json"
	"errors"
	"hkorpo/book/internal/primitive"
	"hkorpo/book/pkg/errorwrapper"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v86"
	"github.com/stripe/stripe-go/v86/webhook"
)

// https://docs.stripe.com/api

type ConfigStripe struct {
	SecretKey     string `envconfig:"STRIPE_SECRET_KEY"`
	WebhookSecret string `envconfig:"STRIPE_WEBHOOK_SECRET"`
}

// userIDMetadataKey is how a Stripe Subscription is tied back to our own
// user: set once at Checkout time (see CreateCheckoutSession) and present
// on every subsequent customer.subscription.* webhook for that
// subscription, so no separate local lookup table is needed to map a
// webhook back to a user.
const userIDMetadataKey = "user_id"

type StripeClient struct {
	client *stripe.Client
	cfg    *ConfigStripe
	planID string
}

func NewStripeClient(cfg *ConfigStripe, priceID string) *StripeClient {
	return &StripeClient{
		client: stripe.NewClient(cfg.SecretKey),
		cfg:    cfg,
		planID: priceID,
	}
}

func (c *StripeClient) CreateCheckoutSession(ctx context.Context, req CreateCheckoutSessionRequest) (string, error) {
	params := &stripe.CheckoutSessionCreateParams{
		Mode: stripe.String(stripe.CheckoutSessionModeSubscription),
		LineItems: []*stripe.CheckoutSessionCreateLineItemParams{
			{Price: stripe.String(c.planID), Quantity: stripe.Int64(1)},
		},
		ClientReferenceID: stripe.String(req.UserID.String()),
		CustomerEmail:     stripe.String(req.CustomerEmail),
		SuccessURL:        stripe.String(req.SuccessURL),
		CancelURL:         stripe.String(req.CancelURL),
		SubscriptionData: &stripe.CheckoutSessionCreateSubscriptionDataParams{
			Metadata: map[string]string{userIDMetadataKey: req.UserID.String()},
		},
	}

	session, err := c.client.V1CheckoutSessions.Create(ctx, params)
	if err != nil {
		return "", errorwrapper.Wrap(err)
	}
	return session.URL, nil
}

func (c *StripeClient) CreateBillingPortalSession(ctx context.Context, stripeCustomerID, returnURL string) (string, error) {
	params := &stripe.BillingPortalSessionCreateParams{
		Customer:  stripe.String(stripeCustomerID),
		ReturnURL: stripe.String(returnURL),
	}
	session, err := c.client.V1BillingPortalSessions.Create(ctx, params)
	if err != nil {
		return "", errorwrapper.Wrap(err)
	}
	return session.URL, nil
}

func (c *StripeClient) GetCheckoutSessionUserID(ctx context.Context, sessionID string) (uuid.UUID, string, bool, error) {
	session, err := c.client.V1CheckoutSessions.Retrieve(ctx, sessionID, nil)
	if err != nil {
		return uuid.UUID{}, "", false, errorwrapper.Wrap(err)
	}

	userID, err := uuid.Parse(session.ClientReferenceID)
	if err != nil {
		return uuid.UUID{}, "", false, errorwrapper.Wrap(err)
	}

	stripeSubscriptionID := ""
	if session.Subscription != nil {
		stripeSubscriptionID = session.Subscription.ID
	}

	return userID, stripeSubscriptionID, session.Status == stripe.CheckoutSessionStatusComplete, nil
}

func (c *StripeClient) GetSubscriptionSnapshot(ctx context.Context, stripeSubscriptionID string) (*SubscriptionSnapshot, error) {
	sub, err := c.client.V1Subscriptions.Retrieve(ctx, stripeSubscriptionID, nil)
	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}
	return subscriptionToSnapshot(sub)
}

func (c *StripeClient) ConstructWebhookEvent(rawBody []byte, signatureHeader string) (string, *SubscriptionSnapshot, error) {
	event, err := webhook.ConstructEvent(rawBody, signatureHeader, c.cfg.WebhookSecret)
	if err != nil {
		return "", nil, errorwrapper.Wrap(err)
	}

	switch event.Type {
	case stripe.EventTypeCustomerSubscriptionCreated,
		stripe.EventTypeCustomerSubscriptionUpdated,
		stripe.EventTypeCustomerSubscriptionDeleted:
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return string(event.Type), nil, errorwrapper.Wrap(err)
		}
		snapshot, err := subscriptionToSnapshot(&sub)
		if err != nil {
			return string(event.Type), nil, err
		}
		return string(event.Type), snapshot, nil
	default:
		return string(event.Type), nil, nil
	}
}

func subscriptionToSnapshot(sub *stripe.Subscription) (*SubscriptionSnapshot, error) {
	rawUserID, ok := sub.Metadata[userIDMetadataKey]
	if !ok {
		return nil, errorwrapper.Wrap(errors.New("stripe subscription is missing the user_id metadata key"))
	}
	userID, err := uuid.Parse(rawUserID)
	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}

	var periodEnd *time.Time
	if sub.Items != nil && len(sub.Items.Data) > 0 {
		t := time.Unix(sub.Items.Data[0].CurrentPeriodEnd, 0)
		periodEnd = &t
	}

	customerID := ""
	if sub.Customer != nil {
		customerID = sub.Customer.ID
	}

	return &SubscriptionSnapshot{
		UserID:               userID,
		StripeCustomerID:     customerID,
		StripeSubscriptionID: sub.ID,
		Status:               primitive.SubscriptionStatus(sub.Status),
		CurrentPeriodEnd:     periodEnd,
		CancelAtPeriodEnd:    sub.CancelAtPeriodEnd,
	}, nil
}

package handler

import (
	"io"
	"net/http"

	billingapp "github.com/erp/backend/internal/application/billing"
	"github.com/gin-gonic/gin"
)

// Maximum webhook payload size (64KB - Stripe webhooks are typically small)
const maxWebhookPayloadSize = 65536

// StripeWebhookHandler handles Stripe webhook endpoints
// These endpoints are called by Stripe and do not require authentication
type StripeWebhookHandler struct {
	BaseHandler
	webhookService *billingapp.StripeWebhookService
}

// NewStripeWebhookHandler creates a new StripeWebhookHandler
func NewStripeWebhookHandler(webhookService *billingapp.StripeWebhookService) *StripeWebhookHandler {
	return &StripeWebhookHandler{
		webhookService: webhookService,
	}
}

// StripeWebhookResponse represents the response for Stripe webhook
//
//	@Description	Stripe webhook response
type StripeWebhookResponse struct {
	Received  bool   `json:"received" example:"true"`
	EventID   string `json:"event_id,omitempty" example:"evt_1234567890"`
	EventType string `json:"event_type,omitempty" example:"customer.subscription.created"`
	Message   string `json:"message,omitempty" example:"Webhook processed successfully"`
}

// HandleStripeWebhook godoc
//
//	@ID				handleStripeWebhook
//	@Summary		Handle Stripe webhook
//	@Description	Receive and process webhook events from Stripe for subscription management
//	@Tags			webhooks
//	@Accept			json
//	@Produce		json
//	@Param			Stripe-Signature	header		string					true	"Stripe webhook signature"
//	@Success		200					{object}	StripeWebhookResponse	"Webhook processed successfully"
//	@Failure		400					{object}	StripeWebhookResponse	"Invalid request"
//	@Failure		401					{object}	StripeWebhookResponse	"Invalid signature"
//	@Failure		413					{object}	StripeWebhookResponse	"Payload too large"
//	@Failure		500					{object}	StripeWebhookResponse	"Internal server error"
//	@Router			/webhooks/stripe [post]
func (h *StripeWebhookHandler) HandleStripeWebhook(c *gin.Context) {
	// Read the raw request body with size limit to prevent DoS attacks
	// Stripe requires the raw body for signature verification
	payload, err := io.ReadAll(io.LimitReader(c.Request.Body, maxWebhookPayloadSize+1))
	if err != nil {
		c.JSON(http.StatusBadRequest, StripeWebhookResponse{
			Received: false,
			Message:  "Failed to read request body",
		})
		return
	}

	// Check if payload exceeds size limit
	if len(payload) > maxWebhookPayloadSize {
		c.JSON(http.StatusRequestEntityTooLarge, StripeWebhookResponse{
			Received: false,
			Message:  "Payload too large",
		})
		return
	}

	// Get signature from header
	signature := c.GetHeader("Stripe-Signature")
	if signature == "" {
		c.JSON(http.StatusUnauthorized, StripeWebhookResponse{
			Received: false,
			Message:  "Missing Stripe-Signature header",
		})
		return
	}

	// Process the webhook
	result, err := h.webhookService.ProcessWebhook(c.Request.Context(), payload, signature)
	if err != nil {
		// Check if it's a signature verification error
		if result == nil {
			c.JSON(http.StatusUnauthorized, StripeWebhookResponse{
				Received: false,
				Message:  "Webhook signature verification failed",
			})
			return
		}

		// Other processing errors - still return 200 to prevent Stripe retries
		// for errors that won't be fixed by retrying
		// Note: Don't expose internal error details in response for security
		c.JSON(http.StatusOK, StripeWebhookResponse{
			Received:  true,
			EventID:   result.EventID,
			EventType: result.EventType,
			Message:   "Webhook received but processing encountered an issue",
		})
		return
	}

	c.JSON(http.StatusOK, StripeWebhookResponse{
		Received:  true,
		EventID:   result.EventID,
		EventType: result.EventType,
		Message:   result.Message,
	})
}

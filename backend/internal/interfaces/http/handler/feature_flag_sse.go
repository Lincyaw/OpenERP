package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/erp/backend/internal/domain/featureflag"
	"github.com/erp/backend/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// SSEClient represents a connected SSE client
type SSEClient struct {
	ID       string
	UserID   string
	TenantID string
	Chan     chan SSEMessage
	Done     chan struct{}
}

// SSEMessage represents a message to be sent to SSE clients
type SSEMessage struct {
	Event string `json:"event"`
	Data  string `json:"data"`
	ID    string `json:"id,omitempty"`
}

// FlagUpdatedEvent represents the data sent when a flag is updated
type FlagUpdatedEvent struct {
	Key   string                `json:"key"`
	Value FlagUpdatedEventValue `json:"value"`
}

// FlagUpdatedEventValue represents the flag value in the event
type FlagUpdatedEventValue struct {
	Enabled  bool    `json:"enabled"`
	Variant  *string `json:"variant,omitempty"`
	Metadata any     `json:"metadata,omitempty"`
}

// FeatureFlagSSEHandler handles SSE connections for feature flag updates
type FeatureFlagSSEHandler struct {
	BaseHandler
	invalidator featureflag.CacheInvalidator
	logger      *zap.Logger
	clients     sync.Map // map[string]*SSEClient
	ctx         context.Context
	cancel      context.CancelFunc
	heartbeat   time.Duration
	started     bool
	startMu     sync.Mutex
	maxClients  int // Maximum number of concurrent SSE clients
}

// FeatureFlagSSEOption is a functional option for configuring the handler
type FeatureFlagSSEOption func(*FeatureFlagSSEHandler)

// WithSSELogger sets the logger for the handler
func WithSSELogger(logger *zap.Logger) FeatureFlagSSEOption {
	return func(h *FeatureFlagSSEHandler) {
		h.logger = logger
	}
}

// WithSSEHeartbeat sets the heartbeat interval
func WithSSEHeartbeat(interval time.Duration) FeatureFlagSSEOption {
	return func(h *FeatureFlagSSEHandler) {
		h.heartbeat = interval
	}
}

// WithSSEMaxClients sets the maximum number of concurrent SSE clients
func WithSSEMaxClients(max int) FeatureFlagSSEOption {
	return func(h *FeatureFlagSSEHandler) {
		h.maxClients = max
	}
}

// NewFeatureFlagSSEHandler creates a new SSE handler for feature flag updates
func NewFeatureFlagSSEHandler(invalidator featureflag.CacheInvalidator, opts ...FeatureFlagSSEOption) *FeatureFlagSSEHandler {
	ctx, cancel := context.WithCancel(context.Background())
	h := &FeatureFlagSSEHandler{
		invalidator: invalidator,
		logger:      zap.NewNop(),
		ctx:         ctx,
		cancel:      cancel,
		heartbeat:   30 * time.Second,
		maxClients:  10000, // Default max clients
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// Start begins listening for cache updates and broadcasting to clients
func (h *FeatureFlagSSEHandler) Start() error {
	h.startMu.Lock()
	defer h.startMu.Unlock()

	if h.started {
		return fmt.Errorf("SSE handler already started")
	}

	// Start heartbeat goroutine
	go h.sendHeartbeats()

	// Start subscription to cache invalidation events
	go func() {
		err := h.invalidator.Subscribe(h.ctx, h.handleCacheUpdate)
		if err != nil && h.ctx.Err() == nil {
			h.logger.Error("SSE subscription error", zap.Error(err))
		}
	}()

	h.started = true
	h.logger.Info("Feature flag SSE handler started")
	return nil
}

// Stop stops the SSE handler
func (h *FeatureFlagSSEHandler) Stop() {
	h.cancel()

	// Close all client connections
	h.clients.Range(func(key, value any) bool {
		if client, ok := value.(*SSEClient); ok {
			close(client.Done)
		}
		return true
	})

	h.logger.Info("Feature flag SSE handler stopped")
}

// handleCacheUpdate processes cache update messages and broadcasts to clients
func (h *FeatureFlagSSEHandler) handleCacheUpdate(msg featureflag.CacheUpdateMessage) {
	// Convert cache update to SSE message
	event := h.cacheUpdateToEvent(msg)
	if event == nil {
		return
	}

	data, err := json.Marshal(event)
	if err != nil {
		h.logger.Error("Failed to marshal SSE event", zap.Error(err))
		return
	}

	sseMsg := SSEMessage{
		Event: "flag_updated",
		Data:  string(data),
		ID:    fmt.Sprintf("%d", msg.Timestamp),
	}

	h.broadcast(sseMsg)
}

// cacheUpdateToEvent converts a cache update message to an SSE event
func (h *FeatureFlagSSEHandler) cacheUpdateToEvent(msg featureflag.CacheUpdateMessage) *FlagUpdatedEvent {
	switch msg.Action {
	case featureflag.CacheUpdateActionUpdated:
		// Flag was updated - we send a simplified event
		// The client will need to fetch the full value
		return &FlagUpdatedEvent{
			Key: msg.FlagKey,
			Value: FlagUpdatedEventValue{
				Enabled: true, // Client should refetch to get actual value
			},
		}
	case featureflag.CacheUpdateActionDeleted:
		return &FlagUpdatedEvent{
			Key: msg.FlagKey,
			Value: FlagUpdatedEventValue{
				Enabled: false,
			},
		}
	case featureflag.CacheUpdateActionOverrideUpdated, featureflag.CacheUpdateActionOverrideDeleted:
		// For override changes, broadcast as flag update
		return &FlagUpdatedEvent{
			Key: msg.FlagKey,
			Value: FlagUpdatedEventValue{
				Enabled: true, // Client should refetch
			},
		}
	case featureflag.CacheUpdateActionInvalidateAll:
		// Signal that all flags should be refreshed
		return &FlagUpdatedEvent{
			Key: "*", // Special key to indicate all flags
			Value: FlagUpdatedEventValue{
				Enabled: true,
			},
		}
	default:
		return nil
	}
}

// broadcast sends a message to all connected clients
func (h *FeatureFlagSSEHandler) broadcast(msg SSEMessage) {
	h.clients.Range(func(key, value any) bool {
		client, ok := value.(*SSEClient)
		if !ok {
			return true
		}

		select {
		case client.Chan <- msg:
			h.logger.Debug("Sent SSE message to client",
				zap.String("client_id", client.ID),
				zap.String("event", msg.Event))
		default:
			// Channel full, client might be slow
			h.logger.Warn("Client channel full, dropping message",
				zap.String("client_id", client.ID))
		}
		return true
	})
}

// sendHeartbeats periodically sends heartbeat messages to keep connections alive
func (h *FeatureFlagSSEHandler) sendHeartbeats() {
	ticker := time.NewTicker(h.heartbeat)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.broadcast(SSEMessage{
				Event: "heartbeat",
				Data:  fmt.Sprintf(`{"timestamp":%d}`, time.Now().Unix()),
			})
		}
	}
}

// Stream godoc
//
//	@Summary		Subscribe to feature flag updates via SSE
//	@Description	Establishes a Server-Sent Events connection for real-time feature flag updates
//	@Tags			feature-flags
//	@Produce		text/event-stream
//	@Success		200	{string}	string	"SSE stream"
//	@Failure		401	{object}	dto.Response{error=dto.ErrorInfo}
//	@Failure		503	{object}	dto.Response{error=dto.ErrorInfo}
//	@Security		BearerAuth
//	@Router			/feature-flags/stream [get]
func (h *FeatureFlagSSEHandler) Stream(c *gin.Context) {
	// Rate limiting: Check if max clients reached
	if h.maxClients > 0 && h.GetClientCount() >= h.maxClients {
		c.JSON(503, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "MAX_CONNECTIONS_REACHED",
				"message": "Maximum number of SSE connections reached",
			},
		})
		return
	}

	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Get user context from JWT
	userID := middleware.GetJWTUserID(c)
	tenantID := middleware.GetJWTTenantID(c)

	// Create client with buffered channel
	// Buffer size allows messages to queue without blocking broadcast
	const sseMessageBufferSize = 100
	client := &SSEClient{
		ID:       uuid.New().String(),
		UserID:   userID,
		TenantID: tenantID,
		Chan:     make(chan SSEMessage, sseMessageBufferSize),
		Done:     make(chan struct{}),
	}

	// Register client
	h.clients.Store(client.ID, client)
	defer func() {
		// Close channel first to prevent sends to closed channel
		close(client.Chan)
		// Then delete from map
		h.clients.Delete(client.ID)
	}()

	h.logger.Info("SSE client connected",
		zap.String("client_id", client.ID),
		zap.String("user_id", userID),
		zap.String("tenant_id", tenantID))

	// Send initial connection event
	h.sendEvent(c.Writer, SSEMessage{
		Event: "connected",
		Data:  fmt.Sprintf(`{"client_id":"%s","timestamp":%d}`, client.ID, time.Now().Unix()),
	})
	c.Writer.Flush()

	// Get request context for cancellation
	reqCtx := c.Request.Context()

	// Stream events to client
	for {
		select {
		case <-reqCtx.Done():
			h.logger.Info("SSE client disconnected (request context done)",
				zap.String("client_id", client.ID))
			return
		case <-client.Done:
			h.logger.Info("SSE client disconnected (done signal)",
				zap.String("client_id", client.ID))
			return
		case <-h.ctx.Done():
			h.logger.Info("SSE handler stopped, disconnecting client",
				zap.String("client_id", client.ID))
			return
		case msg, ok := <-client.Chan:
			if !ok {
				// Channel closed
				return
			}
			h.sendEvent(c.Writer, msg)
			c.Writer.Flush()
		}
	}
}

// sendEvent writes an SSE event to the response writer
func (h *FeatureFlagSSEHandler) sendEvent(w io.Writer, msg SSEMessage) {
	if msg.Event != "" {
		fmt.Fprintf(w, "event: %s\n", msg.Event)
	}
	if msg.ID != "" {
		fmt.Fprintf(w, "id: %s\n", msg.ID)
	}
	fmt.Fprintf(w, "data: %s\n\n", msg.Data)
}

// GetClientCount returns the number of connected SSE clients
func (h *FeatureFlagSSEHandler) GetClientCount() int {
	count := 0
	h.clients.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

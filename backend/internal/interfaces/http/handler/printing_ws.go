package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/auth"
	"github.com/erp/backend/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Ensure net.Conn interface is used properly
type wsConn interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close() error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}

// PrintingWSClient represents a connected WebSocket client
type PrintingWSClient struct {
	ID       string
	UserID   string
	TenantID string
	conn     interface{} // net.Conn - using interface to avoid import cycle in tests
	writeMu  sync.Mutex
	done     chan struct{}
	lastPing time.Time
	pingMu   sync.RWMutex
}

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type      string          `json:"type"`
	EventType string          `json:"event_type,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Timestamp int64           `json:"timestamp"`
	ID        string          `json:"id,omitempty"`
}

// PrintingEventPayload represents the payload for printing events
type PrintingEventPayload struct {
	EventID        string `json:"event_id"`
	EventType      string `json:"event_type"`
	AggregateID    string `json:"aggregate_id"`
	AggregateType  string `json:"aggregate_type"`
	TenantID       string `json:"tenant_id"`
	OccurredAt     string `json:"occurred_at"`
	DocumentType   string `json:"document_type,omitempty"`
	DocumentID     string `json:"document_id,omitempty"`
	DocumentNumber string `json:"document_number,omitempty"`
	JobID          string `json:"job_id,omitempty"`
	TemplateID     string `json:"template_id,omitempty"`
	Status         string `json:"status,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
	PdfURL         string `json:"pdf_url,omitempty"`
	Copies         int    `json:"copies,omitempty"`
}

// PrintingWSHandler handles WebSocket connections for printing events
type PrintingWSHandler struct {
	BaseHandler
	eventBus    shared.EventSubscriber
	jwtService  *auth.JWTService
	logger      *zap.Logger
	clients     sync.Map // map[string]*PrintingWSClient (clientID -> client)
	tenantIndex sync.Map // map[string]map[string]bool (tenantID -> set of clientIDs)
	ctx         context.Context
	cancel      context.CancelFunc
	heartbeat   time.Duration
	pingTimeout time.Duration
	started     atomic.Bool
	startMu     sync.Mutex
	maxClients  int
	clientCount atomic.Int32
	addMu       sync.Mutex // Mutex for atomic check-and-add of clients
}

// PrintingWSOption is a functional option for configuring the handler
type PrintingWSOption func(*PrintingWSHandler)

// WithPrintingWSLogger sets the logger for the handler
func WithPrintingWSLogger(logger *zap.Logger) PrintingWSOption {
	return func(h *PrintingWSHandler) {
		h.logger = logger
	}
}

// WithPrintingWSHeartbeat sets the heartbeat interval
func WithPrintingWSHeartbeat(interval time.Duration) PrintingWSOption {
	return func(h *PrintingWSHandler) {
		h.heartbeat = interval
	}
}

// WithPrintingWSPingTimeout sets the ping timeout
func WithPrintingWSPingTimeout(timeout time.Duration) PrintingWSOption {
	return func(h *PrintingWSHandler) {
		h.pingTimeout = timeout
	}
}

// WithPrintingWSMaxClients sets the maximum number of concurrent clients
func WithPrintingWSMaxClients(max int) PrintingWSOption {
	return func(h *PrintingWSHandler) {
		h.maxClients = max
	}
}

// NewPrintingWSHandler creates a new WebSocket handler for printing events
func NewPrintingWSHandler(eventBus shared.EventSubscriber, jwtService *auth.JWTService, opts ...PrintingWSOption) *PrintingWSHandler {
	ctx, cancel := context.WithCancel(context.Background())
	h := &PrintingWSHandler{
		eventBus:    eventBus,
		jwtService:  jwtService,
		logger:      zap.NewNop(),
		ctx:         ctx,
		cancel:      cancel,
		heartbeat:   30 * time.Second,
		pingTimeout: 60 * time.Second,
		maxClients:  100, // Default max clients per requirement
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// Start begins listening for domain events and broadcasting to clients
func (h *PrintingWSHandler) Start() error {
	h.startMu.Lock()
	defer h.startMu.Unlock()

	if h.started.Load() {
		return fmt.Errorf("printing WebSocket handler already started")
	}

	// Subscribe to printing domain events
	h.eventBus.Subscribe(h, h.EventTypes()...)

	// Start heartbeat goroutine
	go h.sendHeartbeats()

	// Start ping checker goroutine
	go h.checkPingTimeouts()

	h.started.Store(true)
	h.logger.Info("Printing WebSocket handler started")
	return nil
}

// Stop stops the WebSocket handler
func (h *PrintingWSHandler) Stop() {
	h.cancel()

	// Unsubscribe from events
	h.eventBus.Unsubscribe(h)

	// Close all client connections
	h.clients.Range(func(key, value any) bool {
		if client, ok := value.(*PrintingWSClient); ok {
			h.closeClient(client)
		}
		return true
	})

	h.logger.Info("Printing WebSocket handler stopped")
}

// EventTypes returns the event types this handler is interested in
func (h *PrintingWSHandler) EventTypes() []string {
	return []string{
		// PrintJob events
		"PrintJobCreated",
		"PrintJobStatusChanged",
		"PrintJobCompleted",
		"PrintJobFailed",
		// PrintTemplate events
		"PrintTemplateCreated",
		"PrintTemplateUpdated",
		"PrintTemplateStatusChanged",
		"PrintTemplateSetAsDefault",
		"PrintTemplateDeleted",
		// AutoPrintRule events
		"AutoPrintRuleCreated",
		"AutoPrintRuleUpdated",
		"AutoPrintRuleEnabled",
		"AutoPrintRuleDisabled",
		// Trade events that trigger auto-print
		"SalesOrderConfirmed",
		"SalesOrderShipped",
		"PurchaseOrderReceived",
		"ReceiptVoucherPaid",
	}
}

// Handle processes domain events and broadcasts to relevant clients
func (h *PrintingWSHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	tenantID := event.TenantID().String()

	// Get clients for this tenant
	clientIDs := h.getClientIDsForTenant(tenantID)
	if len(clientIDs) == 0 {
		return nil // No clients for this tenant
	}

	// Build event payload
	payload := h.buildEventPayload(event)
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		h.logger.Error("Failed to marshal event payload",
			zap.String("event_type", event.EventType()),
			zap.Error(err))
		return nil // Don't return error to avoid blocking event bus
	}

	msg := WSMessage{
		Type:      "event",
		EventType: event.EventType(),
		Data:      payloadBytes,
		Timestamp: time.Now().UnixMilli(),
		ID:        event.EventID().String(),
	}

	// Broadcast to all clients of this tenant
	for _, clientID := range clientIDs {
		if clientVal, ok := h.clients.Load(clientID); ok {
			client := clientVal.(*PrintingWSClient)
			go h.sendToClient(client, msg)
		}
	}

	h.logger.Debug("Broadcasted event to tenant clients",
		zap.String("event_type", event.EventType()),
		zap.String("tenant_id", tenantID),
		zap.Int("client_count", len(clientIDs)))

	return nil
}

// buildEventPayload creates a payload from a domain event
func (h *PrintingWSHandler) buildEventPayload(event shared.DomainEvent) PrintingEventPayload {
	payload := PrintingEventPayload{
		EventID:       event.EventID().String(),
		EventType:     event.EventType(),
		AggregateID:   event.AggregateID().String(),
		AggregateType: event.AggregateType(),
		TenantID:      event.TenantID().String(),
		OccurredAt:    event.OccurredAt().Format(time.RFC3339),
	}

	// Extract additional fields based on event type using type assertion
	// This is a simplified approach - in production you might use reflection or interfaces
	if data, err := json.Marshal(event); err == nil {
		var extra map[string]interface{}
		if json.Unmarshal(data, &extra) == nil {
			if v, ok := extra["document_type"].(string); ok {
				payload.DocumentType = v
			}
			if v, ok := extra["document_id"].(string); ok {
				payload.DocumentID = v
			}
			if v, ok := extra["document_number"].(string); ok {
				payload.DocumentNumber = v
			}
			if v, ok := extra["job_id"].(string); ok {
				payload.JobID = v
			}
			if v, ok := extra["template_id"].(string); ok {
				payload.TemplateID = v
			}
			if v, ok := extra["status"].(string); ok {
				payload.Status = v
			}
			if v, ok := extra["new_status"].(string); ok {
				payload.Status = v
			}
			if v, ok := extra["error_message"].(string); ok {
				payload.ErrorMessage = v
			}
			if v, ok := extra["pdf_url"].(string); ok {
				payload.PdfURL = v
			}
			if v, ok := extra["copies"].(float64); ok {
				payload.Copies = int(v)
			}
		}
	}

	return payload
}

// getClientIDsForTenant returns all client IDs for a given tenant
func (h *PrintingWSHandler) getClientIDsForTenant(tenantID string) []string {
	var clientIDs []string
	if indexVal, ok := h.tenantIndex.Load(tenantID); ok {
		if index, ok := indexVal.(*sync.Map); ok {
			index.Range(func(key, _ any) bool {
				if clientID, ok := key.(string); ok {
					clientIDs = append(clientIDs, clientID)
				}
				return true
			})
		}
	}
	return clientIDs
}

// sendToClient sends a message to a specific client
func (h *PrintingWSHandler) sendToClient(client *PrintingWSClient, msg WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("Failed to marshal WebSocket message",
			zap.String("client_id", client.ID),
			zap.Error(err))
		return
	}

	client.writeMu.Lock()
	defer client.writeMu.Unlock()

	select {
	case <-client.done:
		return // Client already closed
	default:
	}

	if conn, ok := client.conn.(wsConn); ok {
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := wsutil.WriteServerMessage(conn, ws.OpText, data); err != nil {
			h.logger.Warn("Failed to send message to client",
				zap.String("client_id", client.ID),
				zap.Error(err))
			go h.removeClient(client)
		}
	}
}

// sendHeartbeats periodically sends heartbeat messages to keep connections alive
func (h *PrintingWSHandler) sendHeartbeats() {
	ticker := time.NewTicker(h.heartbeat)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			msg := WSMessage{
				Type:      "heartbeat",
				Timestamp: time.Now().UnixMilli(),
			}
			h.broadcastToAll(msg)
		}
	}
}

// checkPingTimeouts checks for clients that haven't responded to pings
func (h *PrintingWSHandler) checkPingTimeouts() {
	ticker := time.NewTicker(h.pingTimeout / 2)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			h.clients.Range(func(key, value any) bool {
				client := value.(*PrintingWSClient)
				client.pingMu.RLock()
				lastPing := client.lastPing
				client.pingMu.RUnlock()

				if now.Sub(lastPing) > h.pingTimeout {
					h.logger.Info("Client ping timeout, disconnecting",
						zap.String("client_id", client.ID),
						zap.Duration("since_last_ping", now.Sub(lastPing)))
					go h.removeClient(client)
				}
				return true
			})
		}
	}
}

// broadcastToAll sends a message to all connected clients
func (h *PrintingWSHandler) broadcastToAll(msg WSMessage) {
	h.clients.Range(func(key, value any) bool {
		client := value.(*PrintingWSClient)
		go h.sendToClient(client, msg)
		return true
	})
}

// addClient registers a new client
func (h *PrintingWSHandler) addClient(client *PrintingWSClient) {
	h.clients.Store(client.ID, client)
	h.clientCount.Add(1)

	// Add to tenant index
	indexVal, _ := h.tenantIndex.LoadOrStore(client.TenantID, &sync.Map{})
	if index, ok := indexVal.(*sync.Map); ok {
		index.Store(client.ID, true)
	}

	h.logger.Info("WebSocket client connected",
		zap.String("client_id", client.ID),
		zap.Int32("total_clients", h.clientCount.Load()))

	h.logger.Debug("WebSocket client details",
		zap.String("user_id", client.UserID),
		zap.String("tenant_id", client.TenantID))
}

// tryAddClient attempts to add a client atomically, respecting max client limit
// Returns true if client was added, false if max clients reached
func (h *PrintingWSHandler) tryAddClient(client *PrintingWSClient) bool {
	h.addMu.Lock()
	defer h.addMu.Unlock()

	// Check max clients under lock
	if h.maxClients > 0 && int(h.clientCount.Load()) >= h.maxClients {
		return false
	}

	// Add client
	h.addClient(client)
	return true
}

// removeClient unregisters a client
func (h *PrintingWSHandler) removeClient(client *PrintingWSClient) {
	// Remove from clients map
	if _, loaded := h.clients.LoadAndDelete(client.ID); !loaded {
		return // Already removed
	}
	h.clientCount.Add(-1)

	// Remove from tenant index
	if indexVal, ok := h.tenantIndex.Load(client.TenantID); ok {
		if index, ok := indexVal.(*sync.Map); ok {
			index.Delete(client.ID)
		}
	}

	h.closeClient(client)

	h.logger.Info("WebSocket client disconnected",
		zap.String("client_id", client.ID),
		zap.Int32("total_clients", h.clientCount.Load()))

	h.logger.Debug("WebSocket client disconnect details",
		zap.String("user_id", client.UserID),
		zap.String("tenant_id", client.TenantID))
}

// closeClient closes a client connection
func (h *PrintingWSHandler) closeClient(client *PrintingWSClient) {
	select {
	case <-client.done:
		return // Already closed
	default:
		close(client.done)
	}

	if conn, ok := client.conn.(interface{ Close() error }); ok {
		conn.Close()
	}
}

// GetClientCount returns the number of connected clients
func (h *PrintingWSHandler) GetClientCount() int {
	return int(h.clientCount.Load())
}

// GetClientCountForTenant returns the number of connected clients for a tenant
func (h *PrintingWSHandler) GetClientCountForTenant(tenantID string) int {
	count := 0
	if indexVal, ok := h.tenantIndex.Load(tenantID); ok {
		if index, ok := indexVal.(*sync.Map); ok {
			index.Range(func(_, _ any) bool {
				count++
				return true
			})
		}
	}
	return count
}

// =============================================================================
// HTTP Handler for WebSocket Connection
// =============================================================================

// Connect godoc
//
//	@ID				connectPrintingWS
//
//	@Summary		Connect to printing events WebSocket
//	@Description	Establishes a WebSocket connection for real-time printing event updates.
//	@Description	Authentication can be provided via:
//	@Description	- Authorization header: Bearer <token>
//	@Description	- Query parameter: ?token=<token>
//	@Tags			printing-ws
//	@Produce		json
//	@Param			token	query		string	false	"JWT token (alternative to Authorization header)"
//	@Success		101		{string}	string	"Switching Protocols"
//	@Failure		401		{object}	dto.ErrorResponse	"Unauthorized"
//	@Failure		503		{object}	dto.ErrorResponse	"Service Unavailable - Max connections reached"
//	@Security		BearerAuth
//	@Router			/printing/ws [get]
func (h *PrintingWSHandler) Connect(c *gin.Context) {
	// Get authentication from context (set by JWT middleware) or query param
	var userID, tenantID string

	// First try to get from JWT middleware context
	userID = middleware.GetJWTUserID(c)
	tenantID = middleware.GetJWTTenantID(c)

	// If not in context, try query parameter (for WebSocket clients that can't set headers)
	if userID == "" || tenantID == "" {
		tokenString := c.Query("token")
		if tokenString != "" {
			// Basic sanity check - JWT tokens are typically 100-500 chars
			if len(tokenString) > 2000 || len(tokenString) < 50 {
				c.JSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "INVALID_TOKEN_FORMAT",
						"message": "Invalid token format",
					},
				})
				return
			}
			claims, err := h.jwtService.ValidateAccessToken(tokenString)
			if err != nil {
				h.logger.Warn("WebSocket authentication failed",
					zap.Error(err),
					zap.String("path", c.Request.URL.Path))
				c.JSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "UNAUTHORIZED",
						"message": "Invalid or expired token",
					},
				})
				return
			}
			userID = claims.UserID
			tenantID = claims.TenantID
		}
	}

	// Validate we have authentication
	if userID == "" || tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "Authentication required",
			},
		})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, _, _, err := ws.UpgradeHTTP(c.Request, c.Writer)
	if err != nil {
		h.logger.Error("Failed to upgrade WebSocket connection",
			zap.Error(err))
		return // UpgradeHTTP already wrote the error response
	}

	// Create client
	client := &PrintingWSClient{
		ID:       uuid.New().String(),
		UserID:   userID,
		TenantID: tenantID,
		conn:     conn,
		done:     make(chan struct{}),
		lastPing: time.Now(),
	}

	// Try to register client atomically (respects max client limit)
	if !h.tryAddClient(client) {
		// Max clients reached - close connection and return error
		conn.Close()
		h.logger.Warn("WebSocket connection rejected: max clients reached",
			zap.Int("max_clients", h.maxClients))
		return
	}

	// Send connected message
	connectedMsg := WSMessage{
		Type:      "connected",
		Timestamp: time.Now().UnixMilli(),
		ID:        client.ID,
	}
	h.sendToClient(client, connectedMsg)

	// Start reading messages from client in a goroutine
	go h.readClientMessages(client)
}

// readClientMessages reads messages from a WebSocket client
func (h *PrintingWSHandler) readClientMessages(client *PrintingWSClient) {
	defer h.removeClient(client)

	conn, ok := client.conn.(wsConn)
	if !ok {
		return
	}

	for {
		select {
		case <-client.done:
			return
		case <-h.ctx.Done():
			return
		default:
		}

		// Set read deadline
		conn.SetReadDeadline(time.Now().Add(h.pingTimeout))

		// Read message
		data, op, err := wsutil.ReadClientData(conn)
		if err != nil {
			// Connection closed or error
			h.logger.Debug("WebSocket read error",
				zap.String("client_id", client.ID),
				zap.Error(err))
			return
		}

		// Handle different message types
		switch op {
		case ws.OpText:
			h.handleClientMessage(client, data)
		case ws.OpPing:
			// Respond with pong
			client.writeMu.Lock()
			wsutil.WriteServerMessage(conn, ws.OpPong, data)
			client.writeMu.Unlock()
			h.updateClientPing(client)
		case ws.OpPong:
			h.updateClientPing(client)
		case ws.OpClose:
			return
		}
	}
}

// handleClientMessage processes a text message from a client
func (h *PrintingWSHandler) handleClientMessage(client *PrintingWSClient, data []byte) {
	var msg WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		h.logger.Warn("Invalid WebSocket message from client",
			zap.String("client_id", client.ID),
			zap.Error(err))
		return
	}

	switch msg.Type {
	case "ping":
		// Client ping - respond with pong
		h.updateClientPing(client)
		pongMsg := WSMessage{
			Type:      "pong",
			Timestamp: time.Now().UnixMilli(),
		}
		h.sendToClient(client, pongMsg)
	case "pong":
		// Client pong response
		h.updateClientPing(client)
	default:
		h.logger.Debug("Unknown message type from client",
			zap.String("client_id", client.ID),
			zap.String("type", msg.Type))
	}
}

// updateClientPing updates the last ping time for a client
func (h *PrintingWSHandler) updateClientPing(client *PrintingWSClient) {
	client.pingMu.Lock()
	client.lastPing = time.Now()
	client.pingMu.Unlock()
}

// Ensure PrintingWSHandler implements EventHandler
var _ shared.EventHandler = (*PrintingWSHandler)(nil)

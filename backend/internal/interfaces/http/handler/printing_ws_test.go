package handler

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockEventSubscriber implements shared.EventSubscriber for testing
type mockEventSubscriber struct {
	handlers []shared.EventHandler
	mu       sync.Mutex
}

func newMockEventSubscriber() *mockEventSubscriber {
	return &mockEventSubscriber{
		handlers: make([]shared.EventHandler, 0),
	}
}

func (m *mockEventSubscriber) Subscribe(handler shared.EventHandler, eventTypes ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = append(m.handlers, handler)
}

func (m *mockEventSubscriber) Unsubscribe(handler shared.EventHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, h := range m.handlers {
		if h == handler {
			m.handlers = append(m.handlers[:i], m.handlers[i+1:]...)
			break
		}
	}
}

// mockDomainEvent implements shared.DomainEvent for testing
type mockDomainEvent struct {
	id            uuid.UUID
	eventType     string
	occurredAt    time.Time
	aggregateID   uuid.UUID
	aggregateType string
	tenantID      uuid.UUID
	documentType  string
	documentID    string
	jobID         string
	status        string
}

func (e *mockDomainEvent) EventID() uuid.UUID     { return e.id }
func (e *mockDomainEvent) EventType() string      { return e.eventType }
func (e *mockDomainEvent) OccurredAt() time.Time  { return e.occurredAt }
func (e *mockDomainEvent) AggregateID() uuid.UUID { return e.aggregateID }
func (e *mockDomainEvent) AggregateType() string  { return e.aggregateType }
func (e *mockDomainEvent) TenantID() uuid.UUID    { return e.tenantID }

func (e *mockDomainEvent) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"id":             e.id.String(),
		"type":           e.eventType,
		"timestamp":      e.occurredAt,
		"aggregate_id":   e.aggregateID.String(),
		"aggregate_type": e.aggregateType,
		"tenant_id":      e.tenantID.String(),
		"document_type":  e.documentType,
		"document_id":    e.documentID,
		"job_id":         e.jobID,
		"status":         e.status,
	})
}

// mockWSConn implements wsConn for testing
type mockWSConn struct {
	writtenData [][]byte
	readData    []byte
	readErr     error
	writeErr    error
	closed      bool
	mu          sync.Mutex
}

func newMockWSConn() *mockWSConn {
	return &mockWSConn{
		writtenData: make([][]byte, 0),
	}
}

func (c *mockWSConn) Read(b []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.readErr != nil {
		return 0, c.readErr
	}
	n := copy(b, c.readData)
	return n, nil
}

func (c *mockWSConn) Write(b []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.writeErr != nil {
		return 0, c.writeErr
	}
	data := make([]byte, len(b))
	copy(data, b)
	c.writtenData = append(c.writtenData, data)
	return len(b), nil
}

func (c *mockWSConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

func (c *mockWSConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *mockWSConn) SetWriteDeadline(t time.Time) error { return nil }

func (c *mockWSConn) getWrittenData() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.writtenData
}

func (c *mockWSConn) isClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

func TestNewPrintingWSHandler(t *testing.T) {
	eventBus := newMockEventSubscriber()

	handler := NewPrintingWSHandler(eventBus, nil)

	assert.NotNil(t, handler)
	assert.Equal(t, 100, handler.maxClients)
	assert.Equal(t, 30*time.Second, handler.heartbeat)
	assert.Equal(t, 60*time.Second, handler.pingTimeout)
}

func TestNewPrintingWSHandler_WithOptions(t *testing.T) {
	eventBus := newMockEventSubscriber()
	logger := zap.NewNop()

	handler := NewPrintingWSHandler(
		eventBus,
		nil,
		WithPrintingWSLogger(logger),
		WithPrintingWSHeartbeat(15*time.Second),
		WithPrintingWSPingTimeout(30*time.Second),
		WithPrintingWSMaxClients(50),
	)

	assert.NotNil(t, handler)
	assert.Equal(t, 50, handler.maxClients)
	assert.Equal(t, 15*time.Second, handler.heartbeat)
	assert.Equal(t, 30*time.Second, handler.pingTimeout)
}

func TestPrintingWSHandler_EventTypes(t *testing.T) {
	eventBus := newMockEventSubscriber()
	handler := NewPrintingWSHandler(eventBus, nil)

	eventTypes := handler.EventTypes()

	// Verify all expected event types are present
	expectedTypes := []string{
		"PrintJobCreated",
		"PrintJobStatusChanged",
		"PrintJobCompleted",
		"PrintJobFailed",
		"PrintTemplateCreated",
		"PrintTemplateUpdated",
		"PrintTemplateStatusChanged",
		"PrintTemplateSetAsDefault",
		"PrintTemplateDeleted",
		"AutoPrintRuleCreated",
		"AutoPrintRuleUpdated",
		"AutoPrintRuleEnabled",
		"AutoPrintRuleDisabled",
		"SalesOrderConfirmed",
		"SalesOrderShipped",
		"PurchaseOrderReceived",
		"ReceiptVoucherPaid",
	}

	assert.Equal(t, len(expectedTypes), len(eventTypes))
	for _, expected := range expectedTypes {
		assert.Contains(t, eventTypes, expected)
	}
}

func TestPrintingWSHandler_Start(t *testing.T) {
	eventBus := newMockEventSubscriber()
	handler := NewPrintingWSHandler(eventBus, nil)

	err := handler.Start()
	assert.NoError(t, err)
	assert.True(t, handler.started.Load())

	// Verify handler was subscribed
	assert.Len(t, eventBus.handlers, 1)

	// Starting again should return error
	err = handler.Start()
	assert.Error(t, err)

	handler.Stop()
}

func TestPrintingWSHandler_Stop(t *testing.T) {
	eventBus := newMockEventSubscriber()
	handler := NewPrintingWSHandler(eventBus, nil)

	err := handler.Start()
	require.NoError(t, err)

	handler.Stop()

	// Verify handler was unsubscribed
	assert.Len(t, eventBus.handlers, 0)
}

func TestPrintingWSHandler_AddRemoveClient(t *testing.T) {
	eventBus := newMockEventSubscriber()
	handler := NewPrintingWSHandler(eventBus, nil)

	tenantID := uuid.New().String()
	client := &PrintingWSClient{
		ID:       uuid.New().String(),
		UserID:   uuid.New().String(),
		TenantID: tenantID,
		conn:     newMockWSConn(),
		done:     make(chan struct{}),
		lastPing: time.Now(),
	}

	// Add client
	handler.addClient(client)
	assert.Equal(t, 1, handler.GetClientCount())
	assert.Equal(t, 1, handler.GetClientCountForTenant(tenantID))

	// Remove client
	handler.removeClient(client)
	assert.Equal(t, 0, handler.GetClientCount())
	assert.Equal(t, 0, handler.GetClientCountForTenant(tenantID))
}

func TestPrintingWSHandler_TenantIsolation(t *testing.T) {
	eventBus := newMockEventSubscriber()
	handler := NewPrintingWSHandler(eventBus, nil)

	tenant1ID := uuid.New().String()
	tenant2ID := uuid.New().String()

	client1 := &PrintingWSClient{
		ID:       uuid.New().String(),
		UserID:   uuid.New().String(),
		TenantID: tenant1ID,
		conn:     newMockWSConn(),
		done:     make(chan struct{}),
		lastPing: time.Now(),
	}

	client2 := &PrintingWSClient{
		ID:       uuid.New().String(),
		UserID:   uuid.New().String(),
		TenantID: tenant2ID,
		conn:     newMockWSConn(),
		done:     make(chan struct{}),
		lastPing: time.Now(),
	}

	handler.addClient(client1)
	handler.addClient(client2)

	assert.Equal(t, 2, handler.GetClientCount())
	assert.Equal(t, 1, handler.GetClientCountForTenant(tenant1ID))
	assert.Equal(t, 1, handler.GetClientCountForTenant(tenant2ID))

	// Verify getClientIDsForTenant returns correct clients
	tenant1Clients := handler.getClientIDsForTenant(tenant1ID)
	assert.Len(t, tenant1Clients, 1)
	assert.Equal(t, client1.ID, tenant1Clients[0])

	tenant2Clients := handler.getClientIDsForTenant(tenant2ID)
	assert.Len(t, tenant2Clients, 1)
	assert.Equal(t, client2.ID, tenant2Clients[0])

	// Cleanup
	handler.removeClient(client1)
	handler.removeClient(client2)
}

func TestPrintingWSHandler_Handle_NoClients(t *testing.T) {
	eventBus := newMockEventSubscriber()
	handler := NewPrintingWSHandler(eventBus, nil)

	event := &mockDomainEvent{
		id:            uuid.New(),
		eventType:     "PrintJobCreated",
		occurredAt:    time.Now(),
		aggregateID:   uuid.New(),
		aggregateType: "PrintJob",
		tenantID:      uuid.New(),
	}

	// Should not error when no clients
	err := handler.Handle(context.Background(), event)
	assert.NoError(t, err)
}

func TestPrintingWSHandler_Handle_WithClients(t *testing.T) {
	eventBus := newMockEventSubscriber()
	handler := NewPrintingWSHandler(eventBus, nil, WithPrintingWSLogger(zap.NewNop()))

	tenantID := uuid.New()
	mockConn := newMockWSConn()

	client := &PrintingWSClient{
		ID:       uuid.New().String(),
		UserID:   uuid.New().String(),
		TenantID: tenantID.String(),
		conn:     mockConn,
		done:     make(chan struct{}),
		lastPing: time.Now(),
	}

	handler.addClient(client)

	event := &mockDomainEvent{
		id:            uuid.New(),
		eventType:     "PrintJobCreated",
		occurredAt:    time.Now(),
		aggregateID:   uuid.New(),
		aggregateType: "PrintJob",
		tenantID:      tenantID,
		documentType:  "SALES_ORDER",
		documentID:    uuid.New().String(),
		jobID:         uuid.New().String(),
		status:        "PENDING",
	}

	err := handler.Handle(context.Background(), event)
	assert.NoError(t, err)

	// Give time for async send
	time.Sleep(100 * time.Millisecond)

	// Verify message was written (note: wsutil adds framing, so we check length > 0)
	writtenData := mockConn.getWrittenData()
	assert.Greater(t, len(writtenData), 0)

	// Cleanup
	handler.removeClient(client)
}

func TestPrintingWSHandler_Handle_DifferentTenant(t *testing.T) {
	eventBus := newMockEventSubscriber()
	handler := NewPrintingWSHandler(eventBus, nil)

	tenant1ID := uuid.New()
	tenant2ID := uuid.New()
	mockConn := newMockWSConn()

	// Client belongs to tenant1
	client := &PrintingWSClient{
		ID:       uuid.New().String(),
		UserID:   uuid.New().String(),
		TenantID: tenant1ID.String(),
		conn:     mockConn,
		done:     make(chan struct{}),
		lastPing: time.Now(),
	}

	handler.addClient(client)

	// Event is for tenant2
	event := &mockDomainEvent{
		id:            uuid.New(),
		eventType:     "PrintJobCreated",
		occurredAt:    time.Now(),
		aggregateID:   uuid.New(),
		aggregateType: "PrintJob",
		tenantID:      tenant2ID,
	}

	err := handler.Handle(context.Background(), event)
	assert.NoError(t, err)

	// Give time for async operations
	time.Sleep(50 * time.Millisecond)

	// Verify no message was written (different tenant)
	writtenData := mockConn.getWrittenData()
	assert.Len(t, writtenData, 0)

	// Cleanup
	handler.removeClient(client)
}

func TestPrintingWSHandler_BuildEventPayload(t *testing.T) {
	eventBus := newMockEventSubscriber()
	handler := NewPrintingWSHandler(eventBus, nil)

	eventID := uuid.New()
	aggregateID := uuid.New()
	tenantID := uuid.New()
	occurredAt := time.Now()

	event := &mockDomainEvent{
		id:            eventID,
		eventType:     "PrintJobCompleted",
		occurredAt:    occurredAt,
		aggregateID:   aggregateID,
		aggregateType: "PrintJob",
		tenantID:      tenantID,
		documentType:  "SALES_ORDER",
		documentID:    uuid.New().String(),
		jobID:         uuid.New().String(),
		status:        "COMPLETED",
	}

	payload := handler.buildEventPayload(event)

	assert.Equal(t, eventID.String(), payload.EventID)
	assert.Equal(t, "PrintJobCompleted", payload.EventType)
	assert.Equal(t, aggregateID.String(), payload.AggregateID)
	assert.Equal(t, "PrintJob", payload.AggregateType)
	assert.Equal(t, tenantID.String(), payload.TenantID)
	assert.Equal(t, "SALES_ORDER", payload.DocumentType)
	assert.Equal(t, "COMPLETED", payload.Status)
}

func TestPrintingWSHandler_CloseClient(t *testing.T) {
	eventBus := newMockEventSubscriber()
	handler := NewPrintingWSHandler(eventBus, nil)

	mockConn := newMockWSConn()
	client := &PrintingWSClient{
		ID:       uuid.New().String(),
		UserID:   uuid.New().String(),
		TenantID: uuid.New().String(),
		conn:     mockConn,
		done:     make(chan struct{}),
		lastPing: time.Now(),
	}

	handler.closeClient(client)

	// Verify done channel is closed
	select {
	case <-client.done:
		// Expected
	default:
		t.Error("done channel should be closed")
	}

	// Verify connection is closed
	assert.True(t, mockConn.isClosed())

	// Calling closeClient again should not panic
	handler.closeClient(client)
}

func TestPrintingWSHandler_UpdateClientPing(t *testing.T) {
	eventBus := newMockEventSubscriber()
	handler := NewPrintingWSHandler(eventBus, nil)

	oldTime := time.Now().Add(-1 * time.Hour)
	client := &PrintingWSClient{
		ID:       uuid.New().String(),
		UserID:   uuid.New().String(),
		TenantID: uuid.New().String(),
		conn:     newMockWSConn(),
		done:     make(chan struct{}),
		lastPing: oldTime,
	}

	handler.updateClientPing(client)

	client.pingMu.RLock()
	newPing := client.lastPing
	client.pingMu.RUnlock()

	assert.True(t, newPing.After(oldTime))
}

func TestPrintingWSHandler_HandleClientMessage_Ping(t *testing.T) {
	eventBus := newMockEventSubscriber()
	handler := NewPrintingWSHandler(eventBus, nil)

	mockConn := newMockWSConn()
	client := &PrintingWSClient{
		ID:       uuid.New().String(),
		UserID:   uuid.New().String(),
		TenantID: uuid.New().String(),
		conn:     mockConn,
		done:     make(chan struct{}),
		lastPing: time.Now().Add(-1 * time.Hour),
	}

	handler.addClient(client)

	pingMsg := WSMessage{
		Type:      "ping",
		Timestamp: time.Now().UnixMilli(),
	}
	data, _ := json.Marshal(pingMsg)

	handler.handleClientMessage(client, data)

	// Verify ping was updated
	client.pingMu.RLock()
	lastPing := client.lastPing
	client.pingMu.RUnlock()

	assert.True(t, time.Since(lastPing) < time.Second)

	// Verify pong was sent
	time.Sleep(50 * time.Millisecond)
	writtenData := mockConn.getWrittenData()
	assert.Greater(t, len(writtenData), 0)

	// Cleanup
	handler.removeClient(client)
}

func TestPrintingWSHandler_HandleClientMessage_Pong(t *testing.T) {
	eventBus := newMockEventSubscriber()
	handler := NewPrintingWSHandler(eventBus, nil)

	client := &PrintingWSClient{
		ID:       uuid.New().String(),
		UserID:   uuid.New().String(),
		TenantID: uuid.New().String(),
		conn:     newMockWSConn(),
		done:     make(chan struct{}),
		lastPing: time.Now().Add(-1 * time.Hour),
	}

	pongMsg := WSMessage{
		Type:      "pong",
		Timestamp: time.Now().UnixMilli(),
	}
	data, _ := json.Marshal(pongMsg)

	handler.handleClientMessage(client, data)

	// Verify ping was updated
	client.pingMu.RLock()
	lastPing := client.lastPing
	client.pingMu.RUnlock()

	assert.True(t, time.Since(lastPing) < time.Second)
}

func TestPrintingWSHandler_HandleClientMessage_InvalidJSON(t *testing.T) {
	eventBus := newMockEventSubscriber()
	logger := zap.NewNop()
	handler := NewPrintingWSHandler(eventBus, nil, WithPrintingWSLogger(logger))

	client := &PrintingWSClient{
		ID:       uuid.New().String(),
		UserID:   uuid.New().String(),
		TenantID: uuid.New().String(),
		conn:     newMockWSConn(),
		done:     make(chan struct{}),
		lastPing: time.Now(),
	}

	// Should not panic on invalid JSON
	handler.handleClientMessage(client, []byte("invalid json"))
}

func TestPrintingWSHandler_MaxClients(t *testing.T) {
	eventBus := newMockEventSubscriber()
	handler := NewPrintingWSHandler(eventBus, nil, WithPrintingWSMaxClients(2))

	// Add 2 clients (max)
	for i := 0; i < 2; i++ {
		client := &PrintingWSClient{
			ID:       uuid.New().String(),
			UserID:   uuid.New().String(),
			TenantID: uuid.New().String(),
			conn:     newMockWSConn(),
			done:     make(chan struct{}),
			lastPing: time.Now(),
		}
		handler.addClient(client)
	}

	assert.Equal(t, 2, handler.GetClientCount())
}

func TestPrintingWSHandler_TryAddClient(t *testing.T) {
	eventBus := newMockEventSubscriber()
	handler := NewPrintingWSHandler(eventBus, nil, WithPrintingWSMaxClients(2))

	// First client should succeed
	client1 := &PrintingWSClient{
		ID:       uuid.New().String(),
		UserID:   uuid.New().String(),
		TenantID: uuid.New().String(),
		conn:     newMockWSConn(),
		done:     make(chan struct{}),
		lastPing: time.Now(),
	}
	assert.True(t, handler.tryAddClient(client1))
	assert.Equal(t, 1, handler.GetClientCount())

	// Second client should succeed
	client2 := &PrintingWSClient{
		ID:       uuid.New().String(),
		UserID:   uuid.New().String(),
		TenantID: uuid.New().String(),
		conn:     newMockWSConn(),
		done:     make(chan struct{}),
		lastPing: time.Now(),
	}
	assert.True(t, handler.tryAddClient(client2))
	assert.Equal(t, 2, handler.GetClientCount())

	// Third client should fail (max reached)
	client3 := &PrintingWSClient{
		ID:       uuid.New().String(),
		UserID:   uuid.New().String(),
		TenantID: uuid.New().String(),
		conn:     newMockWSConn(),
		done:     make(chan struct{}),
		lastPing: time.Now(),
	}
	assert.False(t, handler.tryAddClient(client3))
	assert.Equal(t, 2, handler.GetClientCount())

	// Remove one client
	handler.removeClient(client1)
	assert.Equal(t, 1, handler.GetClientCount())

	// Now third client should succeed
	assert.True(t, handler.tryAddClient(client3))
	assert.Equal(t, 2, handler.GetClientCount())
}

func TestPrintingWSHandler_TryAddClient_NoLimit(t *testing.T) {
	eventBus := newMockEventSubscriber()
	handler := NewPrintingWSHandler(eventBus, nil, WithPrintingWSMaxClients(0)) // No limit

	// Should be able to add many clients
	for i := 0; i < 10; i++ {
		client := &PrintingWSClient{
			ID:       uuid.New().String(),
			UserID:   uuid.New().String(),
			TenantID: uuid.New().String(),
			conn:     newMockWSConn(),
			done:     make(chan struct{}),
			lastPing: time.Now(),
		}
		assert.True(t, handler.tryAddClient(client))
	}
	assert.Equal(t, 10, handler.GetClientCount())
}

func TestWSMessage_JSON(t *testing.T) {
	msg := WSMessage{
		Type:      "event",
		EventType: "PrintJobCreated",
		Data:      json.RawMessage(`{"job_id":"123"}`),
		Timestamp: 1234567890,
		ID:        "event-123",
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded WSMessage
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, msg.Type, decoded.Type)
	assert.Equal(t, msg.EventType, decoded.EventType)
	assert.Equal(t, msg.Timestamp, decoded.Timestamp)
	assert.Equal(t, msg.ID, decoded.ID)
}

func TestPrintingEventPayload_JSON(t *testing.T) {
	payload := PrintingEventPayload{
		EventID:        uuid.New().String(),
		EventType:      "PrintJobCompleted",
		AggregateID:    uuid.New().String(),
		AggregateType:  "PrintJob",
		TenantID:       uuid.New().String(),
		OccurredAt:     time.Now().Format(time.RFC3339),
		DocumentType:   "SALES_ORDER",
		DocumentID:     uuid.New().String(),
		DocumentNumber: "SO-2024-001",
		JobID:          uuid.New().String(),
		TemplateID:     uuid.New().String(),
		Status:         "COMPLETED",
		PdfURL:         "/api/v1/prints/xxx.pdf",
		Copies:         2,
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded PrintingEventPayload
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, payload.EventID, decoded.EventID)
	assert.Equal(t, payload.EventType, decoded.EventType)
	assert.Equal(t, payload.DocumentType, decoded.DocumentType)
	assert.Equal(t, payload.Status, decoded.Status)
	assert.Equal(t, payload.Copies, decoded.Copies)
}

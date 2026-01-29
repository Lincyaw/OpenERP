package csvimport

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EntityType represents supported entity types for import
type EntityType string

const (
	EntityProducts   EntityType = "products"
	EntityCustomers  EntityType = "customers"
	EntitySuppliers  EntityType = "suppliers"
	EntityInventory  EntityType = "inventory"
	EntityCategories EntityType = "categories"
)

// ValidEntityTypes returns all valid entity types
func ValidEntityTypes() []EntityType {
	return []EntityType{
		EntityProducts,
		EntityCustomers,
		EntitySuppliers,
		EntityInventory,
		EntityCategories,
	}
}

// IsValidEntityType checks if the entity type is valid
func IsValidEntityType(t string) bool {
	for _, valid := range ValidEntityTypes() {
		if string(valid) == t {
			return true
		}
	}
	return false
}

// ImportState represents the current state of an import session
type ImportState string

const (
	StateCreated    ImportState = "created"
	StateValidating ImportState = "validating"
	StateValidated  ImportState = "validated"
	StateImporting  ImportState = "importing"
	StateCompleted  ImportState = "completed"
	StateFailed     ImportState = "failed"
	StateCancelled  ImportState = "cancelled"
)

// ImportSession represents an import session with its state
type ImportSession struct {
	ID          uuid.UUID        `json:"id"`
	TenantID    uuid.UUID        `json:"tenant_id"`
	UserID      uuid.UUID        `json:"user_id"`
	EntityType  EntityType       `json:"entity_type"`
	FileName    string           `json:"file_name"`
	FileSize    int64            `json:"file_size"`
	State       ImportState      `json:"state"`
	TotalRows   int              `json:"total_rows"`
	ValidRows   int              `json:"valid_rows"`
	ErrorRows   int              `json:"error_rows"`
	Errors      []RowError       `json:"errors,omitempty"`
	Preview     []map[string]any `json:"preview,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	CompletedAt *time.Time       `json:"completed_at,omitempty"`
}

// NewImportSession creates a new import session
func NewImportSession(tenantID, userID uuid.UUID, entityType EntityType, fileName string, fileSize int64) *ImportSession {
	now := time.Now()
	return &ImportSession{
		ID:         uuid.New(),
		TenantID:   tenantID,
		UserID:     userID,
		EntityType: entityType,
		FileName:   fileName,
		FileSize:   fileSize,
		State:      StateCreated,
		CreatedAt:  now,
		UpdatedAt:  now,
		Preview:    make([]map[string]any, 0),
		Errors:     make([]RowError, 0),
	}
}

// UpdateState updates the session state
func (s *ImportSession) UpdateState(state ImportState) {
	s.State = state
	s.UpdatedAt = time.Now()
	if state == StateCompleted || state == StateFailed || state == StateCancelled {
		now := time.Now()
		s.CompletedAt = &now
	}
}

// SetValidationResult sets the validation result
func (s *ImportSession) SetValidationResult(result *ValidationResult) {
	s.TotalRows = result.TotalRows
	s.ValidRows = result.ValidRows
	s.ErrorRows = result.ErrorRows
	s.Errors = result.Errors
	s.Preview = result.Preview
	s.UpdatedAt = time.Now()
}

// IsValid returns true if the session has no errors
func (s *ImportSession) IsValid() bool {
	return s.ErrorRows == 0
}

// ImportContext manages the context for an import operation
type ImportContext struct {
	ctx             context.Context
	cancel          context.CancelFunc
	session         *ImportSession
	parser          *CSVParser
	fieldValidator  *FieldValidator
	refValidator    *ReferenceValidator
	uniqueValidator *UniquenessValidator
	errors          *ErrorCollection
	validRows       []*Row
	errorRowNums    map[int]bool
	mu              sync.RWMutex
}

// ImportContextOption is a functional option for ImportContext
type ImportContextOption func(*ImportContext)

// WithFieldValidator sets the field validator
func WithFieldValidator(v *FieldValidator) ImportContextOption {
	return func(ic *ImportContext) {
		ic.fieldValidator = v
	}
}

// WithReferenceValidator sets the reference validator
func WithReferenceValidator(v *ReferenceValidator) ImportContextOption {
	return func(ic *ImportContext) {
		ic.refValidator = v
	}
}

// WithUniquenessValidator sets the uniqueness validator
func WithUniquenessValidator(v *UniquenessValidator) ImportContextOption {
	return func(ic *ImportContext) {
		ic.uniqueValidator = v
	}
}

// NewImportContext creates a new import context
func NewImportContext(ctx context.Context, session *ImportSession, opts ...ImportContextOption) *ImportContext {
	ctxWithCancel, cancel := context.WithCancel(ctx)

	ic := &ImportContext{
		ctx:          ctxWithCancel,
		cancel:       cancel,
		session:      session,
		errors:       NewErrorCollection(100),
		validRows:    make([]*Row, 0),
		errorRowNums: make(map[int]bool),
	}

	for _, opt := range opts {
		opt(ic)
	}

	return ic
}

// Context returns the context
func (ic *ImportContext) Context() context.Context {
	return ic.ctx
}

// Cancel cancels the import context
func (ic *ImportContext) Cancel() {
	ic.cancel()
	ic.session.UpdateState(StateCancelled)
}

// Session returns the import session
func (ic *ImportContext) Session() *ImportSession {
	return ic.session
}

// Parser returns the CSV parser
func (ic *ImportContext) Parser() *CSVParser {
	return ic.parser
}

// SetParser sets the CSV parser
func (ic *ImportContext) SetParser(p *CSVParser) {
	ic.parser = p
}

// ValidRows returns the valid rows
func (ic *ImportContext) ValidRows() []*Row {
	ic.mu.RLock()
	defer ic.mu.RUnlock()
	return ic.validRows
}

// AddValidRow adds a valid row
func (ic *ImportContext) AddValidRow(row *Row) {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	ic.validRows = append(ic.validRows, row)
}

// MarkRowError marks a row as having an error
func (ic *ImportContext) MarkRowError(rowNum int) {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	ic.errorRowNums[rowNum] = true
}

// HasRowError checks if a row has an error
func (ic *ImportContext) HasRowError(rowNum int) bool {
	ic.mu.RLock()
	defer ic.mu.RUnlock()
	return ic.errorRowNums[rowNum]
}

// ErrorCount returns the number of error rows
func (ic *ImportContext) ErrorCount() int {
	ic.mu.RLock()
	defer ic.mu.RUnlock()
	return len(ic.errorRowNums)
}

// Errors returns the error collection
func (ic *ImportContext) Errors() *ErrorCollection {
	return ic.errors
}

// ImportProcessor processes a CSV import with validation
type ImportProcessor struct {
	maxFileSize     int64
	maxRows         int
	maxErrors       int
	previewRows     int
	referenceLookup func(refType, value string) (bool, error)
	uniqueLookup    func(entityType, field, value string) (bool, error)
}

// ProcessorOption is a functional option for ImportProcessor
type ProcessorOption func(*ImportProcessor)

// WithMaxFileSize sets the maximum file size
func WithMaxFileSize(size int64) ProcessorOption {
	return func(p *ImportProcessor) {
		p.maxFileSize = size
	}
}

// WithMaxRows sets the maximum number of rows
func WithMaxRows(rows int) ProcessorOption {
	return func(p *ImportProcessor) {
		p.maxRows = rows
	}
}

// WithMaxErrors sets the maximum number of errors to collect
func WithMaxErrors(errors int) ProcessorOption {
	return func(p *ImportProcessor) {
		p.maxErrors = errors
	}
}

// WithPreviewRows sets the number of preview rows
func WithPreviewRows(rows int) ProcessorOption {
	return func(p *ImportProcessor) {
		p.previewRows = rows
	}
}

// WithReferenceLookup sets the reference lookup function
func WithReferenceLookup(fn func(refType, value string) (bool, error)) ProcessorOption {
	return func(p *ImportProcessor) {
		p.referenceLookup = fn
	}
}

// WithUniqueLookup sets the uniqueness lookup function
func WithUniqueLookup(fn func(entityType, field, value string) (bool, error)) ProcessorOption {
	return func(p *ImportProcessor) {
		p.uniqueLookup = fn
	}
}

// NewImportProcessor creates a new import processor
func NewImportProcessor(opts ...ProcessorOption) *ImportProcessor {
	p := &ImportProcessor{
		maxFileSize: 10 * 1024 * 1024, // 10MB default
		maxRows:     100000,           // 100K rows default
		maxErrors:   100,              // 100 errors default
		previewRows: 5,                // 5 preview rows default
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Validate validates a CSV file without importing
func (p *ImportProcessor) Validate(ctx context.Context, session *ImportSession, reader io.Reader, rules []FieldRule) (*ValidationResult, error) {
	// Update session state
	session.UpdateState(StateValidating)

	// Create parser
	parser, err := NewCSVParser(reader)
	if err != nil {
		session.UpdateState(StateFailed)
		return nil, err
	}

	// Parse header
	if err := parser.ParseHeader(); err != nil {
		session.UpdateState(StateFailed)
		return nil, err
	}

	// Create validators
	fieldValidator := NewFieldValidator(rules, p.maxErrors)

	var refValidator *ReferenceValidator
	if p.referenceLookup != nil {
		refValidator = NewReferenceValidator(p.referenceLookup, p.maxErrors)
	}

	var uniqueValidator *UniquenessValidator
	if p.uniqueLookup != nil {
		uniqueValidator = NewUniquenessValidator(p.uniqueLookup, p.maxErrors)
	}

	// Create import context
	importCtx := NewImportContext(ctx, session,
		WithFieldValidator(fieldValidator),
		WithReferenceValidator(refValidator),
		WithUniquenessValidator(uniqueValidator),
	)
	importCtx.SetParser(parser)

	// Process rows
	result := NewValidationResult(session.ID.String())
	totalRows := 0
	validRows := 0
	errorRows := 0

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			session.UpdateState(StateCancelled)
			return nil, ctx.Err()
		default:
		}

		row, err := parser.ReadRow()
		if err == io.EOF {
			break
		}
		if err != nil {
			importCtx.Errors().Add(NewRowError(parser.CurrentRow(), "", ErrCodeImportCSVParsing, err.Error()))
			errorRows++
			continue
		}

		// Skip empty rows
		if row.IsEmpty() {
			continue
		}

		totalRows++

		// Check max rows
		if totalRows > p.maxRows {
			importCtx.Errors().Add(NewRowError(row.LineNumber, "", ErrCodeImportValidation,
				"exceeded maximum number of rows"))
			break
		}

		// Validate row
		hasError := false

		// Field validation
		if !fieldValidator.ValidateRow(row) {
			hasError = true
		}

		// Reference validation
		if refValidator != nil {
			for _, rule := range rules {
				if rule.Reference != "" {
					value := row.Get(rule.Column)
					if !refValidator.ValidateReference(row.LineNumber, rule.Column, rule.Reference, value) {
						hasError = true
					}
				}
			}
		}

		// Uniqueness validation
		if uniqueValidator != nil {
			for _, rule := range rules {
				if rule.Unique {
					value := row.Get(rule.Column)
					if !uniqueValidator.ValidateUnique(row.LineNumber, rule.Column, string(session.EntityType), value) {
						hasError = true
					}
				}
			}
		}

		if hasError {
			errorRows++
			importCtx.MarkRowError(row.LineNumber)
		} else {
			validRows++
			importCtx.AddValidRow(row)

			// Add to preview (first N valid rows)
			if len(result.Preview) < p.previewRows {
				// Convert map[string]string to map[string]any
				previewRow := make(map[string]any, len(row.Data))
				for k, v := range row.Data {
					previewRow[k] = v
				}
				result.AddPreview(previewRow)
			}
		}
	}

	// Collect all errors
	allErrors := NewErrorCollection(p.maxErrors)
	for _, e := range importCtx.Errors().Errors() {
		allErrors.Add(e)
	}
	for _, e := range fieldValidator.Errors().Errors() {
		allErrors.Add(e)
	}
	if refValidator != nil {
		for _, e := range refValidator.Errors().Errors() {
			allErrors.Add(e)
		}
	}
	if uniqueValidator != nil {
		for _, e := range uniqueValidator.Errors().Errors() {
			allErrors.Add(e)
		}
	}

	// Set result
	result.SetCounts(totalRows, validRows, errorRows)
	result.SetErrors(allErrors)

	// Update session
	session.SetValidationResult(result)
	if errorRows > 0 {
		session.UpdateState(StateFailed)
	} else {
		session.UpdateState(StateValidated)
	}

	return result, nil
}

// SessionStore interface for storing import sessions
type SessionStore interface {
	Save(session *ImportSession) error
	Get(id uuid.UUID) (*ImportSession, error)
	GetByTenant(tenantID uuid.UUID, limit int) ([]*ImportSession, error)
	Delete(id uuid.UUID) error
}

// InMemorySessionStore is an in-memory implementation of SessionStore
type InMemorySessionStore struct {
	sessions map[uuid.UUID]*ImportSession
	mu       sync.RWMutex
	ttl      time.Duration
	stopCh   chan struct{}
}

// NewInMemorySessionStore creates a new in-memory session store
func NewInMemorySessionStore(ttl time.Duration) *InMemorySessionStore {
	store := &InMemorySessionStore{
		sessions: make(map[uuid.UUID]*ImportSession),
		ttl:      ttl,
		stopCh:   make(chan struct{}),
	}
	// Start background cleanup goroutine
	go store.startCleanupLoop()
	return store
}

// startCleanupLoop periodically removes expired sessions
func (s *InMemorySessionStore) startCleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.Cleanup()
		case <-s.stopCh:
			return
		}
	}
}

// Stop stops the background cleanup goroutine
func (s *InMemorySessionStore) Stop() {
	close(s.stopCh)
}

// Save saves a session
func (s *InMemorySessionStore) Save(session *ImportSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	return nil
}

// Get retrieves a session by ID
func (s *InMemorySessionStore) Get(id uuid.UUID) (*ImportSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[id]
	if !ok {
		return nil, nil
	}
	// Check TTL
	if time.Since(session.CreatedAt) > s.ttl {
		return nil, nil
	}
	return session, nil
}

// GetByTenant retrieves sessions by tenant ID
func (s *InMemorySessionStore) GetByTenant(tenantID uuid.UUID, limit int) ([]*ImportSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*ImportSession
	for _, session := range s.sessions {
		if session.TenantID == tenantID && time.Since(session.CreatedAt) <= s.ttl {
			result = append(result, session)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

// Delete deletes a session by ID
func (s *InMemorySessionStore) Delete(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
	return nil
}

// Cleanup removes expired sessions
func (s *InMemorySessionStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, session := range s.sessions {
		if time.Since(session.CreatedAt) > s.ttl {
			delete(s.sessions, id)
		}
	}
}

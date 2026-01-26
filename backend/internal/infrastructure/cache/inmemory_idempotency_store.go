package cache

import (
	"context"
	"sync"
	"time"

	"github.com/erp/backend/internal/domain/shared"
)

// entry represents a stored event ID with expiration
type entry struct {
	expiresAt time.Time
}

// InMemoryIdempotencyStore implements IdempotencyStore using an in-memory map
// This is suitable for single-instance deployments and testing
type InMemoryIdempotencyStore struct {
	mu        sync.RWMutex
	entries   map[string]entry
	stopChan  chan struct{}
	wg        sync.WaitGroup
	closeOnce sync.Once
}

// NewInMemoryIdempotencyStore creates a new in-memory idempotency store
// It starts a background goroutine to clean up expired entries
func NewInMemoryIdempotencyStore() *InMemoryIdempotencyStore {
	store := &InMemoryIdempotencyStore{
		entries:  make(map[string]entry),
		stopChan: make(chan struct{}),
	}

	// Start cleanup goroutine
	store.wg.Add(1)
	go store.cleanupLoop()

	return store
}

// MarkProcessed marks an event as processed with a TTL
// Returns true if the event was newly marked, false if it was already processed
func (s *InMemoryIdempotencyStore) MarkProcessed(ctx context.Context, eventID string, ttl time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already exists and not expired
	if e, exists := s.entries[eventID]; exists {
		if time.Now().Before(e.expiresAt) {
			return false, nil // Already processed
		}
		// Entry exists but expired, will be overwritten
	}

	// Mark as processed
	s.entries[eventID] = entry{
		expiresAt: time.Now().Add(ttl),
	}

	return true, nil
}

// IsProcessed checks if an event has already been processed
func (s *InMemoryIdempotencyStore) IsProcessed(ctx context.Context, eventID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e, exists := s.entries[eventID]
	if !exists {
		return false, nil
	}

	// Check if entry has expired
	if time.Now().After(e.expiresAt) {
		return false, nil // Expired, treat as not processed
	}

	return true, nil
}

// Close stops the cleanup goroutine and releases resources
// Safe to call multiple times
func (s *InMemoryIdempotencyStore) Close() error {
	s.closeOnce.Do(func() {
		close(s.stopChan)
		s.wg.Wait()
	})
	return nil
}

// cleanupLoop periodically removes expired entries
func (s *InMemoryIdempotencyStore) cleanupLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.cleanup()
		}
	}
}

// cleanup removes expired entries from the store
func (s *InMemoryIdempotencyStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for eventID, e := range s.entries {
		if now.After(e.expiresAt) {
			delete(s.entries, eventID)
		}
	}
}

// Size returns the number of entries in the store (for testing/monitoring)
func (s *InMemoryIdempotencyStore) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

// Ensure InMemoryIdempotencyStore implements IdempotencyStore
var _ shared.IdempotencyStore = (*InMemoryIdempotencyStore)(nil)

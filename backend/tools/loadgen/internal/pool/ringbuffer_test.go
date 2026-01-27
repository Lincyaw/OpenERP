package pool

import (
	"sync"
	"testing"
	"time"
)

func TestRingBufferBasicOperations(t *testing.T) {
	rb := NewRingBuffer(5, EvictionFIFO)

	// Test empty buffer
	if !rb.IsEmpty() {
		t.Error("New buffer should be empty")
	}

	if rb.IsFull() {
		t.Error("New buffer should not be full")
	}

	// Test Add
	v1 := NewParameterValue("value1", SemanticTypeCustomerID, 0)
	evicted := rb.Add(v1)

	if evicted != 0 {
		t.Errorf("Evicted = %d, want 0", evicted)
	}

	if rb.Count() != 1 {
		t.Errorf("Count = %d, want 1", rb.Count())
	}

	// Test Get
	got := rb.Get()
	if got != v1 {
		t.Error("Get should return the added value")
	}
}

func TestRingBufferFIFOEviction(t *testing.T) {
	rb := NewRingBuffer(3, EvictionFIFO)

	v1 := NewParameterValue("value1", SemanticTypeCustomerID, 0)
	v2 := NewParameterValue("value2", SemanticTypeCustomerID, 0)
	v3 := NewParameterValue("value3", SemanticTypeCustomerID, 0)
	v4 := NewParameterValue("value4", SemanticTypeCustomerID, 0)

	rb.Add(v1)
	rb.Add(v2)
	rb.Add(v3)

	if rb.Count() != 3 {
		t.Errorf("Count = %d, want 3", rb.Count())
	}

	// Adding fourth value should evict first
	evicted := rb.Add(v4)

	if evicted != 1 {
		t.Errorf("Evicted = %d, want 1", evicted)
	}

	if rb.Count() != 3 {
		t.Errorf("Count after eviction = %d, want 3", rb.Count())
	}

	if rb.EvictionCount() != 1 {
		t.Errorf("EvictionCount = %d, want 1", rb.EvictionCount())
	}

	// v1 should be gone
	all := rb.GetAll()
	for _, v := range all {
		if v == v1 {
			t.Error("v1 should have been evicted")
		}
	}
}

func TestRingBufferLRUEviction(t *testing.T) {
	rb := NewRingBuffer(3, EvictionLRU)

	v1 := NewParameterValue("value1", SemanticTypeCustomerID, 0)
	v2 := NewParameterValue("value2", SemanticTypeCustomerID, 0)
	v3 := NewParameterValue("value3", SemanticTypeCustomerID, 0)

	rb.Add(v1)
	rb.Add(v2)
	rb.Add(v3)

	if rb.Count() != 3 {
		t.Errorf("Count = %d, want 3", rb.Count())
	}

	// Access v1 to update its LRU order
	time.Sleep(time.Millisecond)
	rb.Get() // Gets v1 (first in FIFO order)

	// Add v4 - should evict the least recently used
	v4 := NewParameterValue("value4", SemanticTypeCustomerID, 0)
	evicted := rb.Add(v4)

	if evicted != 1 {
		t.Errorf("Evicted = %d, want 1", evicted)
	}

	if rb.Count() != 3 {
		t.Errorf("Count after eviction = %d, want 3", rb.Count())
	}

	// Verify eviction count
	if rb.EvictionCount() != 1 {
		t.Errorf("EvictionCount = %d, want 1", rb.EvictionCount())
	}
}

func TestRingBufferRandomEviction(t *testing.T) {
	rb := NewRingBuffer(3, EvictionRandom)

	v1 := NewParameterValue("value1", SemanticTypeCustomerID, 0)
	v2 := NewParameterValue("value2", SemanticTypeCustomerID, 0)
	v3 := NewParameterValue("value3", SemanticTypeCustomerID, 0)
	v4 := NewParameterValue("value4", SemanticTypeCustomerID, 0)

	rb.Add(v1)
	rb.Add(v2)
	rb.Add(v3)

	if rb.Count() != 3 {
		t.Errorf("Count before eviction = %d, want 3", rb.Count())
	}

	evicted := rb.Add(v4)

	if evicted != 1 {
		t.Errorf("Evicted = %d, want 1", evicted)
	}

	if rb.Count() != 3 {
		t.Errorf("Count after eviction = %d, want 3", rb.Count())
	}

	// Verify eviction count
	if rb.EvictionCount() != 1 {
		t.Errorf("EvictionCount = %d, want 1", rb.EvictionCount())
	}
}

func TestRingBufferGetRandom(t *testing.T) {
	rb := NewRingBuffer(10, EvictionFIFO)

	// Empty buffer returns nil
	if rb.GetRandom() != nil {
		t.Error("GetRandom on empty buffer should return nil")
	}

	// Add some values
	for i := 0; i < 5; i++ {
		v := NewParameterValue(i, SemanticTypeCustomerID, 0)
		rb.Add(v)
	}

	// GetRandom should return a value
	got := rb.GetRandom()
	if got == nil {
		t.Error("GetRandom should return a value")
	}

	// Multiple calls should update access statistics
	initialCount := got.AccessCount()
	for range 10 {
		rb.GetRandom()
	}

	// At least some values should have been accessed
	all := rb.GetAll()
	totalAccess := int64(0)
	for _, v := range all {
		totalAccess += v.AccessCount()
	}
	if totalAccess <= initialCount {
		t.Error("GetRandom should update access counts")
	}
}

func TestRingBufferRemove(t *testing.T) {
	rb := NewRingBuffer(5, EvictionFIFO)

	v1 := NewParameterValue("value1", SemanticTypeCustomerID, 0)
	v2 := NewParameterValue("value2", SemanticTypeCustomerID, 0)

	rb.Add(v1)
	rb.Add(v2)

	// Remove v1
	removed := rb.Remove(v1)
	if !removed {
		t.Error("Remove should return true for existing value")
	}

	if rb.Count() != 1 {
		t.Errorf("Count = %d, want 1", rb.Count())
	}

	// Try to remove again
	removed = rb.Remove(v1)
	if removed {
		t.Error("Remove should return false for non-existing value")
	}
}

func TestRingBufferClear(t *testing.T) {
	rb := NewRingBuffer(5, EvictionFIFO)

	for i := 0; i < 5; i++ {
		v := NewParameterValue(i, SemanticTypeCustomerID, 0)
		rb.Add(v)
	}

	cleared := rb.Clear()
	if cleared != 5 {
		t.Errorf("Cleared = %d, want 5", cleared)
	}

	if !rb.IsEmpty() {
		t.Error("Buffer should be empty after clear")
	}
}

func TestRingBufferRemoveExpired(t *testing.T) {
	rb := NewRingBuffer(5, EvictionFIFO)

	// Add some values with short TTL
	v1 := NewParameterValue("value1", SemanticTypeCustomerID, time.Millisecond)
	v2 := NewParameterValue("value2", SemanticTypeCustomerID, time.Hour)
	v3 := NewParameterValue("value3", SemanticTypeCustomerID, time.Millisecond)

	rb.Add(v1)
	rb.Add(v2)
	rb.Add(v3)

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	removed := rb.RemoveExpired()
	if removed != 2 {
		t.Errorf("RemoveExpired = %d, want 2", removed)
	}

	if rb.Count() != 1 {
		t.Errorf("Count = %d, want 1", rb.Count())
	}
}

func TestRingBufferConcurrency(t *testing.T) {
	rb := NewRingBuffer(100, EvictionFIFO)

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	// Concurrent adds
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				v := NewParameterValue(id*1000+j, SemanticTypeCustomerID, 0)
				rb.Add(v)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				rb.Get()
				rb.GetRandom()
				rb.Count()
			}
		}()
	}

	wg.Wait()

	// Buffer should not be corrupted
	if rb.Count() > rb.Capacity() {
		t.Errorf("Count (%d) exceeds capacity (%d)", rb.Count(), rb.Capacity())
	}
}

func TestRingBufferCapacity(t *testing.T) {
	rb := NewRingBuffer(10, EvictionFIFO)

	if rb.Capacity() != 10 {
		t.Errorf("Capacity = %d, want 10", rb.Capacity())
	}
}

func TestNewRingBufferDefaults(t *testing.T) {
	// Zero capacity should use default
	rb := NewRingBuffer(0, EvictionFIFO)
	if rb.Capacity() != 1000 {
		t.Errorf("Default capacity = %d, want 1000", rb.Capacity())
	}

	// Negative capacity should use default
	rb = NewRingBuffer(-5, EvictionFIFO)
	if rb.Capacity() != 1000 {
		t.Errorf("Negative capacity = %d, want 1000", rb.Capacity())
	}
}

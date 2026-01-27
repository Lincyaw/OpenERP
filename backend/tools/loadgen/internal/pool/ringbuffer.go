package pool

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// RingBuffer is a thread-safe circular buffer for storing ParameterValues.
// It automatically evicts the oldest values when full (FIFO), or can be configured
// for LRU or Random eviction policies.
type RingBuffer struct {
	mu       sync.RWMutex
	items    []*ParameterValue
	head     int // next write position
	tail     int // next read position (for FIFO Get)
	count    int
	capacity int

	evictionPolicy EvictionPolicy
	evictionCount  atomic.Int64

	// For LRU tracking
	accessOrder []int // indices sorted by access time (oldest first)

	// Random source
	rng *rand.Rand
}

// NewRingBuffer creates a new RingBuffer with the given capacity and eviction policy.
func NewRingBuffer(capacity int, policy EvictionPolicy) *RingBuffer {
	if capacity <= 0 {
		capacity = 1000 // default capacity
	}
	return &RingBuffer{
		items:          make([]*ParameterValue, capacity),
		capacity:       capacity,
		evictionPolicy: policy,
		accessOrder:    make([]int, 0, capacity),
		rng:            rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Add adds a value to the buffer, evicting old values if necessary.
// Returns the number of values evicted.
func (rb *RingBuffer) Add(value *ParameterValue) int {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	evicted := 0

	// If buffer is full, evict according to policy
	if rb.count >= rb.capacity {
		evicted = rb.evictOne()
	}

	// Add the new value at head position
	rb.items[rb.head] = value
	if rb.evictionPolicy == EvictionLRU {
		rb.accessOrder = append(rb.accessOrder, rb.head)
	}
	rb.head = (rb.head + 1) % rb.capacity
	rb.count++

	return evicted
}

// evictOne removes one value according to the eviction policy.
// Must be called with lock held.
func (rb *RingBuffer) evictOne() int {
	if rb.count == 0 {
		return 0
	}

	var evictIdx int

	switch rb.evictionPolicy {
	case EvictionFIFO:
		// Evict from tail (oldest)
		evictIdx = rb.tail
		rb.tail = (rb.tail + 1) % rb.capacity

	case EvictionLRU:
		// Evict the least recently accessed
		if len(rb.accessOrder) > 0 {
			evictIdx = rb.accessOrder[0]
			rb.accessOrder = rb.accessOrder[1:]
			// If this position is the tail, move tail forward
			if evictIdx == rb.tail {
				rb.tail = (rb.tail + 1) % rb.capacity
			}
		} else {
			evictIdx = rb.tail
			rb.tail = (rb.tail + 1) % rb.capacity
		}

	case EvictionRandom:
		// Evict a random value
		evictIdx = rb.findRandomOccupiedIndex()
		if evictIdx == rb.tail {
			rb.tail = (rb.tail + 1) % rb.capacity
		}
	}

	rb.items[evictIdx] = nil
	rb.count--
	rb.evictionCount.Add(1)

	return 1
}

// findRandomOccupiedIndex finds a random index that contains a value.
// Must be called with lock held and count > 0.
func (rb *RingBuffer) findRandomOccupiedIndex() int {
	// Simple approach: pick random offset from tail
	offset := rb.rng.Intn(rb.count)
	idx := (rb.tail + offset) % rb.capacity

	// Linear search from the random position to find an occupied slot
	for i := 0; i < rb.capacity; i++ {
		checkIdx := (idx + i) % rb.capacity
		if rb.items[checkIdx] != nil {
			return checkIdx
		}
	}
	return rb.tail // fallback
}

// Get retrieves the next value in FIFO order without removing it.
// Returns nil if the buffer is empty.
func (rb *RingBuffer) Get() *ParameterValue {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count == 0 {
		return nil
	}

	// Find next non-nil value from tail
	for i := 0; i < rb.capacity; i++ {
		idx := (rb.tail + i) % rb.capacity
		if rb.items[idx] != nil {
			value := rb.items[idx]
			value.Touch()
			rb.updateLRUAccess(idx)
			return value
		}
	}
	return nil
}

// GetRandom retrieves a random value from the buffer without removing it.
// Returns nil if the buffer is empty.
func (rb *RingBuffer) GetRandom() *ParameterValue {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count == 0 {
		return nil
	}

	// Pick a random starting point and find the next non-nil value
	start := rb.rng.Intn(rb.capacity)
	for i := 0; i < rb.capacity; i++ {
		idx := (start + i) % rb.capacity
		if rb.items[idx] != nil {
			value := rb.items[idx]
			value.Touch()
			rb.updateLRUAccess(idx)
			return value
		}
	}
	return nil
}

// updateLRUAccess moves the given index to the end of the access order.
// Must be called with lock held.
func (rb *RingBuffer) updateLRUAccess(idx int) {
	if rb.evictionPolicy != EvictionLRU {
		return
	}

	// Remove idx from current position and append to end
	for i, accessIdx := range rb.accessOrder {
		if accessIdx == idx {
			rb.accessOrder = append(rb.accessOrder[:i], rb.accessOrder[i+1:]...)
			break
		}
	}
	rb.accessOrder = append(rb.accessOrder, idx)
}

// GetAll returns all non-nil values in the buffer.
func (rb *RingBuffer) GetAll() []*ParameterValue {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	result := make([]*ParameterValue, 0, rb.count)
	for _, item := range rb.items {
		if item != nil {
			result = append(result, item)
		}
	}
	return result
}

// Count returns the number of values in the buffer.
func (rb *RingBuffer) Count() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count
}

// Capacity returns the maximum capacity of the buffer.
func (rb *RingBuffer) Capacity() int {
	return rb.capacity
}

// EvictionCount returns the total number of values that have been evicted.
func (rb *RingBuffer) EvictionCount() int64 {
	return rb.evictionCount.Load()
}

// Remove removes a specific value from the buffer.
// Returns true if the value was found and removed.
func (rb *RingBuffer) Remove(value *ParameterValue) bool {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	for i, item := range rb.items {
		if item == value {
			rb.items[i] = nil
			rb.count--

			// Update LRU access order
			if rb.evictionPolicy == EvictionLRU {
				for j, accessIdx := range rb.accessOrder {
					if accessIdx == i {
						rb.accessOrder = append(rb.accessOrder[:j], rb.accessOrder[j+1:]...)
						break
					}
				}
			}
			return true
		}
	}
	return false
}

// Clear removes all values from the buffer.
// Returns the number of values removed.
func (rb *RingBuffer) Clear() int {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	removed := rb.count
	for i := range rb.items {
		rb.items[i] = nil
	}
	rb.head = 0
	rb.tail = 0
	rb.count = 0
	rb.accessOrder = rb.accessOrder[:0]

	return removed
}

// RemoveExpired removes all expired values from the buffer.
// Returns the number of values removed.
func (rb *RingBuffer) RemoveExpired() int {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	removed := 0
	for i, item := range rb.items {
		if item != nil && item.IsExpired() {
			rb.items[i] = nil
			rb.count--
			removed++

			// Update LRU access order
			if rb.evictionPolicy == EvictionLRU {
				for j, accessIdx := range rb.accessOrder {
					if accessIdx == i {
						rb.accessOrder = append(rb.accessOrder[:j], rb.accessOrder[j+1:]...)
						break
					}
				}
			}
		}
	}
	return removed
}

// IsFull returns true if the buffer is at capacity.
func (rb *RingBuffer) IsFull() bool {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count >= rb.capacity
}

// IsEmpty returns true if the buffer has no values.
func (rb *RingBuffer) IsEmpty() bool {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count == 0
}

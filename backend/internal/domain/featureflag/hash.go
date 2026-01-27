package featureflag

import (
	"encoding/binary"
	"hash/fnv"
)

// HashBucketCount is the number of buckets for consistent hashing (0-99)
const HashBucketCount = 100

// ComputeHashBucket computes a consistent hash bucket for the given flag key and user ID.
// Uses MurmurHash3 algorithm to produce a bucket number between 0 and 99.
// The same combination of flagKey and userID will always produce the same bucket number,
// ensuring that users consistently receive the same variant/percentage value.
//
// This is essential for percentage-based rollouts and A/B testing to ensure
// users don't flip-flop between variants on each request.
func ComputeHashBucket(flagKey, userID string) int {
	// Combine flag key and user ID for the hash input
	// This ensures that different flags produce different results for the same user
	hashInput := flagKey + ":" + userID

	// Use MurmurHash3 algorithm for consistent hashing
	h := murmur3Hash32([]byte(hashInput), 0)

	// Map the hash to a bucket (0-99)
	return int(h % HashBucketCount)
}

// ComputeHashBucketWithSeed computes a consistent hash bucket with a custom seed.
// Useful for scenarios where you need multiple independent hash buckets for the same user/flag.
func ComputeHashBucketWithSeed(flagKey, userID string, seed uint32) int {
	hashInput := flagKey + ":" + userID
	h := murmur3Hash32([]byte(hashInput), seed)
	return int(h % HashBucketCount)
}

// IsInPercentage checks if a user falls within the given percentage for a flag.
// percentage should be 0-100. Returns true if the user should be included.
func IsInPercentage(flagKey, userID string, percentage int) bool {
	if percentage <= 0 {
		return false
	}
	if percentage >= 100 {
		return true
	}
	bucket := ComputeHashBucket(flagKey, userID)
	return bucket < percentage
}

// GetVariantBucket returns which variant bucket a user falls into.
// variantCount is the number of variants (e.g., 2 for A/B, 3 for A/B/C).
// Returns a value from 0 to variantCount-1.
func GetVariantBucket(flagKey, userID string, variantCount int) int {
	if variantCount <= 1 {
		return 0
	}

	hashInput := flagKey + ":variant:" + userID
	h := murmur3Hash32([]byte(hashInput), 0)
	return int(h % uint32(variantCount))
}

// SelectVariant selects a variant from a list based on consistent hashing.
// Returns the selected variant or an empty string if variants is empty.
func SelectVariant(flagKey, userID string, variants []string) string {
	if len(variants) == 0 {
		return ""
	}
	if len(variants) == 1 {
		return variants[0]
	}
	bucket := GetVariantBucket(flagKey, userID, len(variants))
	return variants[bucket]
}

// SelectVariantWeighted selects a variant based on weights.
// weights should correspond to variants and represent relative weights (not percentages).
// For example: variants=["A", "B", "C"] with weights=[50, 30, 20] gives A 50%, B 30%, C 20%.
func SelectVariantWeighted(flagKey, userID string, variants []string, weights []int) string {
	if len(variants) == 0 || len(weights) == 0 {
		return ""
	}
	if len(variants) != len(weights) {
		// Fallback to uniform distribution
		return SelectVariant(flagKey, userID, variants)
	}

	// Calculate total weight
	totalWeight := 0
	for _, w := range weights {
		if w > 0 {
			totalWeight += w
		}
	}
	if totalWeight == 0 {
		return SelectVariant(flagKey, userID, variants)
	}

	// Get user's bucket position
	hashInput := flagKey + ":variant:" + userID
	h := murmur3Hash32([]byte(hashInput), 0)
	position := int(h % uint32(totalWeight))

	// Find which variant the position falls into
	cumulative := 0
	for i, w := range weights {
		if w > 0 {
			cumulative += w
			if position < cumulative {
				return variants[i]
			}
		}
	}

	// Fallback (should not reach here)
	return variants[len(variants)-1]
}

// murmur3Hash32 implements the MurmurHash3 32-bit hash algorithm
// This is a pure Go implementation for consistency across platforms
func murmur3Hash32(data []byte, seed uint32) uint32 {
	const (
		c1 = 0xcc9e2d51
		c2 = 0x1b873593
		r1 = 15
		r2 = 13
		m  = 5
		n  = 0xe6546b64
	)

	h := seed
	length := len(data)
	nblocks := length / 4

	// Process the body
	for i := range nblocks {
		k := binary.LittleEndian.Uint32(data[i*4:])

		k *= c1
		k = rotl32(k, r1)
		k *= c2

		h ^= k
		h = rotl32(h, r2)
		h = h*m + n
	}

	// Process the tail
	tail := data[nblocks*4:]
	var k1 uint32

	switch len(tail) {
	case 3:
		k1 ^= uint32(tail[2]) << 16
		fallthrough
	case 2:
		k1 ^= uint32(tail[1]) << 8
		fallthrough
	case 1:
		k1 ^= uint32(tail[0])
		k1 *= c1
		k1 = rotl32(k1, r1)
		k1 *= c2
		h ^= k1
	}

	// Finalization
	h ^= uint32(length)
	h = fmix32(h)

	return h
}

// rotl32 performs a 32-bit left rotation
func rotl32(x uint32, r uint8) uint32 {
	return (x << r) | (x >> (32 - r))
}

// fmix32 is the finalization mix function for MurmurHash3
func fmix32(h uint32) uint32 {
	h ^= h >> 16
	h *= 0x85ebca6b
	h ^= h >> 13
	h *= 0xc2b2ae35
	h ^= h >> 16
	return h
}

// FNVHash computes a FNV-1a hash bucket (alternative to MurmurHash)
// Useful when you need a different hash function for comparison or fallback
func FNVHash(flagKey, userID string) int {
	h := fnv.New32a()
	h.Write([]byte(flagKey + ":" + userID))
	return int(h.Sum32() % HashBucketCount)
}

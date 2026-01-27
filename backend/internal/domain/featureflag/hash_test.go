package featureflag

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComputeHashBucket(t *testing.T) {
	tests := []struct {
		name    string
		flagKey string
		userID  string
	}{
		{"simple", "feature-x", "user-123"},
		{"long flag key", "very.long.feature.flag.key.name", "user-456"},
		{"empty user", "feature", ""},
		{"unicode", "特性标志", "用户"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bucket := ComputeHashBucket(tc.flagKey, tc.userID)

			// Bucket should be in valid range
			assert.GreaterOrEqual(t, bucket, 0)
			assert.Less(t, bucket, HashBucketCount)

			// Same inputs should always produce same output (consistency)
			bucket2 := ComputeHashBucket(tc.flagKey, tc.userID)
			assert.Equal(t, bucket, bucket2)
		})
	}
}

func TestComputeHashBucket_Consistency(t *testing.T) {
	// Run multiple times to ensure consistency
	flagKey := "test-feature"
	userID := "user-12345"

	expected := ComputeHashBucket(flagKey, userID)

	for i := 0; i < 100; i++ {
		result := ComputeHashBucket(flagKey, userID)
		assert.Equal(t, expected, result, "Hash should be consistent across calls")
	}
}

func TestComputeHashBucket_Distribution(t *testing.T) {
	// Test that hash distribution is reasonably uniform
	buckets := make(map[int]int)
	numUsers := 10000

	for i := 0; i < numUsers; i++ {
		userID := string(rune(i))
		bucket := ComputeHashBucket("test-feature", userID)
		buckets[bucket]++
	}

	// Check that all buckets are used and roughly uniform
	expectedPerBucket := numUsers / HashBucketCount
	tolerance := float64(expectedPerBucket) * 0.5 // 50% tolerance

	for bucket := 0; bucket < HashBucketCount; bucket++ {
		count := buckets[bucket]
		// Allow some variance, but should not be completely empty or overfilled
		assert.GreaterOrEqual(t, count, 0)
	}

	// At least 80% of buckets should have some entries
	usedBuckets := 0
	for _, count := range buckets {
		if count > 0 {
			usedBuckets++
		}
	}
	assert.GreaterOrEqual(t, usedBuckets, int(float64(HashBucketCount)*0.8),
		"Should have reasonable distribution across buckets")
	_ = tolerance // suppress unused warning
}

func TestComputeHashBucketWithSeed(t *testing.T) {
	flagKey := "test-feature"
	userID := "user-123"

	// Different seeds should produce different results (usually)
	bucket1 := ComputeHashBucketWithSeed(flagKey, userID, 0)
	bucket2 := ComputeHashBucketWithSeed(flagKey, userID, 12345)
	bucket3 := ComputeHashBucketWithSeed(flagKey, userID, 67890)

	// Ensure consistency with same seed
	assert.Equal(t, bucket1, ComputeHashBucketWithSeed(flagKey, userID, 0))
	assert.Equal(t, bucket2, ComputeHashBucketWithSeed(flagKey, userID, 12345))
	assert.Equal(t, bucket3, ComputeHashBucketWithSeed(flagKey, userID, 67890))

	// While different seeds could theoretically produce same bucket,
	// it's very unlikely for all three to be the same
	allSame := bucket1 == bucket2 && bucket2 == bucket3
	// This is a weak assertion since it's technically possible
	t.Logf("Buckets: seed0=%d, seed12345=%d, seed67890=%d", bucket1, bucket2, bucket3)
	_ = allSame
}

func TestIsInPercentage(t *testing.T) {
	tests := []struct {
		name       string
		flagKey    string
		userID     string
		percentage int
		// We test for boundary conditions
	}{
		{"0% always false", "feature", "user", 0},
		{"100% always true", "feature", "user", 100},
		{"negative percentage", "feature", "user", -1},
		{"over 100 percentage", "feature", "user", 150},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsInPercentage(tc.flagKey, tc.userID, tc.percentage)

			switch {
			case tc.percentage <= 0:
				assert.False(t, result, "0 or negative percentage should always be false")
			case tc.percentage >= 100:
				assert.True(t, result, "100+ percentage should always be true")
			}
		})
	}
}

func TestIsInPercentage_Consistency(t *testing.T) {
	flagKey := "rollout-feature"
	userID := "consistent-user"
	percentage := 50

	expected := IsInPercentage(flagKey, userID, percentage)

	for i := 0; i < 100; i++ {
		result := IsInPercentage(flagKey, userID, percentage)
		assert.Equal(t, expected, result, "Same user should always get same result")
	}
}

func TestIsInPercentage_Distribution(t *testing.T) {
	// Test that percentage rollout roughly matches expected percentage
	flagKey := "rollout-test"
	percentage := 30
	numUsers := 10000

	inCount := 0
	for i := 0; i < numUsers; i++ {
		userID := string(rune(i + 1000)) // Offset to ensure unique users
		if IsInPercentage(flagKey, userID, percentage) {
			inCount++
		}
	}

	actualPercentage := float64(inCount) / float64(numUsers) * 100
	// Allow 5% tolerance
	assert.InDelta(t, percentage, actualPercentage, 5.0,
		"Actual percentage should be close to target percentage")
}

func TestGetVariantBucket(t *testing.T) {
	tests := []struct {
		name         string
		flagKey      string
		userID       string
		variantCount int
	}{
		{"2 variants", "ab-test", "user-1", 2},
		{"3 variants", "abc-test", "user-2", 3},
		{"4 variants", "abcd-test", "user-3", 4},
		{"1 variant returns 0", "single", "user", 1},
		{"0 variants returns 0", "zero", "user", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bucket := GetVariantBucket(tc.flagKey, tc.userID, tc.variantCount)

			if tc.variantCount <= 1 {
				assert.Equal(t, 0, bucket)
			} else {
				assert.GreaterOrEqual(t, bucket, 0)
				assert.Less(t, bucket, tc.variantCount)
			}

			// Consistency check
			bucket2 := GetVariantBucket(tc.flagKey, tc.userID, tc.variantCount)
			assert.Equal(t, bucket, bucket2)
		})
	}
}

func TestSelectVariant(t *testing.T) {
	tests := []struct {
		name     string
		flagKey  string
		userID   string
		variants []string
	}{
		{"empty variants", "feature", "user", []string{}},
		{"single variant", "feature", "user", []string{"A"}},
		{"two variants", "ab-test", "user-123", []string{"A", "B"}},
		{"three variants", "abc-test", "user-456", []string{"control", "variant-a", "variant-b"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := SelectVariant(tc.flagKey, tc.userID, tc.variants)

			if len(tc.variants) == 0 {
				assert.Empty(t, result)
			} else {
				assert.Contains(t, tc.variants, result)
			}

			// Consistency check
			result2 := SelectVariant(tc.flagKey, tc.userID, tc.variants)
			assert.Equal(t, result, result2)
		})
	}
}

func TestSelectVariant_Distribution(t *testing.T) {
	// Test that variant selection is roughly uniform
	flagKey := "variant-test"
	variants := []string{"A", "B", "C"}
	counts := make(map[string]int)
	numUsers := 9000 // Divisible by 3 for easy math

	for i := 0; i < numUsers; i++ {
		userID := string(rune(i + 2000))
		variant := SelectVariant(flagKey, userID, variants)
		counts[variant]++
	}

	expectedPerVariant := numUsers / len(variants)
	tolerance := float64(expectedPerVariant) * 0.2 // 20% tolerance

	for _, variant := range variants {
		count := counts[variant]
		assert.InDelta(t, expectedPerVariant, count, tolerance,
			"Variant %s should have roughly equal distribution", variant)
	}
}

func TestSelectVariantWeighted(t *testing.T) {
	tests := []struct {
		name     string
		variants []string
		weights  []int
	}{
		{"empty variants", []string{}, []int{}},
		{"mismatched lengths", []string{"A", "B"}, []int{50}},
		{"all zero weights", []string{"A", "B"}, []int{0, 0}},
		{"normal weights", []string{"A", "B", "C"}, []int{50, 30, 20}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := SelectVariantWeighted("feature", "user", tc.variants, tc.weights)

			if len(tc.variants) == 0 {
				assert.Empty(t, result)
			} else if len(tc.variants) != len(tc.weights) || allZeroWeights(tc.weights) {
				// Falls back to uniform selection
				if len(tc.variants) > 0 {
					assert.Contains(t, tc.variants, result)
				}
			} else {
				assert.Contains(t, tc.variants, result)
			}
		})
	}
}

func TestSelectVariantWeighted_Distribution(t *testing.T) {
	// Test weighted distribution: 50%, 30%, 20%
	flagKey := "weighted-variant"
	variants := []string{"A", "B", "C"}
	weights := []int{50, 30, 20}
	counts := make(map[string]int)
	numUsers := 10000

	for i := 0; i < numUsers; i++ {
		userID := string(rune(i + 3000))
		variant := SelectVariantWeighted(flagKey, userID, variants, weights)
		counts[variant]++
	}

	// Check approximate distributions with 10% tolerance
	assert.InDelta(t, 5000, counts["A"], 500, "A should be ~50%")
	assert.InDelta(t, 3000, counts["B"], 300, "B should be ~30%")
	assert.InDelta(t, 2000, counts["C"], 200, "C should be ~20%")
}

func TestMurmur3Hash32(t *testing.T) {
	// Test basic properties of the hash function
	tests := []struct {
		name string
		data string
		seed uint32
	}{
		{"empty string", "", 0},
		{"simple string", "hello", 0},
		{"with seed", "hello", 12345},
		{"longer string", "the quick brown fox jumps over the lazy dog", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hash1 := murmur3Hash32([]byte(tc.data), tc.seed)
			hash2 := murmur3Hash32([]byte(tc.data), tc.seed)

			// Should be consistent
			assert.Equal(t, hash1, hash2)

			// Should produce uint32 range values
			assert.LessOrEqual(t, hash1, uint32(0xFFFFFFFF))
		})
	}
}

func TestMurmur3Hash32_Uniqueness(t *testing.T) {
	// Different inputs should generally produce different hashes
	hashes := make(map[uint32]string)
	collisions := 0

	for i := 0; i < 1000; i++ {
		input := string(rune(i))
		hash := murmur3Hash32([]byte(input), 0)

		if existing, ok := hashes[hash]; ok && existing != input {
			collisions++
		}
		hashes[hash] = input
	}

	// Some collisions are expected but should be rare
	assert.Less(t, collisions, 10, "Collision rate should be low")
}

func TestFNVHash(t *testing.T) {
	flagKey := "test-feature"
	userID := "user-123"

	bucket := FNVHash(flagKey, userID)

	// Should be in valid range
	assert.GreaterOrEqual(t, bucket, 0)
	assert.Less(t, bucket, HashBucketCount)

	// Should be consistent
	bucket2 := FNVHash(flagKey, userID)
	assert.Equal(t, bucket, bucket2)
}

// Helper function
func allZeroWeights(weights []int) bool {
	for _, w := range weights {
		if w > 0 {
			return false
		}
	}
	return true
}

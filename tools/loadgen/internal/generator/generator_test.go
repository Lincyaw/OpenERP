package generator

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewGenerator tests the factory function for creating generators.
func TestNewGenerator(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "faker generator",
			config: Config{
				Type:  TypeFaker,
				Faker: &FakerConfig{Type: "name"},
			},
			wantErr: false,
		},
		{
			name: "random generator",
			config: Config{
				Type:   TypeRandom,
				Random: &RandomConfig{Type: "int", Min: 1, Max: 100},
			},
			wantErr: false,
		},
		{
			name: "pattern generator",
			config: Config{
				Type:    TypePattern,
				Pattern: &PatternConfig{Pattern: "TEST-{RANDOM:4}"},
			},
			wantErr: false,
		},
		{
			name: "sequence generator",
			config: Config{
				Type:     TypeSequence,
				Sequence: &SequenceConfig{Start: 1, Step: 1},
			},
			wantErr: false,
		},
		{
			name: "sequence generator with nil config uses defaults",
			config: Config{
				Type: TypeSequence,
			},
			wantErr: false,
		},
		{
			name: "missing faker config",
			config: Config{
				Type: TypeFaker,
			},
			wantErr: true,
			errMsg:  "faker config is required",
		},
		{
			name: "missing random config",
			config: Config{
				Type: TypeRandom,
			},
			wantErr: true,
			errMsg:  "random config is required",
		},
		{
			name: "missing pattern config",
			config: Config{
				Type: TypePattern,
			},
			wantErr: true,
			errMsg:  "pattern config is required",
		},
		{
			name: "unknown type",
			config: Config{
				Type: "unknown",
			},
			wantErr: true,
			errMsg:  "unknown generator type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := New(tt.config)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, gen)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, gen)
			}
		})
	}
}

// TestRegistry tests the generator registry functionality.
func TestRegistry(t *testing.T) {
	t.Run("register and get", func(t *testing.T) {
		reg := NewRegistry()

		gen, err := NewFakerGenerator(&FakerConfig{Type: "name"})
		require.NoError(t, err)

		reg.Register("test.name", gen)

		retrieved, err := reg.Get("test.name")
		require.NoError(t, err)
		assert.Equal(t, gen, retrieved)
	})

	t.Run("get non-existent", func(t *testing.T) {
		reg := NewRegistry()

		_, err := reg.Get("nonexistent")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrGeneratorNotFound)
	})

	t.Run("has", func(t *testing.T) {
		reg := NewRegistry()

		gen, err := NewFakerGenerator(&FakerConfig{Type: "name"})
		require.NoError(t, err)

		reg.Register("test.name", gen)

		assert.True(t, reg.Has("test.name"))
		assert.False(t, reg.Has("nonexistent"))
	})

	t.Run("names", func(t *testing.T) {
		reg := NewRegistry()

		gen1, _ := NewFakerGenerator(&FakerConfig{Type: "name"})
		gen2, _ := NewFakerGenerator(&FakerConfig{Type: "email"})

		reg.Register("test.name", gen1)
		reg.Register("test.email", gen2)

		names := reg.Names()
		assert.Len(t, names, 2)
		assert.Contains(t, names, "test.name")
		assert.Contains(t, names, "test.email")
	})

	t.Run("generate", func(t *testing.T) {
		reg := NewRegistry()

		gen, err := NewFakerGenerator(&FakerConfig{Type: "name"})
		require.NoError(t, err)

		reg.Register("test.name", gen)

		value, err := reg.Generate("test.name")
		require.NoError(t, err)
		assert.NotEmpty(t, value)
	})

	t.Run("generate non-existent", func(t *testing.T) {
		reg := NewRegistry()

		_, err := reg.Generate("nonexistent")
		require.Error(t, err)
	})

	t.Run("load from config", func(t *testing.T) {
		reg := NewRegistry()

		configs := map[string]Config{
			"common.name": {
				Type:  TypeFaker,
				Faker: &FakerConfig{Type: "name"},
			},
			"common.code": {
				Type:    TypePattern,
				Pattern: &PatternConfig{Pattern: "CODE-{RANDOM:4}"},
			},
		}

		err := reg.LoadFromConfig(configs)
		require.NoError(t, err)

		assert.True(t, reg.Has("common.name"))
		assert.True(t, reg.Has("common.code"))
	})

	t.Run("load from config with error", func(t *testing.T) {
		reg := NewRegistry()

		configs := map[string]Config{
			"invalid": {
				Type: TypeFaker,
				// Missing faker config
			},
		}

		err := reg.LoadFromConfig(configs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid")
	})
}

// TestFakerGenerator tests the faker generator.
func TestFakerGenerator(t *testing.T) {
	t.Run("create with nil config", func(t *testing.T) {
		_, err := NewFakerGenerator(nil)
		require.Error(t, err)
	})

	t.Run("create with empty type", func(t *testing.T) {
		_, err := NewFakerGenerator(&FakerConfig{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "type is required")
	})

	t.Run("create with unknown type", func(t *testing.T) {
		_, err := NewFakerGenerator(&FakerConfig{Type: "unknown"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown faker type")
	})

	// Test various faker types
	fakerTypes := []string{
		"name", "firstName", "lastName",
		"email", "phone",
		"address", "city", "country",
		"company",
		"url", "ipv4", "ipv6",
		"uuid",
		"word", "sentence", "paragraph",
		"number", "float", "bool",
		"date", "time", "datetime",
	}

	for _, ft := range fakerTypes {
		t.Run("generate "+ft, func(t *testing.T) {
			gen, err := NewFakerGenerator(&FakerConfig{Type: ft})
			require.NoError(t, err)

			assert.Equal(t, TypeFaker, gen.Type())

			value, err := gen.Generate()
			require.NoError(t, err)
			assert.NotNil(t, value)
		})
	}
}

// TestPatternGenerator tests the pattern generator.
func TestPatternGenerator(t *testing.T) {
	t.Run("create with nil config", func(t *testing.T) {
		_, err := NewPatternGenerator(nil)
		require.Error(t, err)
	})

	t.Run("create with empty pattern", func(t *testing.T) {
		_, err := NewPatternGenerator(&PatternConfig{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pattern is required")
	})

	t.Run("PREFIX placeholder", func(t *testing.T) {
		gen, err := NewPatternGenerator(&PatternConfig{
			Pattern: "{PREFIX}-test",
			Prefix:  "PROD",
		})
		require.NoError(t, err)

		value, err := gen.Generate()
		require.NoError(t, err)
		assert.Equal(t, "PROD-test", value)
	})

	t.Run("TIMESTAMP placeholder", func(t *testing.T) {
		gen, err := NewPatternGenerator(&PatternConfig{
			Pattern: "{TIMESTAMP}",
		})
		require.NoError(t, err)

		value, err := gen.Generate()
		require.NoError(t, err)

		// Should be a valid timestamp (within last second)
		ts, err := strconv.ParseInt(value.(string), 10, 64)
		require.NoError(t, err)
		assert.InDelta(t, time.Now().UnixMilli(), ts, 1000)
	})

	t.Run("RANDOM placeholder with length", func(t *testing.T) {
		gen, err := NewPatternGenerator(&PatternConfig{
			Pattern: "{RANDOM:10}",
		})
		require.NoError(t, err)

		value, err := gen.Generate()
		require.NoError(t, err)

		assert.Len(t, value.(string), 10)
	})

	t.Run("RANDOM placeholder default length", func(t *testing.T) {
		gen, err := NewPatternGenerator(&PatternConfig{
			Pattern: "{RANDOM}",
		})
		require.NoError(t, err)

		value, err := gen.Generate()
		require.NoError(t, err)

		assert.Len(t, value.(string), 8) // default
	})

	t.Run("UUID placeholder", func(t *testing.T) {
		gen, err := NewPatternGenerator(&PatternConfig{
			Pattern: "{UUID}",
		})
		require.NoError(t, err)

		value, err := gen.Generate()
		require.NoError(t, err)

		// UUID format: 8-4-4-4-12
		uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
		assert.True(t, uuidRegex.MatchString(value.(string)))
	})

	t.Run("DATE placeholder", func(t *testing.T) {
		gen, err := NewPatternGenerator(&PatternConfig{
			Pattern: "{DATE}",
		})
		require.NoError(t, err)

		value, err := gen.Generate()
		require.NoError(t, err)

		// Should match YYYY-MM-DD format
		dateRegex := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
		assert.True(t, dateRegex.MatchString(value.(string)))
	})

	t.Run("TIME placeholder", func(t *testing.T) {
		gen, err := NewPatternGenerator(&PatternConfig{
			Pattern: "{TIME}",
		})
		require.NoError(t, err)

		value, err := gen.Generate()
		require.NoError(t, err)

		// Should match HH:MM:SS format
		timeRegex := regexp.MustCompile(`^\d{2}:\d{2}:\d{2}$`)
		assert.True(t, timeRegex.MatchString(value.(string)))
	})

	t.Run("DATETIME placeholder", func(t *testing.T) {
		gen, err := NewPatternGenerator(&PatternConfig{
			Pattern: "{DATETIME}",
		})
		require.NoError(t, err)

		value, err := gen.Generate()
		require.NoError(t, err)

		// Should be parseable as RFC3339
		_, err = time.Parse(time.RFC3339, value.(string))
		assert.NoError(t, err)
	})

	t.Run("INT placeholder with range", func(t *testing.T) {
		gen, err := NewPatternGenerator(&PatternConfig{
			Pattern: "{INT:10:20}",
		})
		require.NoError(t, err)

		for i := 0; i < 100; i++ {
			value, err := gen.Generate()
			require.NoError(t, err)

			num, err := strconv.Atoi(value.(string))
			require.NoError(t, err)
			assert.GreaterOrEqual(t, num, 10)
			assert.LessOrEqual(t, num, 20)
		}
	})

	t.Run("ALPHA placeholder", func(t *testing.T) {
		gen, err := NewPatternGenerator(&PatternConfig{
			Pattern: "{ALPHA:6}",
		})
		require.NoError(t, err)

		value, err := gen.Generate()
		require.NoError(t, err)

		assert.Len(t, value.(string), 6)
		assert.True(t, regexp.MustCompile(`^[a-zA-Z]+$`).MatchString(value.(string)))
	})

	t.Run("HEX placeholder", func(t *testing.T) {
		gen, err := NewPatternGenerator(&PatternConfig{
			Pattern: "{HEX:8}",
		})
		require.NoError(t, err)

		value, err := gen.Generate()
		require.NoError(t, err)

		assert.Len(t, value.(string), 8)
		assert.True(t, regexp.MustCompile(`^[0-9a-f]+$`).MatchString(value.(string)))
	})

	t.Run("SEQUENCE placeholder", func(t *testing.T) {
		gen, err := NewPatternGenerator(&PatternConfig{
			Pattern: "SEQ-{SEQUENCE}",
		})
		require.NoError(t, err)

		v1, _ := gen.Generate()
		v2, _ := gen.Generate()
		v3, _ := gen.Generate()

		assert.Equal(t, "SEQ-1", v1)
		assert.Equal(t, "SEQ-2", v2)
		assert.Equal(t, "SEQ-3", v3)
	})

	t.Run("complex pattern", func(t *testing.T) {
		gen, err := NewPatternGenerator(&PatternConfig{
			Pattern: "{PREFIX}_{DATE}_{RANDOM:4}",
			Prefix:  "ORDER",
		})
		require.NoError(t, err)

		value, err := gen.Generate()
		require.NoError(t, err)

		parts := strings.Split(value.(string), "_")
		assert.Len(t, parts, 3) // ORDER, YYYY-MM-DD, XXXX

		// Check it starts with ORDER
		assert.True(t, strings.HasPrefix(value.(string), "ORDER_"))
	})

	t.Run("unknown placeholder preserved", func(t *testing.T) {
		gen, err := NewPatternGenerator(&PatternConfig{
			Pattern: "{UNKNOWN}",
		})
		require.NoError(t, err)

		value, err := gen.Generate()
		require.NoError(t, err)
		assert.Equal(t, "{UNKNOWN}", value)
	})

	t.Run("type returns pattern", func(t *testing.T) {
		gen, err := NewPatternGenerator(&PatternConfig{
			Pattern: "test",
		})
		require.NoError(t, err)
		assert.Equal(t, TypePattern, gen.Type())
	})
}

// TestRandomGenerator tests the random generator.
func TestRandomGenerator(t *testing.T) {
	t.Run("create with nil config", func(t *testing.T) {
		_, err := NewRandomGenerator(nil)
		require.Error(t, err)
	})

	t.Run("create with empty type", func(t *testing.T) {
		_, err := NewRandomGenerator(&RandomConfig{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "type is required")
	})

	t.Run("create with unknown type", func(t *testing.T) {
		_, err := NewRandomGenerator(&RandomConfig{Type: "unknown"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown random type")
	})

	t.Run("int generator", func(t *testing.T) {
		gen, err := NewRandomGenerator(&RandomConfig{
			Type: "int",
			Min:  10,
			Max:  20,
		})
		require.NoError(t, err)

		for i := 0; i < 100; i++ {
			value, err := gen.Generate()
			require.NoError(t, err)

			num := value.(int)
			assert.GreaterOrEqual(t, num, 10)
			assert.LessOrEqual(t, num, 20)
		}
	})

	t.Run("int generator default range", func(t *testing.T) {
		gen, err := NewRandomGenerator(&RandomConfig{Type: "int"})
		require.NoError(t, err)

		value, err := gen.Generate()
		require.NoError(t, err)

		num := value.(int)
		assert.GreaterOrEqual(t, num, 0)
		assert.LessOrEqual(t, num, 100) // default max
	})

	t.Run("float generator", func(t *testing.T) {
		gen, err := NewRandomGenerator(&RandomConfig{
			Type: "float",
			Min:  1.5,
			Max:  3.5,
		})
		require.NoError(t, err)

		for i := 0; i < 100; i++ {
			value, err := gen.Generate()
			require.NoError(t, err)

			num := value.(float64)
			assert.GreaterOrEqual(t, num, 1.5)
			assert.LessOrEqual(t, num, 3.5)
		}
	})

	t.Run("string generator", func(t *testing.T) {
		gen, err := NewRandomGenerator(&RandomConfig{
			Type:   "string",
			Length: 16,
		})
		require.NoError(t, err)

		value, err := gen.Generate()
		require.NoError(t, err)

		assert.Len(t, value.(string), 16)
	})

	t.Run("string generator default length", func(t *testing.T) {
		gen, err := NewRandomGenerator(&RandomConfig{Type: "string"})
		require.NoError(t, err)

		value, err := gen.Generate()
		require.NoError(t, err)

		assert.Len(t, value.(string), 8) // default
	})

	t.Run("string generator charsets", func(t *testing.T) {
		charsets := map[string]*regexp.Regexp{
			"alpha":          regexp.MustCompile(`^[a-zA-Z]+$`),
			"numeric":        regexp.MustCompile(`^[0-9]+$`),
			"hex":            regexp.MustCompile(`^[0-9a-f]+$`),
			"alphanum_lower": regexp.MustCompile(`^[a-z0-9]+$`),
			"alphanum_upper": regexp.MustCompile(`^[A-Z0-9]+$`),
			"alphanumeric":   regexp.MustCompile(`^[a-zA-Z0-9]+$`),
		}

		for charset, regex := range charsets {
			t.Run(charset, func(t *testing.T) {
				gen, err := NewRandomGenerator(&RandomConfig{
					Type:    "string",
					Length:  20,
					Charset: charset,
				})
				require.NoError(t, err)

				value, err := gen.Generate()
				require.NoError(t, err)

				assert.True(t, regex.MatchString(value.(string)),
					"expected %s to match %s", value, charset)
			})
		}
	})

	t.Run("uuid generator", func(t *testing.T) {
		gen, err := NewRandomGenerator(&RandomConfig{Type: "uuid"})
		require.NoError(t, err)

		value, err := gen.Generate()
		require.NoError(t, err)

		uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
		assert.True(t, uuidRegex.MatchString(value.(string)))
	})

	t.Run("bool generator", func(t *testing.T) {
		gen, err := NewRandomGenerator(&RandomConfig{Type: "bool"})
		require.NoError(t, err)

		trueCount := 0
		falseCount := 0

		for i := 0; i < 100; i++ {
			value, err := gen.Generate()
			require.NoError(t, err)

			if value.(bool) {
				trueCount++
			} else {
				falseCount++
			}
		}

		// Should have at least some of each (statistically unlikely to fail)
		assert.Greater(t, trueCount, 0)
		assert.Greater(t, falseCount, 0)
	})

	t.Run("type returns random", func(t *testing.T) {
		gen, err := NewRandomGenerator(&RandomConfig{Type: "int"})
		require.NoError(t, err)
		assert.Equal(t, TypeRandom, gen.Type())
	})
}

// TestSequenceGenerator tests the sequence generator.
func TestSequenceGenerator(t *testing.T) {
	t.Run("create with nil config uses defaults", func(t *testing.T) {
		gen, err := NewSequenceGenerator(nil)
		require.NoError(t, err)

		v1, _ := gen.Generate()
		v2, _ := gen.Generate()

		assert.Equal(t, "1", v1)
		assert.Equal(t, "2", v2)
	})

	t.Run("custom start", func(t *testing.T) {
		gen, err := NewSequenceGenerator(&SequenceConfig{Start: 100})
		require.NoError(t, err)

		v1, _ := gen.Generate()
		v2, _ := gen.Generate()

		assert.Equal(t, "100", v1)
		assert.Equal(t, "101", v2)
	})

	t.Run("custom step", func(t *testing.T) {
		gen, err := NewSequenceGenerator(&SequenceConfig{Start: 5, Step: 5})
		require.NoError(t, err)

		v1, _ := gen.Generate()
		v2, _ := gen.Generate()
		v3, _ := gen.Generate()

		assert.Equal(t, "5", v1)
		assert.Equal(t, "10", v2)
		assert.Equal(t, "15", v3)
	})

	t.Run("prefix and suffix", func(t *testing.T) {
		gen, err := NewSequenceGenerator(&SequenceConfig{
			Prefix: "ORDER-",
			Suffix: "-END",
		})
		require.NoError(t, err)

		v1, _ := gen.Generate()
		assert.Equal(t, "ORDER-1-END", v1)
	})

	t.Run("padding", func(t *testing.T) {
		gen, err := NewSequenceGenerator(&SequenceConfig{
			Start:   1,
			Padding: 5,
		})
		require.NoError(t, err)

		v1, _ := gen.Generate()
		v2, _ := gen.Generate()

		assert.Equal(t, "00001", v1)
		assert.Equal(t, "00002", v2)
	})

	t.Run("reset", func(t *testing.T) {
		gen, err := NewSequenceGenerator(&SequenceConfig{Start: 1})
		require.NoError(t, err)

		gen.Generate()
		gen.Generate()
		gen.Reset()

		v1, _ := gen.Generate()
		assert.Equal(t, "1", v1)
	})

	t.Run("current", func(t *testing.T) {
		gen, err := NewSequenceGenerator(&SequenceConfig{Start: 1})
		require.NoError(t, err)

		gen.Generate()
		gen.Generate()

		assert.Equal(t, int64(2), gen.Current())
	})

	t.Run("type returns sequence", func(t *testing.T) {
		gen, err := NewSequenceGenerator(&SequenceConfig{})
		require.NoError(t, err)
		assert.Equal(t, TypeSequence, gen.Type())
	})
}

// TestSupportedFakerTypes tests that we can list supported faker types.
func TestSupportedFakerTypes(t *testing.T) {
	types := SupportedFakerTypes()
	assert.NotEmpty(t, types)
	assert.Contains(t, types, "name")
	assert.Contains(t, types, "email")
	assert.Contains(t, types, "phone")
}

// BenchmarkGenerators benchmarks generator performance.
func BenchmarkGenerators(b *testing.B) {
	b.Run("faker/name", func(b *testing.B) {
		gen, _ := NewFakerGenerator(&FakerConfig{Type: "name"})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			gen.Generate()
		}
	})

	b.Run("pattern/complex", func(b *testing.B) {
		gen, _ := NewPatternGenerator(&PatternConfig{
			Pattern: "{PREFIX}-{TIMESTAMP}-{RANDOM:8}",
			Prefix:  "TEST",
		})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			gen.Generate()
		}
	})

	b.Run("random/string", func(b *testing.B) {
		gen, _ := NewRandomGenerator(&RandomConfig{
			Type:   "string",
			Length: 32,
		})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			gen.Generate()
		}
	})

	b.Run("sequence", func(b *testing.B) {
		gen, _ := NewSequenceGenerator(&SequenceConfig{})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			gen.Generate()
		}
	})
}

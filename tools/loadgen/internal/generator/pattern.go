// Package generator provides data generation capabilities for the load generator.
package generator

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// PatternGenerator generates values based on patterns with placeholders.
// Supported placeholders:
//   - {PREFIX}         - A configurable prefix string
//   - {TIMESTAMP}      - Current Unix timestamp in milliseconds
//   - {RANDOM:N}       - Random alphanumeric string of length N
//   - {UUID}           - Random UUID v4
//   - {DATE}           - Current date in YYYY-MM-DD format
//   - {TIME}           - Current time in HH:MM:SS format
//   - {DATETIME}       - Current datetime in ISO 8601 format
//   - {INT:MIN:MAX}    - Random integer between MIN and MAX
//   - {ALPHA:N}        - Random alphabetic string of length N
//   - {HEX:N}          - Random hexadecimal string of length N
//   - {SEQUENCE}       - Auto-incrementing sequence number
type PatternGenerator struct {
	config   *PatternConfig
	sequence int64 // atomic counter for {SEQUENCE}
}

// placeholderRegex matches all supported placeholders.
var placeholderRegex = regexp.MustCompile(`\{([A-Z]+)(:[^}]*)?\}`)

// NewPatternGenerator creates a new pattern generator with the given configuration.
func NewPatternGenerator(cfg *PatternConfig) (*PatternGenerator, error) {
	if cfg == nil {
		return nil, fmt.Errorf("%w: pattern config is nil", ErrInvalidConfig)
	}
	if cfg.Pattern == "" {
		return nil, fmt.Errorf("%w: pattern is required", ErrInvalidConfig)
	}

	return &PatternGenerator{
		config:   cfg,
		sequence: 0,
	}, nil
}

// Generate produces a new value based on the pattern.
func (p *PatternGenerator) Generate() (any, error) {
	result := placeholderRegex.ReplaceAllStringFunc(p.config.Pattern, func(match string) string {
		// Extract placeholder name and arguments
		parts := placeholderRegex.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		name := parts[1]
		args := ""
		if len(parts) > 2 && parts[2] != "" {
			args = parts[2][1:] // Remove leading ':'
		}

		return p.replacePlaceholder(name, args)
	})

	return result, nil
}

// Type returns the generator type.
func (p *PatternGenerator) Type() GeneratorType {
	return TypePattern
}

// replacePlaceholder replaces a single placeholder with its generated value.
func (p *PatternGenerator) replacePlaceholder(name, args string) string {
	switch name {
	case "PREFIX":
		return p.config.Prefix

	case "TIMESTAMP":
		return strconv.FormatInt(time.Now().UnixMilli(), 10)

	case "RANDOM":
		length := 8 // default
		if args != "" {
			if n, err := strconv.Atoi(args); err == nil && n > 0 {
				length = n
			}
		}
		return randomAlphanumeric(length)

	case "UUID":
		return uuid.New().String()

	case "DATE":
		return time.Now().Format("2006-01-02")

	case "TIME":
		return time.Now().Format("15:04:05")

	case "DATETIME":
		return time.Now().Format(time.RFC3339)

	case "INT":
		min, max := parseIntRange(args)
		return strconv.Itoa(randomInt(min, max))

	case "ALPHA":
		length := 8 // default
		if args != "" {
			if n, err := strconv.Atoi(args); err == nil && n > 0 {
				length = n
			}
		}
		return randomAlpha(length)

	case "HEX":
		length := 8 // default
		if args != "" {
			if n, err := strconv.Atoi(args); err == nil && n > 0 {
				length = n
			}
		}
		return randomHex(length)

	case "SEQUENCE":
		seq := atomic.AddInt64(&p.sequence, 1)
		return strconv.FormatInt(seq, 10)

	default:
		return "{" + name + "}"
	}
}

// parseIntRange parses "MIN:MAX" string into two integers.
// Returns (0, 100) as default if parsing fails.
func parseIntRange(args string) (int, int) {
	if args == "" {
		return 0, 100
	}

	parts := strings.Split(args, ":")
	if len(parts) != 2 {
		return 0, 100
	}

	min, err1 := strconv.Atoi(parts[0])
	max, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return 0, 100
	}

	if min > max {
		min, max = max, min
	}

	return min, max
}

// Character sets for random string generation.
const (
	alphanumericChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	alphaChars        = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numericChars      = "0123456789"
)

// randomAlphanumeric generates a random alphanumeric string of the given length.
func randomAlphanumeric(length int) string {
	return randomString(length, alphanumericChars)
}

// randomAlpha generates a random alphabetic string of the given length.
func randomAlpha(length int) string {
	return randomString(length, alphaChars)
}

// randomHex generates a random hexadecimal string of the given length.
func randomHex(length int) string {
	bytes := make([]byte, (length+1)/2)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to simple random
		return randomString(length, "0123456789abcdef")
	}
	return hex.EncodeToString(bytes)[:length]
}

// randomString generates a random string from the given character set.
func randomString(length int, charset string) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// This should never happen in practice
		return strings.Repeat(string(charset[0]), length)
	}

	charsetLen := len(charset)
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = charset[int(bytes[i])%charsetLen]
	}
	return string(result)
}

// randomInt generates a random integer between min and max (inclusive).
func randomInt(min, max int) int {
	if min >= max {
		return min
	}

	rangeSize := max - min + 1
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return min
	}

	// Convert bytes to uint32 and map to range
	n := int(bytes[0])<<24 | int(bytes[1])<<16 | int(bytes[2])<<8 | int(bytes[3])
	if n < 0 {
		n = -n
	}
	return min + (n % rangeSize)
}

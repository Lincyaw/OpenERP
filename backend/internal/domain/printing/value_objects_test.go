package printing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMargins(t *testing.T) {
	tests := []struct {
		name        string
		top         int
		right       int
		bottom      int
		left        int
		expectError bool
		errorCode   string
	}{
		{"valid margins", 10, 10, 10, 10, false, ""},
		{"zero margins", 0, 0, 0, 0, false, ""},
		{"max margins", 100, 100, 100, 100, false, ""},
		{"mixed margins", 5, 10, 15, 20, false, ""},
		{"negative top", -1, 10, 10, 10, true, "INVALID_MARGINS"},
		{"negative right", 10, -1, 10, 10, true, "INVALID_MARGINS"},
		{"negative bottom", 10, 10, -1, 10, true, "INVALID_MARGINS"},
		{"negative left", 10, 10, 10, -1, true, "INVALID_MARGINS"},
		{"exceeds max top", 101, 10, 10, 10, true, "INVALID_MARGINS"},
		{"exceeds max right", 10, 101, 10, 10, true, "INVALID_MARGINS"},
		{"exceeds max bottom", 10, 10, 101, 10, true, "INVALID_MARGINS"},
		{"exceeds max left", 10, 10, 10, 101, true, "INVALID_MARGINS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			margins, err := NewMargins(tt.top, tt.right, tt.bottom, tt.left)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "cannot")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.top, margins.Top)
				assert.Equal(t, tt.right, margins.Right)
				assert.Equal(t, tt.bottom, margins.Bottom)
				assert.Equal(t, tt.left, margins.Left)
			}
		})
	}
}

func TestDefaultMargins(t *testing.T) {
	margins := DefaultMargins()
	assert.Equal(t, 10, margins.Top)
	assert.Equal(t, 10, margins.Right)
	assert.Equal(t, 10, margins.Bottom)
	assert.Equal(t, 10, margins.Left)
}

func TestReceiptMargins(t *testing.T) {
	margins := ReceiptMargins()
	assert.Equal(t, 2, margins.Top)
	assert.Equal(t, 2, margins.Right)
	assert.Equal(t, 2, margins.Bottom)
	assert.Equal(t, 2, margins.Left)
}

func TestMargins_IsZero(t *testing.T) {
	tests := []struct {
		name     string
		margins  Margins
		expected bool
	}{
		{"all zero", Margins{0, 0, 0, 0}, true},
		{"top non-zero", Margins{1, 0, 0, 0}, false},
		{"right non-zero", Margins{0, 1, 0, 0}, false},
		{"bottom non-zero", Margins{0, 0, 1, 0}, false},
		{"left non-zero", Margins{0, 0, 0, 1}, false},
		{"all non-zero", Margins{10, 10, 10, 10}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.margins.IsZero())
		})
	}
}

func TestMargins_Equals(t *testing.T) {
	tests := []struct {
		name     string
		m1       Margins
		m2       Margins
		expected bool
	}{
		{"equal margins", Margins{10, 10, 10, 10}, Margins{10, 10, 10, 10}, true},
		{"zero margins equal", Margins{0, 0, 0, 0}, Margins{0, 0, 0, 0}, true},
		{"different top", Margins{10, 10, 10, 10}, Margins{5, 10, 10, 10}, false},
		{"different right", Margins{10, 10, 10, 10}, Margins{10, 5, 10, 10}, false},
		{"different bottom", Margins{10, 10, 10, 10}, Margins{10, 10, 5, 10}, false},
		{"different left", Margins{10, 10, 10, 10}, Margins{10, 10, 10, 5}, false},
		{"all different", Margins{1, 2, 3, 4}, Margins{5, 6, 7, 8}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.m1.Equals(tt.m2))
		})
	}
}

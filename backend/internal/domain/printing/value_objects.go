package printing

import "github.com/erp/backend/internal/domain/shared"

// Margins represents the page margins in millimeters
type Margins struct {
	Top    int `json:"top"`    // Top margin in mm
	Right  int `json:"right"`  // Right margin in mm
	Bottom int `json:"bottom"` // Bottom margin in mm
	Left   int `json:"left"`   // Left margin in mm
}

// NewMargins creates a new Margins value object
func NewMargins(top, right, bottom, left int) (Margins, error) {
	if top < 0 || right < 0 || bottom < 0 || left < 0 {
		return Margins{}, shared.NewDomainError("INVALID_MARGINS", "Margins cannot be negative")
	}
	if top > 100 || right > 100 || bottom > 100 || left > 100 {
		return Margins{}, shared.NewDomainError("INVALID_MARGINS", "Margins cannot exceed 100mm")
	}
	return Margins{
		Top:    top,
		Right:  right,
		Bottom: bottom,
		Left:   left,
	}, nil
}

// DefaultMargins returns the default page margins for A4 paper
func DefaultMargins() Margins {
	return Margins{
		Top:    10,
		Right:  10,
		Bottom: 10,
		Left:   10,
	}
}

// ReceiptMargins returns minimal margins suitable for receipt paper
func ReceiptMargins() Margins {
	return Margins{
		Top:    2,
		Right:  2,
		Bottom: 2,
		Left:   2,
	}
}

// IsZero returns true if all margins are zero
func (m Margins) IsZero() bool {
	return m.Top == 0 && m.Right == 0 && m.Bottom == 0 && m.Left == 0
}

// Equals checks if two Margins are equal
func (m Margins) Equals(other Margins) bool {
	return m.Top == other.Top &&
		m.Right == other.Right &&
		m.Bottom == other.Bottom &&
		m.Left == other.Left
}

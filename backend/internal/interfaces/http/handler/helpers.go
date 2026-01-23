package handler

import "github.com/shopspring/decimal"

// toDecimalPtr converts a float64 to a *decimal.Decimal
func toDecimalPtr(f float64) *decimal.Decimal {
	d := decimal.NewFromFloat(f)
	return &d
}

// toDecimal converts a float64 to a decimal.Decimal
func toDecimal(f float64) decimal.Decimal {
	return decimal.NewFromFloat(f)
}

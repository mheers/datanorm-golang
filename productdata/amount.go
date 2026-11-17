package productdata

import (
	"fmt"
	"strings"
)

// Currency is an ISO 4217 currency code, for example "EUR".
type Currency string

// Common currencies. The list is not exhaustive; any valid ISO 4217 code is
// accepted by the model.
const (
	EUR Currency = "EUR"
	USD Currency = "USD"
	CHF Currency = "CHF"
	GBP Currency = "GBP"
	DEM Currency = "DEM"
	ATS Currency = "ATS"
)

// Valid reports whether c looks like an ISO 4217 code: three uppercase letters.
func (c Currency) Valid() bool {
	if len(c) != 3 {
		return false
	}
	for i := 0; i < 3; i++ {
		if c[i] < 'A' || c[i] > 'Z' {
			return false
		}
	}
	return true
}

// Amount is an exact monetary amount in a currency.
type Amount struct {
	Value    Decimal  `json:"value"`
	Currency Currency `json:"currency"`
}

// NewAmount returns an amount with the given value and currency.
func NewAmount(value Decimal, currency Currency) Amount {
	return Amount{Value: value, Currency: currency}
}

// AmountFromMinorUnits returns an amount from minor units (cents for a
// two-decimal currency). For example, AmountFromMinorUnits(129900, EUR) is
// 1299.00 EUR. The function assumes two decimal places; use NewAmount for
// currencies with a different minor-unit scale.
func AmountFromMinorUnits(minor int64, currency Currency) Amount {
	return Amount{Value: NewDecimal(minor, 2), Currency: currency}
}

// ParseAmount parses a decimal string into an amount and validates the currency.
func ParseAmount(s string, currency Currency) (Amount, error) {
	if !currency.Valid() {
		return Amount{}, fmt.Errorf("%w: %q", ErrInvalidCurrency, currency)
	}
	value, err := ParseDecimal(s)
	if err != nil {
		return Amount{}, err
	}
	return Amount{Value: value, Currency: currency}, nil
}

// IsZero reports whether the amount is zero.
func (a Amount) IsZero() bool { return a.Value.IsZero() }

// String returns "value currency", for example "1299.00 EUR".
func (a Amount) String() string {
	if a.Currency == "" {
		return a.Value.String()
	}
	return a.Value.String() + " " + string(a.Currency)
}

// Equal reports whether both the value and the currency are equal.
func (a Amount) Equal(other Amount) bool {
	return a.Currency == other.Currency && a.Value.Equal(other.Value)
}

// Add returns a + other. The currencies must match.
func (a Amount) Add(other Amount) (Amount, error) {
	if a.Currency != other.Currency {
		return Amount{}, fmt.Errorf("%w: %s + %s", ErrCurrencyMismatch, a.Currency, other.Currency)
	}
	value, err := a.Value.Add(other.Value)
	if err != nil {
		return Amount{}, err
	}
	return Amount{Value: value, Currency: a.Currency}, nil
}

// Sub returns a - other. The currencies must match.
func (a Amount) Sub(other Amount) (Amount, error) {
	return a.Add(other.negate())
}

func (a Amount) negate() Amount {
	return Amount{Value: a.Value.Neg(), Currency: a.Currency}
}

// NormalizeCurrency uppercases and trims a currency code.
func NormalizeCurrency(s string) Currency {
	return Currency(strings.ToUpper(strings.TrimSpace(s)))
}

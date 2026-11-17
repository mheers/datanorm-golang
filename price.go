package datanorm

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/mheers/datanorm-golang/productdata"
)

// Price parsing errors.
var (
	ErrNoPrice      = errors.New("datanorm: no price value")
	ErrInvalidPrice = errors.New("datanorm: invalid price")
)

// ParsePrice parses a DATANORM price value.
//
// DATANORM normally stores prices as integer minor units (cents): "129900"
// means 1299.00 EUR. Some version 5 exporters write an explicit decimal:
// "1299,00" or "1299.00". The presence of a comma or dot selects the explicit
// decimal interpretation; otherwise the value is minor units.
func ParsePrice(raw string, currency productdata.Currency) (productdata.Amount, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return productdata.Amount{}, ErrNoPrice
	}
	if !currency.Valid() {
		return productdata.Amount{}, fmt.Errorf("%w: %q", productdata.ErrInvalidCurrency, currency)
	}

	if strings.ContainsAny(value, ",.") {
		decimal, err := productdata.ParseDecimal(value)
		if err != nil {
			return productdata.Amount{}, fmt.Errorf("%w: %q: %w", ErrInvalidPrice, raw, err)
		}
		return productdata.NewAmount(decimal, currency), nil
	}

	minor, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return productdata.Amount{}, fmt.Errorf("%w: %q: %w", ErrInvalidPrice, raw, err)
	}
	return productdata.AmountFromMinorUnits(minor, currency), nil
}

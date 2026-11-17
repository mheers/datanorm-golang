// Package productdata provides a format-neutral model for product master data,
// prices, texts, groups, media, and the messages exchanged around them.
//
// The model is designed to hold DATANORM 5, IDS Connect 2, Open Masterdata, and
// similar formats without flattening away information. Format packages map their
// records onto this model; this package has no knowledge of any file format.
package productdata

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
)

// Decimal is an exact base-10 number represented as an unscaled integer and a
// scale. For example, 1299.00 is stored as unscaled=129900, scale=2.
//
// Decimal is used for money and quantities. It never uses float64 internally.
// The range is limited by int64 and a maximum scale of 18.
type Decimal struct {
	unscaled int64
	scale    uint8
}

const maxDecimalScale = 18

// Errors returned by Decimal operations.
var (
	ErrDecimalSyntax    = errors.New("invalid decimal syntax")
	ErrDecimalScale     = errors.New("decimal scale out of range")
	ErrDecimalOverflow  = errors.New("decimal overflow")
	ErrDecimalInexact   = errors.New("decimal rescale would lose precision")
	ErrCurrencyMismatch = errors.New("currency mismatch")
	ErrInvalidCurrency  = errors.New("invalid currency code")
	ErrInvalidUnit      = errors.New("invalid unit")
)

// Zero returns the decimal 0.
func Zero() Decimal {
	return Decimal{}
}

// NewDecimal returns a decimal from an unscaled integer and a scale.
// It panics if scale is greater than 18; this is a programmer error.
func NewDecimal(unscaled int64, scale uint8) Decimal {
	if scale > maxDecimalScale {
		panic("productdata: decimal scale exceeds 18")
	}
	return Decimal{unscaled: unscaled, scale: scale}
}

// ParseDecimal parses a decimal string.
//
// Accepted forms:
//
//	"33000"      -> 33000 (scale 0)
//	"1299,00"    -> 1299.00
//	"1299.00"    -> 1299.00
//	"1.234,56"   -> 1234.56
//	"1,234.56"   -> 1234.56
//	"-0,05"      -> -0.05
//	",5"         -> 0.5
//	"5,"         -> 5
//
// A single comma or dot is treated as the decimal separator. When both are
// present, the rightmost one is the decimal separator and the other is a
// grouping separator. Grouping separators are removed. A single separator with
// exactly three digits after it is NOT treated as grouping: "1,234" is 1.234,
// not 1234. Callers that need to parse grouping-separated thousands without a
// decimal part must normalize the input first.
func ParseDecimal(s string) (Decimal, error) {
	original := s
	s = strings.TrimSpace(s)
	if s == "" {
		return Decimal{}, fmt.Errorf("%w: %q", ErrDecimalSyntax, original)
	}

	negative := false
	switch s[0] {
	case '+':
		s = s[1:]
	case '-':
		negative = true
		s = s[1:]
	}
	if s == "" {
		return Decimal{}, fmt.Errorf("%w: %q", ErrDecimalSyntax, original)
	}

	lastComma := strings.LastIndexByte(s, ',')
	lastDot := strings.LastIndexByte(s, '.')

	var decimalSep byte
	var groupingSep byte
	switch {
	case lastComma >= 0 && lastDot >= 0:
		if lastComma > lastDot {
			decimalSep, groupingSep = ',', '.'
		} else {
			decimalSep, groupingSep = '.', ','
		}
	case lastComma >= 0:
		decimalSep = ','
	case lastDot >= 0:
		decimalSep = '.'
	}

	intPart := s
	fracPart := ""
	if decimalSep != 0 {
		idx := strings.LastIndexByte(s, decimalSep)
		intPart = s[:idx]
		fracPart = s[idx+1:]
	}

	if groupingSep != 0 {
		intPart = strings.ReplaceAll(intPart, string(groupingSep), "")
	}
	intPart = strings.ReplaceAll(intPart, " ", "")

	if intPart == "" && fracPart == "" {
		return Decimal{}, fmt.Errorf("%w: %q", ErrDecimalSyntax, original)
	}
	if !allDigits(intPart) || !allDigits(fracPart) {
		return Decimal{}, fmt.Errorf("%w: %q", ErrDecimalSyntax, original)
	}
	if len(fracPart) > maxDecimalScale {
		return Decimal{}, fmt.Errorf("%w: %q", ErrDecimalScale, original)
	}

	digits := intPart + fracPart
	if digits == "" {
		digits = "0"
	}
	unscaled, err := strconv.ParseInt(digits, 10, 64)
	if err != nil {
		return Decimal{}, fmt.Errorf("%w: %q", ErrDecimalOverflow, original)
	}
	if negative {
		unscaled = -unscaled
	}
	return Decimal{unscaled: unscaled, scale: uint8(len(fracPart))}, nil
}

// MustParseDecimal is ParseDecimal for constants and tests. It panics on error.
func MustParseDecimal(s string) Decimal {
	d, err := ParseDecimal(s)
	if err != nil {
		panic(err)
	}
	return d
}

func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// Unscaled returns the unscaled integer.
func (d Decimal) Unscaled() int64 { return d.unscaled }

// Scale returns the number of fractional digits.
func (d Decimal) Scale() uint8 { return d.scale }

// IsZero reports whether d is zero.
func (d Decimal) IsZero() bool { return d.unscaled == 0 }

// Sign returns -1, 0, or +1.
func (d Decimal) Sign() int {
	switch {
	case d.unscaled < 0:
		return -1
	case d.unscaled > 0:
		return 1
	default:
		return 0
	}
}

// Neg returns -d.
func (d Decimal) Neg() Decimal {
	return Decimal{unscaled: -d.unscaled, scale: d.scale}
}

// Abs returns the absolute value of d.
func (d Decimal) Abs() Decimal {
	if d.unscaled < 0 {
		return Decimal{unscaled: -d.unscaled, scale: d.scale}
	}
	return d
}

// Float64 returns an approximate float64 representation. Do not use it for
// money arithmetic.
func (d Decimal) Float64() float64 {
	return float64(d.unscaled) / math.Pow10(int(d.scale))
}

// String returns the decimal with exactly Scale fractional digits.
func (d Decimal) String() string {
	if d.scale == 0 {
		return strconv.FormatInt(d.unscaled, 10)
	}

	negative := d.unscaled < 0
	digits := strconv.FormatInt(d.unscaled, 10)
	if negative {
		digits = digits[1:]
	}
	for len(digits) <= int(d.scale) {
		digits = "0" + digits
	}
	cut := len(digits) - int(d.scale)
	out := digits[:cut] + "." + digits[cut:]
	if negative {
		out = "-" + out
	}
	return out
}

// GoString implements fmt.GoStringer for debugging.
func (d Decimal) GoString() string {
	return fmt.Sprintf("Decimal(%s)", d.String())
}

// MarshalText implements encoding.TextMarshaler.
func (d Decimal) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (d *Decimal) UnmarshalText(text []byte) error {
	parsed, err := ParseDecimal(string(text))
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// MarshalJSON encodes the decimal as a JSON string to avoid float precision loss.
func (d Decimal) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON accepts a JSON string or a JSON number.
func (d *Decimal) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("%w: empty JSON value", ErrDecimalSyntax)
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		return d.UnmarshalText([]byte(s))
	}
	return d.UnmarshalText(data)
}

// Add returns d + other, aligning scales. It returns an overflow error rather
// than wrapping around.
func (d Decimal) Add(other Decimal) (Decimal, error) {
	scale := d.scale
	if other.scale > scale {
		scale = other.scale
	}
	a, err := d.Rescale(scale)
	if err != nil {
		return Decimal{}, err
	}
	b, err := other.Rescale(scale)
	if err != nil {
		return Decimal{}, err
	}
	sum, ok := addInt64(a.unscaled, b.unscaled)
	if !ok {
		return Decimal{}, ErrDecimalOverflow
	}
	return Decimal{unscaled: sum, scale: scale}, nil
}

// Sub returns d - other. It returns an overflow error rather than wrapping.
func (d Decimal) Sub(other Decimal) (Decimal, error) {
	return d.Add(other.Neg())
}

// Mul returns d × other. The scale of the result is the sum of the scales and
// may be larger than 18; that is an error.
func (d Decimal) Mul(other Decimal) (Decimal, error) {
	scale := int(d.scale) + int(other.scale)
	if scale > maxDecimalScale {
		return Decimal{}, ErrDecimalScale
	}
	product, ok := mulInt64(d.unscaled, other.unscaled)
	if !ok {
		return Decimal{}, ErrDecimalOverflow
	}
	return Decimal{unscaled: product, scale: uint8(scale)}, nil
}

// Rescale returns d with the requested scale. Increasing the scale is exact;
// decreasing it is only allowed when no non-zero digits are lost.
func (d Decimal) Rescale(scale uint8) (Decimal, error) {
	if scale > maxDecimalScale {
		return Decimal{}, ErrDecimalScale
	}
	if scale == d.scale {
		return d, nil
	}
	if scale > d.scale {
		factor := pow10(scale - d.scale)
		value, ok := mulInt64(d.unscaled, factor)
		if !ok {
			return Decimal{}, ErrDecimalOverflow
		}
		return Decimal{unscaled: value, scale: scale}, nil
	}
	factor := pow10(d.scale - scale)
	if d.unscaled%factor != 0 {
		return Decimal{}, ErrDecimalInexact
	}
	return Decimal{unscaled: d.unscaled / factor, scale: scale}, nil
}

// Cmp compares d and other and returns -1, 0, or +1.
func (d Decimal) Cmp(other Decimal) int {
	scale := d.scale
	if other.scale > scale {
		scale = other.scale
	}
	a, err := d.Rescale(scale)
	if err == nil {
		b, otherErr := other.Rescale(scale)
		if otherErr == nil {
			switch {
			case a.unscaled < b.unscaled:
				return -1
			case a.unscaled > b.unscaled:
				return 1
			default:
				return 0
			}
		}
	}
	// Rescaling can overflow for extreme values. Compare exact rationals as a
	// fallback; this is not the hot path.
	return compareExact(d, other)
}

func compareExact(a, b Decimal) int {
	ar := new(big.Rat).SetFrac(big.NewInt(a.unscaled), big.NewInt(pow10(a.scale)))
	br := new(big.Rat).SetFrac(big.NewInt(b.unscaled), big.NewInt(pow10(b.scale)))
	return ar.Cmp(br)
}

// Equal reports whether d and other have the same numeric value.
func (d Decimal) Equal(other Decimal) bool {
	return d.Cmp(other) == 0
}

func pow10(n uint8) int64 {
	if n == 0 {
		return 1
	}
	if n > maxDecimalScale {
		return 0
	}
	return int64(math.Pow10(int(n)))
}

func addInt64(a, b int64) (int64, bool) {
	sum := a + b
	if (b > 0 && sum < a) || (b < 0 && sum > a) {
		return 0, false
	}
	return sum, true
}

func mulInt64(a, b int64) (int64, bool) {
	if a == 0 || b == 0 {
		return 0, true
	}
	if a == math.MinInt64 || b == math.MinInt64 {
		return 0, false
	}
	if a > 0 && b > 0 && a > math.MaxInt64/b {
		return 0, false
	}
	if a < 0 && b < 0 && a < math.MaxInt64/b {
		return 0, false
	}
	if a > 0 && b < 0 && b < math.MinInt64/a {
		return 0, false
	}
	if a < 0 && b > 0 && a < math.MinInt64/b {
		return 0, false
	}
	return a * b, true
}

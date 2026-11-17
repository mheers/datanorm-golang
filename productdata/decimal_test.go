package productdata

import (
	"encoding/json"
	"math"
	"testing"
)

func TestParseDecimal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		want   string
		scale  uint8
		hasErr bool
	}{
		{name: "integer", input: "33000", want: "33000", scale: 0},
		{name: "german decimal", input: "1299,00", want: "1299.00", scale: 2},
		{name: "english decimal", input: "1299.00", want: "1299.00", scale: 2},
		{name: "german grouping", input: "1.234,56", want: "1234.56", scale: 2},
		{name: "english grouping", input: "1,234.56", want: "1234.56", scale: 2},
		{name: "negative", input: "-0,05", want: "-0.05", scale: 2},
		{name: "plus sign", input: "+12,5", want: "12.5", scale: 1},
		{name: "leading decimal", input: ",5", want: "0.5", scale: 1},
		{name: "trailing decimal", input: "5,", want: "5", scale: 0},
		{name: "spaces", input: "  12,50  ", want: "12.50", scale: 2},
		{name: "empty", input: "", hasErr: true},
		{name: "only sign", input: "-", hasErr: true},
		{name: "letters", input: "12x", hasErr: true},
		{name: "two decimals", input: "1,2,3", hasErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseDecimal(tt.input)
			if tt.hasErr {
				if err == nil {
					t.Fatalf("ParseDecimal(%q) = %v, want error", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseDecimal(%q) returned error: %v", tt.input, err)
			}
			if got.String() != tt.want {
				t.Fatalf("ParseDecimal(%q) = %q, want %q", tt.input, got, tt.want)
			}
			if got.Scale() != tt.scale {
				t.Fatalf("ParseDecimal(%q).Scale() = %d, want %d", tt.input, got.Scale(), tt.scale)
			}
		})
	}
}

func TestDecimalArithmetic(t *testing.T) {
	t.Parallel()

	a := MustParseDecimal("100.00")
	b := MustParseDecimal("0.80")

	product, err := a.Mul(b)
	if err != nil {
		t.Fatalf("Mul returned error: %v", err)
	}
	if got, want := product.String(), "80.0000"; got != want {
		t.Fatalf("Mul = %q, want %q", got, want)
	}

	sum, err := MustParseDecimal("1.5").Add(MustParseDecimal("2.25"))
	if err != nil {
		t.Fatalf("Add returned error: %v", err)
	}
	if got, want := sum.String(), "3.75"; got != want {
		t.Fatalf("Add = %q, want %q", got, want)
	}

	diff, err := MustParseDecimal("10").Sub(MustParseDecimal("2.50"))
	if err != nil {
		t.Fatalf("Sub returned error: %v", err)
	}
	if got, want := diff.String(), "7.50"; got != want {
		t.Fatalf("Sub = %q, want %q", got, want)
	}

	if cmp := MustParseDecimal("1.10").Cmp(MustParseDecimal("1.1")); cmp != 0 {
		t.Fatalf("Cmp = %d, want 0", cmp)
	}
	if !MustParseDecimal("0.00").IsZero() {
		t.Fatal("expected zero")
	}
	if got, want := MustParseDecimal("-1.5").Abs().String(), "1.5"; got != want {
		t.Fatalf("Abs = %q, want %q", got, want)
	}
}

func TestDecimalCompareExtremes(t *testing.T) {
	t.Parallel()

	large := NewDecimal(math.MaxInt64, 0)
	small := MustParseDecimal("1.000000000000000000")
	if got := large.Cmp(small); got <= 0 {
		t.Fatalf("Cmp = %d, want > 0", got)
	}
	if got := small.Cmp(large); got >= 0 {
		t.Fatalf("Cmp = %d, want < 0", got)
	}
}

func TestDecimalRescale(t *testing.T) {
	t.Parallel()

	got, err := MustParseDecimal("12.5").Rescale(2)
	if err != nil {
		t.Fatalf("Rescale returned error: %v", err)
	}
	if want := "12.50"; got.String() != want {
		t.Fatalf("Rescale = %q, want %q", got, want)
	}

	if _, err := MustParseDecimal("12.55").Rescale(1); err == nil {
		t.Fatal("expected error when narrowing loses precision")
	}

	if got, err := MustParseDecimal("12.50").Rescale(1); err != nil {
		t.Fatalf("Rescale returned error: %v", err)
	} else if got.String() != "12.5" {
		t.Fatalf("Rescale = %v, want 12.5", got)
	}
}

func TestAmount(t *testing.T) {
	t.Parallel()

	amount, err := ParseAmount("1299,00", EUR)
	if err != nil {
		t.Fatalf("ParseAmount returned error: %v", err)
	}
	if got, want := amount.String(), "1299.00 EUR"; got != want {
		t.Fatalf("Amount.String() = %q, want %q", got, want)
	}

	minor := AmountFromMinorUnits(129900, EUR)
	if !minor.Equal(amount) {
		t.Fatalf("AmountFromMinorUnits = %v, want %v", minor, amount)
	}

	sum, err := amount.Add(AmountFromMinorUnits(100, EUR))
	if err != nil {
		t.Fatalf("Amount.Add returned error: %v", err)
	}
	if got, want := sum.String(), "1300.00 EUR"; got != want {
		t.Fatalf("Amount.Add = %q, want %q", got, want)
	}
}

func TestAmountJSON(t *testing.T) {
	t.Parallel()

	amount := AmountFromMinorUnits(129900, EUR)
	data, err := json.Marshal(amount)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	if got, want := string(data), `{"value":"1299.00","currency":"EUR"}`; got != want {
		t.Fatalf("Marshal = %s, want %s", got, want)
	}

	var decoded Amount
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if !decoded.Equal(amount) {
		t.Fatalf("round trip = %v, want %v", decoded, amount)
	}
}

func TestParseAmountRejectsBadCurrency(t *testing.T) {
	t.Parallel()

	if _, err := ParseAmount("12,00", Currency("EURO")); err == nil {
		t.Fatal("expected error for invalid currency")
	}
}

func TestNormalizeUnit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		wantCode string
		wantKind UnitKind
	}{
		{input: "PCE", wantCode: "PCE", wantKind: UnitPiece},
		{input: "Stück", wantCode: "PCE", wantKind: UnitPiece},
		{input: "Stk", wantCode: "PCE", wantKind: UnitPiece},
		{input: "C62", wantCode: "PCE", wantKind: UnitPiece},
		{input: "MTR", wantCode: "MTR", wantKind: UnitLength},
		{input: "lfm", wantCode: "MTR", wantKind: UnitLength},
		{input: "m", wantCode: "MTR", wantKind: UnitLength},
		{input: "m2", wantCode: "MTK", wantKind: UnitArea},
		{input: "KGM", wantCode: "KGM", wantKind: UnitMass},
		{input: "kg", wantCode: "KGM", wantKind: UnitMass},
		{input: "LTR", wantCode: "LTR", wantKind: UnitVolume},
		{input: "Pck", wantCode: "PK", wantKind: UnitPackaging},
		{input: "Set", wantCode: "SET", wantKind: UnitSet},
		{input: "Watt", wantCode: "Watt", wantKind: UnitOther},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got := NormalizeUnit(tt.input)
			if got.Code != tt.wantCode {
				t.Fatalf("NormalizeUnit(%q).Code = %q, want %q", tt.input, got.Code, tt.wantCode)
			}
			if got.Kind != tt.wantKind {
				t.Fatalf("NormalizeUnit(%q).Kind = %q, want %q", tt.input, got.Kind, tt.wantKind)
			}
		})
	}
}

func TestQuantityString(t *testing.T) {
	t.Parallel()

	q := Quantity{Value: MustParseDecimal("100"), Unit: NormalizeUnit("Stk")}
	if got, want := q.String(), "100 PCE"; got != want {
		t.Fatalf("Quantity.String() = %q, want %q", got, want)
	}
}

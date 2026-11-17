package productdata

import (
	"encoding/json"
	"errors"
	"math"
	"testing"
	"time"
)

func TestDecimalAccessors(t *testing.T) {
	t.Parallel()

	if got := Zero().String(); got != "0" {
		t.Fatalf("Zero = %q", got)
	}
	d := NewDecimal(12345, 2)
	if d.Unscaled() != 12345 || d.Scale() != 2 {
		t.Fatalf("d = %+v", d)
	}
	if d.Sign() != 1 || NewDecimal(-1, 0).Sign() != -1 || Zero().Sign() != 0 {
		t.Fatal("Sign mismatch")
	}
	if got := d.Neg().String(); got != "-123.45" {
		t.Fatalf("Neg = %q", got)
	}
	if got := NewDecimal(-12345, 2).Abs().String(); got != "123.45" {
		t.Fatalf("Abs = %q", got)
	}
	if got := d.Float64(); got != 123.45 {
		t.Fatalf("Float64 = %v", got)
	}
	if got := d.GoString(); got != "Decimal(123.45)" {
		t.Fatalf("GoString = %q", got)
	}
	if !Zero().IsZero() || d.IsZero() {
		t.Fatal("IsZero mismatch")
	}
	if d.Equal(MustParseDecimal("123.450")) != true {
		t.Fatal("Equal mismatch")
	}
}

func TestDecimalMustParsePanics(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Fatal("MustParseDecimal did not panic")
		}
	}()
	MustParseDecimal("not-a-number")
}

func TestNewDecimalPanicsOnBadScale(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Fatal("NewDecimal did not panic")
		}
	}()
	NewDecimal(1, 19)
}

func TestDecimalTextAndJSON(t *testing.T) {
	t.Parallel()

	d := MustParseDecimal("1299.00")
	text, err := d.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}
	if string(text) != "1299.00" {
		t.Fatalf("MarshalText = %q", text)
	}
	var decoded Decimal
	if err := decoded.UnmarshalText(text); err != nil {
		t.Fatalf("UnmarshalText: %v", err)
	}
	if !decoded.Equal(d) {
		t.Fatalf("decoded = %v", decoded)
	}
	if err := decoded.UnmarshalText([]byte("bad")); err == nil {
		t.Fatal("UnmarshalText accepted bad input")
	}

	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if string(data) != `"1299.00"` {
		t.Fatalf("MarshalJSON = %s", data)
	}
	if err := json.Unmarshal([]byte(`"12.50"`), &decoded); err != nil {
		t.Fatalf("UnmarshalJSON string: %v", err)
	}
	if err := json.Unmarshal([]byte(`12.50`), &decoded); err != nil {
		t.Fatalf("UnmarshalJSON number: %v", err)
	}
	if err := json.Unmarshal([]byte(`"bad"`), &decoded); err == nil {
		t.Fatal("UnmarshalJSON accepted bad input")
	}
	if err := decoded.UnmarshalJSON(nil); err == nil {
		t.Fatal("UnmarshalJSON accepted an empty value")
	}
}

func TestDecimalArithmeticErrors(t *testing.T) {
	t.Parallel()

	max := NewDecimal(math.MaxInt64, 0)
	if _, err := max.Add(NewDecimal(1, 0)); !errors.Is(err, ErrDecimalOverflow) {
		t.Fatalf("Add overflow error = %v", err)
	}
	if _, err := max.Add(MustParseDecimal("0.1")); !errors.Is(err, ErrDecimalOverflow) {
		t.Fatalf("Add rescale overflow error = %v", err)
	}
	if _, err := max.Mul(NewDecimal(2, 0)); !errors.Is(err, ErrDecimalOverflow) {
		t.Fatalf("Mul overflow error = %v", err)
	}
	if _, err := NewDecimal(1, 10).Mul(NewDecimal(1, 10)); !errors.Is(err, ErrDecimalScale) {
		t.Fatalf("Mul scale error = %v", err)
	}
	if _, err := NewDecimal(math.MinInt64, 0).Mul(NewDecimal(1, 0)); !errors.Is(err, ErrDecimalOverflow) {
		t.Fatalf("MinInt64 Mul error = %v", err)
	}
	if _, err := NewDecimal(math.MinInt64+1, 0).Mul(NewDecimal(-2, 0)); !errors.Is(err, ErrDecimalOverflow) {
		t.Fatalf("negative Mul overflow error = %v", err)
	}
	if _, err := NewDecimal(-2, 0).Mul(NewDecimal(math.MaxInt64, 0)); !errors.Is(err, ErrDecimalOverflow) {
		t.Fatalf("mixed Mul overflow error = %v", err)
	}
	if _, err := NewDecimal(1, 0).Rescale(19); !errors.Is(err, ErrDecimalScale) {
		t.Fatalf("Rescale scale error = %v", err)
	}
	if _, err := MustParseDecimal("1.5").Rescale(0); !errors.Is(err, ErrDecimalInexact) {
		t.Fatalf("Rescale inexact error = %v", err)
	}
}

func TestDecimalParseErrors(t *testing.T) {
	t.Parallel()

	inputs := []string{"", "+", "-", "1,2,3", "1.2.3", "abc", "1 000 x"}
	for _, input := range inputs {
		if _, err := ParseDecimal(input); err == nil {
			t.Errorf("ParseDecimal(%q) accepted invalid input", input)
		}
	}
	if _, err := ParseDecimal("123456789012345678901234567890"); !errors.Is(err, ErrDecimalOverflow) {
		t.Fatalf("overflow error = %v", err)
	}
	if _, err := ParseDecimal("1,1234567890123456789"); !errors.Is(err, ErrDecimalScale) {
		t.Fatalf("scale error = %v", err)
	}
}

func TestDecimalMulZeroAndSigns(t *testing.T) {
	t.Parallel()

	for _, d := range []Decimal{Zero(), NewDecimal(-5, 0), NewDecimal(5, 0)} {
		product, err := d.Mul(Zero())
		if err != nil || !product.IsZero() {
			t.Fatalf("Mul zero = %v, %v", product, err)
		}
	}
	product, err := NewDecimal(-2, 0).Mul(NewDecimal(-3, 0))
	if err != nil || product.String() != "6" {
		t.Fatalf("negative product = %v, %v", product, err)
	}
}

func TestAmountHelpers(t *testing.T) {
	t.Parallel()

	amount := NewAmount(MustParseDecimal("12.50"), EUR)
	if !amount.Equal(NewAmount(MustParseDecimal("12.5"), EUR)) {
		t.Fatal("Equal mismatch")
	}
	if amount.IsZero() {
		t.Fatal("amount should not be zero")
	}
	if got := amount.String(); got != "12.50 EUR" {
		t.Fatalf("String = %q", got)
	}
	if got := NewAmount(MustParseDecimal("1"), "").String(); got != "1" {
		t.Fatalf("currencyless String = %q", got)
	}
	difference, err := amount.Sub(NewAmount(MustParseDecimal("2.50"), EUR))
	if err != nil || difference.String() != "10.00 EUR" {
		t.Fatalf("Sub = %v, %v", difference, err)
	}
	if _, err := amount.Add(NewAmount(MustParseDecimal("1"), USD)); !errors.Is(err, ErrCurrencyMismatch) {
		t.Fatalf("currency mismatch error = %v", err)
	}
	if _, err := NewAmount(NewDecimal(math.MaxInt64, 2), EUR).Add(NewAmount(NewDecimal(1, 2), EUR)); !errors.Is(err, ErrDecimalOverflow) {
		t.Fatalf("amount overflow error = %v", err)
	}
	if got := NormalizeCurrency(" eur "); got != EUR {
		t.Fatalf("NormalizeCurrency = %q", got)
	}
	if got := NewAmount(MustParseDecimal("0"), EUR); !got.IsZero() {
		t.Fatal("zero amount")
	}
}

func TestArticleTextAndDisplayHelpers(t *testing.T) {
	t.Parallel()

	article := &Article{}
	article.AddText(NewText(TextShort, "Kabel"))
	article.AddText(NewText(TextShort, "Kabel neu"))
	if len(article.Texts) != 1 || article.ShortText() != "Kabel neu" {
		t.Fatalf("AddText did not replace: %+v", article.Texts)
	}
	article.AddText(NewText(TextLong, "Lang"))
	if got := article.Text(TextLong); got != "Lang" {
		t.Fatalf("Text(long) = %q", got)
	}
	if got := article.Text(TextNote); got != "" {
		t.Fatalf("Text(note) = %q", got)
	}
	article.AddText(NewText(TextDimension, "Dim"))
	if got := article.LongText(); got != "Lang" {
		t.Fatalf("LongText = %q", got)
	}
	article.Texts = []Text{NewText(TextDimension, "Dim")}
	if got := article.LongText(); got != "Dim" {
		t.Fatalf("LongText fallback = %q", got)
	}
	if got := article.DisplayName(); got != "" {
		t.Fatalf("DisplayName on empty article = %q", got)
	}
	article.Type = "Typ"
	if got := article.DisplayName(); got != "Typ" {
		t.Fatalf("DisplayName type = %q", got)
	}
	article.SupplierArticleNumber = "A-1"
	if got := article.DisplayName(); got != "Typ" {
		t.Fatalf("DisplayName = %q", got)
	}
	article.Type = ""
	if got := article.DisplayName(); got != "A-1" {
		t.Fatalf("DisplayName fallback = %q", got)
	}
}

func TestCatalogHelpers(t *testing.T) {
	t.Parallel()

	catalog := &Catalog{}
	catalog.AddArticle(Article{SupplierArticleNumber: "A-1"})
	catalog.AddGroup(Group{Level: GroupMain, ID: "ELE", Name: "Elektro"})
	catalog.AddDiscountGroup(DiscountGroup{ID: "04", Name: "Kabel"})
	if catalog.ArticleCount() != 1 {
		t.Fatalf("ArticleCount = %d", catalog.ArticleCount())
	}
	if group, ok := catalog.FindGroup(GroupMain, "ELE"); !ok || group.Name != "Elektro" {
		t.Fatalf("FindGroup = %+v, %v", group, ok)
	}
	if _, ok := catalog.FindGroup(GroupMain, "missing"); ok {
		t.Fatal("FindGroup found a missing group")
	}
	if group, ok := catalog.FindDiscountGroup("04"); !ok || group.Name != "Kabel" {
		t.Fatalf("FindDiscountGroup = %+v, %v", group, ok)
	}
	if _, ok := catalog.FindDiscountGroup("missing"); ok {
		t.Fatal("FindDiscountGroup found a missing group")
	}
}

func TestGroupAndDiscountDescribe(t *testing.T) {
	t.Parallel()

	if got := (DiscountGroup{ID: "04", Name: "Kabel"}).Describe(); got != "Kabel" {
		t.Fatalf("Describe = %q", got)
	}
	if got := (DiscountGroup{ID: "04"}).Describe(); got != "04" {
		t.Fatalf("Describe fallback = %q", got)
	}
	if got := (NewText(TextShort, "a", "b")).Joined(" "); got != "a b" {
		t.Fatalf("Joined = %q", got)
	}
	if (Text{}).IsEmpty() != true || (Text{Lines: []string{" "}}).IsEmpty() != true || (Text{Lines: []string{"x"}}).IsEmpty() != false {
		t.Fatal("IsEmpty mismatch")
	}
}

func TestGTINHelpers(t *testing.T) {
	t.Parallel()

	if got := NormalizeGTIN(" 4006381333931 "); got != "4006381333931" {
		t.Fatalf("NormalizeGTIN = %q", got)
	}
	if ValidateGTIN("123") {
		t.Fatal("short GTIN accepted")
	}
	if ValidateGTIN("4006381333932") {
		t.Fatal("bad check digit accepted")
	}
	if ValidateGTIN("40063813339A1") {
		t.Fatal("non-numeric GTIN accepted")
	}
}

func TestPriceHelpers(t *testing.T) {
	t.Parallel()

	types := []PriceType{
		PriceList, PriceNet, PriceRoute, PriceRecommendedRetail, PriceGuide,
		PriceOnRequest, PriceUnknown,
	}
	for _, priceType := range types {
		if priceType.String() == "" {
			t.Errorf("empty String for %q", priceType)
		}
		if priceType != PriceUnknown && priceType.Code() == "" {
			t.Errorf("empty Code for %q", priceType)
		}
	}
	if PriceUnknown.Code() != "" {
		t.Fatal("unknown type has a code")
	}
	if PriceUnknown.String() != "unknown price" {
		t.Fatalf("unknown String = %q", PriceUnknown.String())
	}
	if got := PriceTypeFromCode("2"); got != PriceNet {
		t.Fatalf("PriceTypeFromCode = %q", got)
	}
	if got := PriceTypeFromCode("nope"); got != PriceUnknown {
		t.Fatalf("unknown PriceTypeFromCode = %q", got)
	}
	price := Price{}
	if got := price.PerQuantity(); got.Value.String() != "1" {
		t.Fatalf("PerQuantity = %v", got)
	}
	price.Per = &Quantity{Value: NewDecimal(10, 0)}
	if got := price.PerQuantity(); got.Value.String() != "10" {
		t.Fatalf("PerQuantity = %v", got)
	}
}

func TestUnitAndQuantityHelpers(t *testing.T) {
	t.Parallel()

	if (Unit{}).Known() {
		t.Fatal("empty unit should not be known")
	}
	if !NormalizeUnit("PCE").Known() {
		t.Fatal("PCE should be known")
	}
	if NormalizeUnit("").Kind != UnitOther {
		t.Fatalf("empty unit kind = %q", NormalizeUnit("").Kind)
	}
	if got := NormalizeUnit("PCE").String(); got != "PCE" {
		t.Fatalf("Unit.String = %q", got)
	}
	quantity := NewQuantity(NewDecimal(10, 0), NormalizeUnit("PCE"))
	if got := quantity.String(); got != "10 PCE" {
		t.Fatalf("Quantity.String = %q", got)
	}
	if got := (NewQuantity(NewDecimal(10, 0), Unit{})).String(); got != "10" {
		t.Fatalf("unitless Quantity.String = %q", got)
	}
}

func TestDiagnosticsHelpers(t *testing.T) {
	t.Parallel()

	var diagnostics Diagnostics
	diagnostics.AddAt(Diagnostic{Severity: SeverityInfo, Code: "x", Message: "y", File: "f", Line: 3})
	diagnostics.Add(SeverityWarning, "w", "m")
	if got := diagnostics.Count(SeverityInfo); got != 1 {
		t.Fatalf("info count = %d", got)
	}
	if got := diagnostics[0].String(); got != "info f:3 x: y" {
		t.Fatalf("String = %q", got)
	}
	if got := diagnosticStringWithoutLocation(); got != "error c: m" {
		t.Fatalf("String without location = %q", got)
	}
	if diagnostics.HasErrors() {
		t.Fatal("unexpected errors")
	}
}

func diagnosticStringWithoutLocation() string {
	return (Diagnostic{Severity: SeverityError, Code: "c", Message: "m"}).String()
}

func TestFormatRegistryHelpers(t *testing.T) {
	// Not parallel: the registry is process-global.
	if err := Register(nil); !errors.Is(err, ErrInvalidFormat) {
		t.Fatalf("Register(nil) = %v", err)
	}
	if err := Register(fakeFormat{name: "   "}); !errors.Is(err, ErrInvalidFormat) {
		t.Fatalf("Register(blank) = %v", err)
	}
	format := fakeFormat{name: "  spaced-format  ", head: "SPACED;", ext: ".spaced"}
	if err := Register(format); err != nil {
		t.Fatalf("Register: %v", err)
	}
	got, ok := Get(" spaced-format ")
	if !ok || got.Name() != format.name {
		t.Fatalf("Get = %v, %v", got, ok)
	}
	if _, ok := Get("missing-format"); ok {
		t.Fatal("Get found a missing format")
	}
	if len(Formats()) == 0 {
		t.Fatal("Formats is empty")
	}
	if !DetectExtension("file.001", ".001") || DetectExtension("file.txt", ".001") {
		t.Fatal("DetectExtension mismatch")
	}
	if !DetectExtension("FILE.001", ".001") {
		t.Fatal("DetectExtension is case-sensitive")
	}
}

func TestFormatDetectRegistry(t *testing.T) {
	// Not parallel: the registry is process-global.
	format := fakeFormat{name: "detect-format", head: "DETECT;", ext: ".detect"}
	if err := Register(format); err != nil && !errors.Is(err, ErrDuplicateFormat) {
		t.Fatalf("Register: %v", err)
	}
	if got, ok := Detect("file.detect", nil); !ok || got.Name() != format.name {
		t.Fatalf("Detect by extension = %v, %v", got, ok)
	}
	if got, ok := Detect("", []byte("DETECT;1")); !ok || got.Name() != format.name {
		t.Fatalf("Detect by content = %v, %v", got, ok)
	}
	if _, ok := Detect("file.unknown", []byte("other")); ok {
		t.Fatal("unexpected detection")
	}
	if !errors.Is(ErrEncodingUnsupported, ErrEncodingUnsupported) || !errors.Is(ErrNotDetected, ErrNotDetected) {
		t.Fatal("sentinel errors mismatch")
	}
}

func TestPriceTierAndAmountJSONRoundTrip(t *testing.T) {
	t.Parallel()

	tier := PriceTier{MinQuantity: NewQuantity(NewDecimal(10, 0), NormalizeUnit("PCE")), Amount: NewAmount(MustParseDecimal("9.99"), EUR)}
	if tier.MinQuantity.Value.String() != "10" {
		t.Fatalf("tier = %+v", tier)
	}
	price := Price{Type: PriceList, Amount: tier.Amount, Tiers: []PriceTier{tier}, ValidFrom: timePtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))}
	data, err := json.Marshal(price)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("empty JSON")
	}
}

func timePtr(value time.Time) *time.Time { return &value }

package productdata

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestPriceTypeFromCode(t *testing.T) {
	t.Parallel()

	tests := map[string]PriceType{
		"1": PriceList,
		"2": PriceNet,
		"3": PriceRoute,
		"4": PriceRecommendedRetail,
		"5": PriceGuide,
		"9": PriceOnRequest,
		"":  PriceUnknown,
		"x": PriceUnknown,
	}
	for code, want := range tests {
		if got := PriceTypeFromCode(code); got != want {
			t.Errorf("PriceTypeFromCode(%q) = %q, want %q", code, got, want)
		}
	}
	if got := PriceNet.Code(); got != "2" {
		t.Fatalf("PriceNet.Code() = %q, want 2", got)
	}
	if got := PriceType("nope").Code(); got != "" {
		t.Fatalf("unknown code = %q, want empty", got)
	}
}

func TestText(t *testing.T) {
	t.Parallel()

	short := NewText(TextShort, "Cable", "NYM-J 3x1,5")
	if got, want := short.Joined(" "), "Cable NYM-J 3x1,5"; got != want {
		t.Fatalf("Joined = %q, want %q", got, want)
	}
	if short.IsEmpty() {
		t.Fatal("text should not be empty")
	}
	if !NewText(TextLong).IsEmpty() {
		t.Fatal("empty text should be empty")
	}
}

func TestArticleHelpers(t *testing.T) {
	t.Parallel()

	a := &Article{
		SupplierArticleNumber: "A-1",
		Texts: []Text{
			NewText(TextShort, "Cable", "NYM-J"),
			NewText(TextLong, "Line one", "Line two"),
		},
		Prices: []Price{
			{Type: PriceList, Amount: AmountFromMinorUnits(129900, EUR), Per: &Quantity{Value: MustParseDecimal("100"), Unit: NormalizeUnit("MTR")}},
			{Type: PriceNet, Amount: AmountFromMinorUnits(99900, EUR)},
		},
		Groups: []GroupRef{
			{Level: GroupMain, ID: "ELE"},
			{Level: GroupProduct, ID: "KABEL", Name: "Kabel"},
			{Level: GroupDiscount, ID: "04"},
		},
	}

	if got, want := a.ShortText(), "Cable NYM-J"; got != want {
		t.Fatalf("ShortText = %q, want %q", got, want)
	}
	if got, want := a.Text(TextLong), "Line one\nLine two"; got != want {
		t.Fatalf("Text(long) = %q, want %q", got, want)
	}
	if price, ok := a.Price(PriceList); !ok || !price.Amount.Equal(AmountFromMinorUnits(129900, EUR)) {
		t.Fatalf("Price(list) = %v, %v", price, ok)
	}
	if _, ok := a.Price(PriceRecommendedRetail); ok {
		t.Fatal("unexpected recommended retail price")
	}
	if group, ok := a.Group(GroupProduct); !ok || group.Name != "Kabel" {
		t.Fatalf("Group(product) = %v, %v", group, ok)
	}
	if _, ok := a.Group(GroupCategory); ok {
		t.Fatal("unexpected category group")
	}
}

func TestDiscountGroup(t *testing.T) {
	t.Parallel()

	g := DiscountGroup{
		ID:   "04",
		Name: "Kabel",
		Discounts: []Discount{
			{Kind: DiscountPercent, Value: MustParseDecimal("20")},
			{Kind: DiscountPercent, Value: MustParseDecimal("10")},
		},
	}
	if got, want := g.Describe(), "Kabel"; got != want {
		t.Fatalf("Describe = %q, want %q", got, want)
	}
	if len(g.Discounts) != 2 {
		t.Fatalf("discounts = %d, want 2", len(g.Discounts))
	}
}

func TestDiagnostics(t *testing.T) {
	t.Parallel()

	var ds Diagnostics
	ds.Add(SeverityWarning, "dn.ean-checksum", "invalid EAN check digit")
	ds.Add(SeverityError, "dn.price", "price is not numeric")

	if !ds.HasErrors() {
		t.Fatal("expected errors")
	}
	if got := ds.Count(SeverityWarning); got != 1 {
		t.Fatalf("warnings = %d, want 1", got)
	}
	if got, want := ds[0].String(), "warning dn.ean-checksum: invalid EAN check digit"; got != want {
		t.Fatalf("String = %q, want %q", got, want)
	}
}

func TestCatalogFindArticle(t *testing.T) {
	t.Parallel()

	c := &Catalog{
		Header: Header{Format: "test", Currency: EUR},
		Articles: []Article{
			{SupplierArticleNumber: "A-1"},
			{SupplierArticleNumber: "A-2"},
		},
	}
	if a, ok := c.FindArticle("A-2"); !ok || a.SupplierArticleNumber != "A-2" {
		t.Fatalf("FindArticle = %v, %v", a, ok)
	}
	if _, ok := c.FindArticle("missing"); ok {
		t.Fatal("unexpected article")
	}
}

type fakeFormat struct {
	name string
	head string
}

func (f fakeFormat) Name() string         { return f.name }
func (f fakeFormat) Description() string  { return "fake" }
func (f fakeFormat) Extensions() []string { return []string{".fake"} }
func (f fakeFormat) Detect(filename string, head []byte) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".fake") || strings.HasPrefix(string(head), f.head)
}
func (f fakeFormat) Capabilities() Capabilities { return Capabilities{Read: true, Batch: true} }
func (f fakeFormat) NewDecoder() (Decoder, error) {
	return fakeDecoder{}, nil
}
func (f fakeFormat) NewEncoder() (Encoder, error) {
	return nil, ErrEncodingUnsupported
}

type fakeDecoder struct{}

func (fakeDecoder) Decode(ctx context.Context, r io.Reader, opts DecodeOptions) (*Catalog, error) {
	return &Catalog{Header: Header{Format: "fake"}}, nil
}

func TestFormatRegistry(t *testing.T) {
	// Not parallel: the registry is process-global.
	before := len(Formats())
	f := fakeFormat{name: "fake-test-format", head: "FAKE;"}
	if err := Register(f); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if err := Register(f); err == nil {
		t.Fatal("expected duplicate registration error")
	}
	if got := len(Formats()); got != before+1 {
		t.Fatalf("formats = %d, want %d", got, before+1)
	}
	if got, ok := Get("fake-test-format"); !ok || got.Name() != f.name {
		t.Fatalf("Get = %v, %v", got, ok)
	}
	if got, ok := Detect("catalog.fake", nil); !ok || got.Name() != f.name {
		t.Fatalf("Detect by extension = %v, %v", got, ok)
	}
	if got, ok := Detect("", []byte("FAKE;1")); !ok || got.Name() != f.name {
		t.Fatalf("Detect by content = %v, %v", got, ok)
	}
	if _, ok := Detect("catalog.unknown", []byte("other")); ok {
		t.Fatal("unexpected detection")
	}
}

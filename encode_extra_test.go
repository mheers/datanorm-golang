package datanorm

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mheers/datanorm-golang/charset"
	"github.com/mheers/datanorm-golang/productdata"
)

func encodeToString(t *testing.T, catalog *productdata.Catalog, opts productdata.EncodeOptions) string {
	t.Helper()
	var out strings.Builder
	if err := NewEncoder().Encode(context.Background(), &out, catalog, opts); err != nil {
		t.Fatalf("Encode returned error: %v", err)
	}
	return out.String()
}

func TestEncodeErrorBranches(t *testing.T) {
	t.Parallel()

	var out strings.Builder
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := NewEncoder().Encode(ctx, &out, sampleCatalog(), productdata.EncodeOptions{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled context error = %v", err)
	}
	if err := NewEncoder().Encode(context.Background(), &out, nil, productdata.EncodeOptions{}); !errors.Is(err, ErrEncodingFailed) {
		t.Fatalf("nil catalog error = %v", err)
	}
	if err := NewEncoder().Encode(context.Background(), &out, sampleCatalog(), productdata.EncodeOptions{Charset: "klingon"}); err == nil {
		t.Fatal("accepted an unknown charset")
	}
	if err := NewEncoder().Encode(context.Background(), &out, sampleCatalog(), productdata.EncodeOptions{Version: "4.0"}); !errors.Is(err, ErrEncodingFailed) {
		t.Fatalf("invalid version error = %v", err)
	}
	if err := NewEncoder().Encode(context.Background(), &out, sampleCatalog(), productdata.EncodeOptions{Version: "050"}); err != nil {
		t.Fatalf("valid version error = %v", err)
	}
}

func TestEncodeOptionEffects(t *testing.T) {
	t.Parallel()

	priceDate := time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC)
	createdAt := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	catalog := sampleCatalog()
	catalog.Header.PriceDate = &priceDate
	out := encodeToString(t, catalog, productdata.EncodeOptions{Charset: "utf-8", LineEnding: "\n"})
	if !strings.Contains(out, "V;050;A;20260203;") {
		t.Fatalf("header does not use PriceDate:\n%s", out)
	}
	if !strings.Contains(out, "\n") || strings.Contains(out, "\r\n") {
		t.Fatal("line ending option was not applied")
	}

	catalog = sampleCatalog()
	catalog.Header.CreatedAt = &createdAt
	out = encodeToString(t, catalog, productdata.EncodeOptions{})
	if !strings.Contains(out, "V;050;A;20260102;") {
		t.Fatalf("header does not use CreatedAt:\n%s", out)
	}

	catalog = sampleCatalog()
	catalog.Header.Currency = "EURO"
	out = encodeToString(t, catalog, productdata.EncodeOptions{})
	if !strings.Contains(out, ";EUR;") {
		t.Fatalf("invalid currency was not replaced:\n%s", out)
	}
}

func TestEncodeValidationErrors(t *testing.T) {
	t.Parallel()

	base := func() *productdata.Catalog {
		return &productdata.Catalog{
			Header: productdata.Header{Currency: productdata.EUR, Supplier: productdata.Supplier{Name: "Beispiel"}},
			Articles: []productdata.Article{
				{SupplierArticleNumber: "A-1", Unit: productdata.NormalizeUnit("PCE"), Texts: []productdata.Text{productdata.NewText(productdata.TextShort, "Kabel")}},
			},
		}
	}

	tests := []struct {
		name   string
		mutate func(*productdata.Catalog)
	}{
		{name: "empty number", mutate: func(c *productdata.Catalog) { c.Articles[0].SupplierArticleNumber = "" }},
		{name: "long number", mutate: func(c *productdata.Catalog) { c.Articles[0].SupplierArticleNumber = strings.Repeat("x", 16) }},
		{name: "long short text", mutate: func(c *productdata.Catalog) {
			c.Articles[0].Texts = []productdata.Text{productdata.NewText(productdata.TextShort, strings.Repeat("x", 41))}
		}},
		{name: "long text too long", mutate: func(c *productdata.Catalog) {
			c.Articles[0].Texts = append(c.Articles[0].Texts,
				productdata.NewText(productdata.TextLong, strings.Repeat("x", 40*100)))
		}},
		{name: "dimension too long", mutate: func(c *productdata.Catalog) {
			c.Articles[0].Texts = append(c.Articles[0].Texts,
				productdata.NewText(productdata.TextDimension, strings.Repeat("x", 40*100)))
		}},
		{name: "semicolon in article", mutate: func(c *productdata.Catalog) {
			c.Articles[0].Texts = []productdata.Text{productdata.NewText(productdata.TextShort, "Kabel; NYM")}
		}},
		{name: "semicolon in header", mutate: func(c *productdata.Catalog) { c.Header.Description = "Preis;liste" }},
		{name: "unencodable rune", mutate: func(c *productdata.Catalog) {
			c.Articles[0].Texts = []productdata.Text{productdata.NewText(productdata.TextShort, "Preis 5€")}
		}},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			catalog := base()
			tt.mutate(catalog)
			var out strings.Builder
			if err := NewEncoder().Encode(context.Background(), &out, catalog, productdata.EncodeOptions{}); !errors.Is(err, ErrEncodingFailed) {
				t.Fatalf("error = %v, want ErrEncodingFailed", err)
			}
			if out.Len() != 0 {
				t.Fatalf("wrote %d bytes before failing", out.Len())
			}
		})
	}
}

func TestEncodeGroupVariants(t *testing.T) {
	t.Parallel()

	catalog := &productdata.Catalog{
		Header: productdata.Header{Currency: productdata.EUR, Supplier: productdata.Supplier{Name: "Beispiel"}},
		Groups: []productdata.Group{
			{Level: productdata.GroupMain, ID: "ELE", Name: "Elektro"},
			{Level: productdata.GroupProduct, ID: "KABEL", Name: "Kabel", ParentID: "ELE"},
			{Level: productdata.GroupProduct, ID: "OHNE", Name: "Ohne Hauptgruppe"},
			{Level: productdata.GroupMain, ID: "", Name: "Leer"},
		},
		DiscountGroups: []productdata.DiscountGroup{
			{ID: "04", Name: "Ohne Rabatt"},
			{ID: "05", Name: "Faktor", Discounts: []productdata.Discount{{Kind: productdata.DiscountFactor, Value: productdata.MustParseDecimal("0.8")}}},
			{ID: "06", Name: "Zuschlag", Discounts: []productdata.Discount{{Kind: productdata.DiscountSurcharge, Value: productdata.MustParseDecimal("5")}}},
			{ID: "07", Name: "Prozent", Discounts: []productdata.Discount{{Kind: productdata.DiscountPercent, Value: productdata.MustParseDecimal("20")}}},
			{ID: "", Name: "Leer"},
		},
	}

	var groups strings.Builder
	if err := NewEncoder().EncodeGroups(context.Background(), &groups, catalog, productdata.EncodeOptions{}); err != nil {
		t.Fatalf("EncodeGroups: %v", err)
	}
	groupOutput := groups.String()
	for _, want := range []string{"S;ELE;;Elektro;", "S;ELE;KABEL;Kabel;", "S;OHNE;;Ohne Hauptgruppe;"} {
		if !strings.Contains(groupOutput, want) {
			t.Errorf("groups missing %q:\n%s", want, groupOutput)
		}
	}
	if strings.Contains(groupOutput, "S;;;") {
		t.Errorf("empty group ID was written:\n%s", groupOutput)
	}

	var discounts strings.Builder
	if err := NewEncoder().EncodeDiscounts(context.Background(), &discounts, catalog, productdata.EncodeOptions{}); err != nil {
		t.Fatalf("EncodeDiscounts: %v", err)
	}
	discountOutput := discounts.String()
	for _, want := range []string{"R;04;1;0;Ohne Rabatt;", "R;05;2;0.8;Faktor;", "R;06;3;5;Zuschlag;", "R;07;1;20;Prozent;"} {
		if !strings.Contains(discountOutput, want) {
			t.Errorf("discounts missing %q:\n%s", want, discountOutput)
		}
	}
	for _, line := range strings.Split(discountOutput, "\n") {
		if strings.HasPrefix(line, "R;;") {
			t.Errorf("empty discount ID was written: %q", line)
		}
	}
}

func TestEncodePriceVariants(t *testing.T) {
	t.Parallel()

	catalog := &productdata.Catalog{
		Header: productdata.Header{Currency: productdata.EUR, Supplier: productdata.Supplier{Name: "Beispiel"}},
		Articles: []productdata.Article{
			{
				SupplierArticleNumber: "A-1",
				Prices: []productdata.Price{
					{Type: productdata.PriceNet, Amount: productdata.NewAmount(productdata.MustParseDecimal("9.99"), productdata.EUR), Per: &productdata.Quantity{Value: productdata.MustParseDecimal("10")}},
					{Type: productdata.PriceOnRequest, Amount: productdata.NewAmount(productdata.Zero(), productdata.EUR)},
					{Type: productdata.PriceUnknown, Amount: productdata.NewAmount(productdata.MustParseDecimal("1"), productdata.EUR)},
					{Type: productdata.PriceGuide, Amount: productdata.NewAmount(productdata.MustParseDecimal("1.234"), productdata.EUR)},
				},
				Groups: []productdata.GroupRef{{Level: productdata.GroupDiscount, ID: "04"}},
			},
		},
	}
	var out strings.Builder
	if err := NewEncoder().EncodePrices(context.Background(), &out, catalog, productdata.EncodeOptions{}); err != nil {
		t.Fatalf("EncodePrices: %v", err)
	}
	output := out.String()
	for _, want := range []string{"P;A-1;2;10;999;04", "P;A-1;9;1;;04", "P;A-1;5;1;1.234;04"} {
		if !strings.Contains(output, want) {
			t.Errorf("prices missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "P;A-1;;") {
		t.Errorf("unknown price type was written:\n%s", output)
	}
}

func TestEncodeArticleStatusesAndTextMarkers(t *testing.T) {
	t.Parallel()

	catalog := &productdata.Catalog{
		Header: productdata.Header{Currency: productdata.EUR, Supplier: productdata.Supplier{Name: "Beispiel"}},
		Articles: []productdata.Article{
			{SupplierArticleNumber: "A-1", Status: productdata.StatusDeleted, Texts: []productdata.Text{productdata.NewText(productdata.TextShort, "Kabel")}},
			{SupplierArticleNumber: "A-2", Status: productdata.StatusChanged, Texts: []productdata.Text{productdata.NewText(productdata.TextShort, "Kabel"), productdata.NewText(productdata.TextLong, "Lang")}},
			{SupplierArticleNumber: "A-3", Status: productdata.StatusDiscontinued, Texts: []productdata.Text{productdata.NewText(productdata.TextShort, "Kabel"), productdata.NewText(productdata.TextDimension, "Dim")}},
			{SupplierArticleNumber: "A-4", Status: productdata.StatusActive, Texts: []productdata.Text{productdata.NewText(productdata.TextShort, "Kabel"), productdata.NewText(productdata.TextLong, "Lang"), productdata.NewText(productdata.TextDimension, "Dim")}},
			{SupplierArticleNumber: "A-5", Status: productdata.StatusActive, Texts: []productdata.Text{productdata.NewText(productdata.TextShort, "Kabel"), productdata.NewText(productdata.TextLong, "Lang")}, Extra: map[string]string{"datanorm.text_marker": "4"}},
			{SupplierArticleNumber: "A-6", Status: productdata.StatusActive, Texts: []productdata.Text{productdata.NewText(productdata.TextShort, "Kabel"), productdata.NewText(productdata.TextLong, "Lang")}, Extra: map[string]string{"datanorm.text_marker": "0"}},
		},
	}
	out := encodeToString(t, catalog, productdata.EncodeOptions{})
	file, err := Parse(strings.NewReader(out))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	flags := map[string]string{}
	for _, article := range file.Articles() {
		flags[article.Number] = string(article.ProcessingFlag)
	}
	if flags["A-1"] != "L" || flags["A-2"] != "A" || flags["A-3"] != "A" || flags["A-4"] != "N" {
		t.Fatalf("flags = %v", flags)
	}
	markers := map[string]string{}
	for _, article := range file.Articles() {
		markers[article.Number] = string(article.TextMarker)
	}
	if markers["A-2"] != "4" || markers["A-3"] != "5" || markers["A-4"] != "6" {
		t.Fatalf("computed markers = %v", markers)
	}
	if markers["A-5"] != "4" {
		t.Fatalf("stored consistent marker = %q, want 4", markers["A-5"])
	}
	if markers["A-6"] != "4" {
		t.Fatalf("inconsistent stored marker = %q, want computed 4", markers["A-6"])
	}
}

func TestEncodeHelpers(t *testing.T) {
	t.Parallel()

	if got := headerLine("A", time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), productdata.EUR,
		productdata.Header{Description: "", Info: []string{"info1", "info2", "info3"}}); !strings.Contains(got, ";info1;info2;info3;") {
		t.Fatalf("headerLine = %q", got)
	}
	if got := groupLine(productdata.Group{Level: productdata.GroupMain, ID: "ELE"}); got != "S;ELE;;;" {
		t.Fatalf("main groupLine = %q", got)
	}
	if got := groupLine(productdata.Group{Level: productdata.GroupProduct, ID: "K"}); got != "S;K;;;" {
		t.Fatalf("parentless product groupLine = %q", got)
	}
	if got := discountKindCode(productdata.DiscountFactor); got != "2" {
		t.Fatalf("factor kind = %q", got)
	}
	if got := discountKindCode(productdata.DiscountSurcharge); got != "3" {
		t.Fatalf("surcharge kind = %q", got)
	}
	if got := discountKindCode(productdata.DiscountPercent); got != "1" {
		t.Fatalf("percent kind = %q", got)
	}
	if got := encodePriceValue(productdata.NewAmount(productdata.MustParseDecimal("1.234"), productdata.EUR)); got != "1.234" {
		t.Fatalf("encodePriceValue = %q", got)
	}
	if got := defaultString("", "fallback"); got != "fallback" {
		t.Fatalf("defaultString = %q", got)
	}
	if got := defaultString("value", "fallback"); got != "value" {
		t.Fatalf("defaultString = %q", got)
	}
	if got := extra(&productdata.Article{}, "x"); got != "" {
		t.Fatalf("extra = %q", got)
	}
}

func TestUniqueTextKey(t *testing.T) {
	t.Parallel()

	used := map[string]string{}
	first := uniqueTextKey("A-1", used)
	second := uniqueTextKey("A_1", used)
	if first == second {
		t.Fatalf("keys collide: %q", first)
	}
	third := uniqueTextKey("----", used)
	if third == "" || third == first || third == second {
		t.Fatalf("third key = %q", third)
	}
	long := uniqueTextKey("ABCDEFGHIJKLMNO", used)
	if len(long) > 18 {
		t.Fatalf("long key = %q", long)
	}
	if got := sanitizeKey("A-1/x"); got != "A_1_x" {
		t.Fatalf("sanitizeKey = %q", got)
	}
}

func TestValidateFieldCounts(t *testing.T) {
	t.Parallel()

	if err := validateFieldCounts([]string{"", "X;anything"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := validateFieldCounts([]string{"A;too;few"}); !errors.Is(err, ErrEncodingFailed) {
		t.Fatalf("error = %v", err)
	}
}

func TestWrapTextBranches(t *testing.T) {
	t.Parallel()

	lines, err := wrapText("\r\n\r\nfirst line\r\n\r\nlast line\r\n\r\n", 40)
	if err != nil {
		t.Fatalf("wrapText: %v", err)
	}
	if len(lines) != 3 || lines[0] != "first line" || lines[2] != "last line" {
		t.Fatalf("lines = %v", lines)
	}
	lines, err = wrapText(strings.Repeat("x", 45), 40)
	if err != nil {
		t.Fatalf("wrapText: %v", err)
	}
	if len(lines) != 2 || lines[0] != strings.Repeat("x", 40) {
		t.Fatalf("long word lines = %v", lines)
	}
	lines, err = wrapText("one two three", 5)
	if err != nil {
		t.Fatalf("wrapText: %v", err)
	}
	for _, line := range lines {
		if len(line) > 5 {
			t.Fatalf("line too long: %q", line)
		}
	}
}

func TestBestPriceFallbacks(t *testing.T) {
	t.Parallel()

	article := &productdata.Article{
		Prices: []productdata.Price{
			{Type: productdata.PriceUnknown},
			{Type: productdata.PriceGuide, Amount: productdata.NewAmount(productdata.MustParseDecimal("1"), productdata.EUR)},
		},
	}
	price, ok := bestPrice(article)
	if !ok || price.Type != productdata.PriceGuide {
		t.Fatalf("bestPrice = %+v, %v", price, ok)
	}
	if _, ok := bestPrice(&productdata.Article{Prices: []productdata.Price{{Type: productdata.PriceUnknown}}}); ok {
		t.Fatal("bestPrice accepted an unknown type")
	}
	if _, ok := bestPrice(&productdata.Article{}); ok {
		t.Fatal("bestPrice accepted an empty price list")
	}
}

func TestTextMarkerForVariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		article    *productdata.Article
		textKey    string
		wantMarker string
	}{
		{name: "none", article: &productdata.Article{}, wantMarker: "0"},
		{name: "long", article: &productdata.Article{}, textKey: "T1", wantMarker: "4"},
		{name: "dimension", article: &productdata.Article{Texts: []productdata.Text{productdata.NewText(productdata.TextDimension, "x")}}, wantMarker: "5"},
		{name: "long and dimension", article: &productdata.Article{Texts: []productdata.Text{productdata.NewText(productdata.TextDimension, "x")}}, textKey: "T1", wantMarker: "6"},
		{name: "stored consistent", article: &productdata.Article{Extra: map[string]string{"datanorm.text_marker": "4"}}, textKey: "T1", wantMarker: "4"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := textMarkerFor(tt.article, tt.textKey); got != tt.wantMarker {
				t.Fatalf("marker = %q, want %q", got, tt.wantMarker)
			}
		})
	}
}

func TestEncodingRoundTripUTF8(t *testing.T) {
	t.Parallel()

	catalog := sampleCatalog()
	var out strings.Builder
	if err := NewEncoder(WithCharset("utf-8")).Encode(context.Background(), &out, catalog, productdata.EncodeOptions{LineEnding: "\n"}); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !strings.Contains(out.String(), "Größe") {
		// sample catalog uses ASCII; add a rune to verify UTF-8 output path.
		catalog.Articles[0].Texts = []productdata.Text{productdata.NewText(productdata.TextShort, "Größe")}
		out.Reset()
		if err := NewEncoder(WithCharset("utf-8")).Encode(context.Background(), &out, catalog, productdata.EncodeOptions{LineEnding: "\n"}); err != nil {
			t.Fatalf("Encode: %v", err)
		}
		if !strings.Contains(out.String(), "Größe") {
			t.Fatalf("UTF-8 output lost the umlaut:\n%s", out.String())
		}
	}
	if _, err := charset.Encode("x", charset.CP850); err != nil {
		t.Fatalf("charset sanity check: %v", err)
	}
}

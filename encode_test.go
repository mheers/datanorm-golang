package datanorm

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mheers/datanorm-golang/charset"
	"github.com/mheers/datanorm-golang/productdata"
)

func sampleCatalog() *productdata.Catalog {
	return &productdata.Catalog{
		Header: productdata.Header{
			Format:        "datanorm",
			FormatVersion: "5.0",
			Description:   "Beispiel GmbH Preisliste",
			Currency:      productdata.EUR,
			Supplier:      productdata.Supplier{Name: "Beispiel GmbH"},
		},
		Articles: []productdata.Article{
			{
				SupplierArticleNumber:     "A-1000",
				ManufacturerArticleNumber: "M-4711",
				Type:                      "NYM-J 3x1.5",
				Matchcode:                 "NYM-J",
				GTIN:                      "4006381333931",
				Unit:                      productdata.NormalizeUnit("MTR"),
				Texts: []productdata.Text{
					productdata.NewText(productdata.TextShort, "Beispiel Kabel NYM-J", "3x1,5 mm²"),
					productdata.NewText(productdata.TextLong, "Dies ist ein langer Beispieltext, der über die vierzig Zeichen hinausgeht und umgebrochen werden muss.", "Zweite Zeile."),
					productdata.NewText(productdata.TextDimension, "Dimension eins", "Dimension zwei"),
				},
				Prices: []productdata.Price{
					{
						Type:   productdata.PriceList,
						Amount: productdata.AmountFromMinorUnits(129900, productdata.EUR),
						Per:    &productdata.Quantity{Value: productdata.NewDecimal(100, 0), Unit: productdata.NormalizeUnit("MTR")},
					},
					{
						Type:   productdata.PriceNet,
						Amount: productdata.AmountFromMinorUnits(99900, productdata.EUR),
					},
				},
				Groups: []productdata.GroupRef{
					{Level: productdata.GroupDiscount, ID: "04"},
					{Level: productdata.GroupMain, ID: "ELE"},
					{Level: productdata.GroupProduct, ID: "KABEL"},
				},
				MinOrderQuantity:  &productdata.Quantity{Value: productdata.NewDecimal(10, 0), Unit: productdata.NormalizeUnit("PCE")},
				PackagingQuantity: &productdata.Quantity{Value: productdata.NewDecimal(50, 0), Unit: productdata.NormalizeUnit("PCE")},
				Status:            productdata.StatusActive,
				Extra: map[string]string{
					"datanorm.catalog_page":     "42",
					"datanorm.cost_type":        "02",
					"datanorm.storage_flag":     "L",
					"datanorm.reference_number": "R-77",
				},
			},
		},
	}
}

func TestEncoderRoundTrip(t *testing.T) {
	t.Parallel()

	encoder := NewEncoder(WithCharset("cp850"))
	var out strings.Builder
	if err := encoder.Encode(context.Background(), &out, sampleCatalog(), productdata.EncodeOptions{}); err != nil {
		t.Fatalf("Encode returned error: %v", err)
	}
	if !strings.Contains(out.String(), "V;050;A;") {
		t.Fatalf("missing V;050;A header:\n%s", out.String())
	}
	decoded, err := charset.Decode([]byte(out.String()), charset.CP850)
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if strings.ContainsRune(decoded, '\uFFFD') {
		t.Fatal("encoded output contains a replacement character")
	}

	file, err := Parse(strings.NewReader(out.String()), WithCharset("cp850"))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	catalog, err := file.Catalog()
	if err != nil {
		t.Fatalf("Catalog returned error: %v", err)
	}
	if len(catalog.Articles) != 1 {
		t.Fatalf("articles = %d, want 1", len(catalog.Articles))
	}
	a := catalog.Articles[0]
	if a.SupplierArticleNumber != "A-1000" {
		t.Errorf("number = %q", a.SupplierArticleNumber)
	}
	if a.ShortText() != "Beispiel Kabel NYM-J 3x1,5 mm²" {
		t.Errorf("short text = %q", a.ShortText())
	}
	if !strings.HasPrefix(a.Text(productdata.TextLong), "Dies ist ein langer Beispieltext") {
		t.Errorf("long text = %q", a.Text(productdata.TextLong))
	}
	if a.Text(productdata.TextDimension) != "Dimension eins\nDimension zwei" {
		t.Errorf("dimension text = %q", a.Text(productdata.TextDimension))
	}
	if a.Unit.Code != "MTR" {
		t.Errorf("unit = %q", a.Unit.Code)
	}
	if a.GTIN != "4006381333931" {
		t.Errorf("gtin = %q", a.GTIN)
	}
	if a.ManufacturerArticleNumber != "M-4711" {
		t.Errorf("manufacturer number = %q", a.ManufacturerArticleNumber)
	}
	if a.Type != "NYM-J 3x1.5" {
		t.Errorf("type = %q", a.Type)
	}
	if price, ok := a.Price(productdata.PriceList); !ok {
		t.Fatal("missing list price")
	} else if price.Amount.String() != "1299.00 EUR" {
		t.Errorf("list price = %q", price.Amount.String())
	} else if price.Per == nil || price.Per.Value.String() != "100" {
		t.Errorf("price per = %+v", price.Per)
	}
	if group, ok := a.Group(productdata.GroupDiscount); !ok || group.ID != "04" {
		t.Errorf("discount group = %+v, %v", group, ok)
	}
	if a.MinOrderQuantity == nil || a.MinOrderQuantity.Value.String() != "10" {
		t.Errorf("min order = %+v", a.MinOrderQuantity)
	}
	if a.PackagingQuantity == nil || a.PackagingQuantity.Value.String() != "50" {
		t.Errorf("packaging = %+v", a.PackagingQuantity)
	}
	if got := a.Extra["datanorm.text_marker"]; got != "6" {
		t.Errorf("text marker = %q, want 6", got)
	}
	if got := a.Extra["datanorm.catalog_page"]; got != "42" {
		t.Errorf("catalog page = %q", got)
	}
	if got := a.Extra["datanorm.cost_type"]; got != "02" {
		t.Errorf("cost type = %q", got)
	}
}

func TestEncodePrices(t *testing.T) {
	t.Parallel()

	encoder := NewEncoder()
	var out strings.Builder
	if err := encoder.EncodePrices(context.Background(), &out, sampleCatalog(), productdata.EncodeOptions{}); err != nil {
		t.Fatalf("EncodePrices returned error: %v", err)
	}
	if !strings.Contains(out.String(), "P;A-1000;1;100;129900;04") {
		t.Fatalf("unexpected prices:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "P;A-1000;2;1;99900;04") {
		t.Fatalf("unexpected prices:\n%s", out.String())
	}

	file, err := Parse(strings.NewReader(out.String()))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	catalog, err := file.Catalog()
	if err != nil {
		t.Fatalf("Catalog returned error: %v", err)
	}
	if len(catalog.Articles) != 1 {
		t.Fatalf("articles = %d, want 1", len(catalog.Articles))
	}
	article := catalog.Articles[0]
	if price, ok := article.Price(productdata.PriceList); !ok || price.Amount.String() != "1299.00 EUR" {
		t.Fatalf("list price = %+v, %v", price, ok)
	}
	if price, ok := article.Price(productdata.PriceNet); !ok || price.Amount.String() != "999.00 EUR" {
		t.Fatalf("net price = %+v, %v", price, ok)
	}
}

func TestEncodeLoadRoundTrip(t *testing.T) {
	t.Parallel()

	catalog := sampleCatalog()
	catalog.Groups = []productdata.Group{
		{Level: productdata.GroupMain, ID: "ELE", Name: "Elektro"},
		{Level: productdata.GroupProduct, ID: "KABEL", Name: "Kabel", ParentID: "ELE"},
	}
	catalog.DiscountGroups = []productdata.DiscountGroup{
		{
			ID:   "04",
			Name: "Kabelrabatt",
			Discounts: []productdata.Discount{
				{Kind: productdata.DiscountPercent, Value: productdata.MustParseDecimal("20")},
			},
		},
	}

	dir := t.TempDir()
	encoder := NewEncoder()
	write := func(name string, encode func(*Encoder, io.Writer) error) string {
		path := filepath.Join(dir, name)
		file, err := os.Create(path)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		defer func() { _ = file.Close() }()
		if err := encode(encoder, file); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return path
	}
	ctx := context.Background()
	articlePath := write("DATANORM.001", func(e *Encoder, w io.Writer) error {
		return e.Encode(ctx, w, catalog, productdata.EncodeOptions{})
	})
	groupPath := write("DATANORM.WRG", func(e *Encoder, w io.Writer) error {
		return e.EncodeGroups(ctx, w, catalog, productdata.EncodeOptions{})
	})
	discountPath := write("DATANORM.RAB", func(e *Encoder, w io.Writer) error {
		return e.EncodeDiscounts(ctx, w, catalog, productdata.EncodeOptions{})
	})
	pricePath := write("DATPREIS.001", func(e *Encoder, w io.Writer) error {
		return e.EncodePrices(ctx, w, catalog, productdata.EncodeOptions{})
	})

	loaded, err := Load(ctx, []string{pricePath, articlePath, groupPath, discountPath})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(loaded.Articles) != 1 {
		t.Fatalf("articles = %d, want 1", len(loaded.Articles))
	}
	article := loaded.Articles[0]
	if price, ok := article.Price(productdata.PriceNet); !ok || price.Amount.String() != "999.00 EUR" {
		t.Fatalf("net price = %+v, %v", price, ok)
	}
	if group, ok := article.Group(productdata.GroupProduct); !ok || group.Name != "Kabel" {
		t.Fatalf("product group = %+v, %v", group, ok)
	}
	if group, ok := article.Group(productdata.GroupDiscount); !ok || group.Name != "Kabelrabatt" {
		t.Fatalf("discount group = %+v, %v", group, ok)
	}
	if !strings.Contains(article.Text(productdata.TextLong), "Beispieltext") {
		t.Fatalf("long text = %q", article.Text(productdata.TextLong))
	}
}

func TestEncodeGroupsAndDiscounts(t *testing.T) {
	t.Parallel()

	catalog := &productdata.Catalog{
		Header: productdata.Header{Currency: productdata.EUR, Supplier: productdata.Supplier{Name: "Beispiel"}},
		Groups: []productdata.Group{
			{Level: productdata.GroupMain, ID: "ELE", Name: "Elektro"},
			{Level: productdata.GroupProduct, ID: "KABEL", Name: "Kabel", ParentID: "ELE"},
		},
		DiscountGroups: []productdata.DiscountGroup{
			{ID: "04", Name: "Kabelrabatt", Discounts: []productdata.Discount{{Kind: productdata.DiscountPercent, Value: productdata.MustParseDecimal("20")}}},
		},
	}

	encoder := NewEncoder()
	var groups strings.Builder
	if err := encoder.EncodeGroups(context.Background(), &groups, catalog, productdata.EncodeOptions{}); err != nil {
		t.Fatalf("EncodeGroups returned error: %v", err)
	}
	if !strings.Contains(groups.String(), "S;ELE;;Elektro;") || !strings.Contains(groups.String(), "S;ELE;KABEL;Kabel;") {
		t.Fatalf("unexpected groups:\n%s", groups.String())
	}

	var discounts strings.Builder
	if err := encoder.EncodeDiscounts(context.Background(), &discounts, catalog, productdata.EncodeOptions{}); err != nil {
		t.Fatalf("EncodeDiscounts returned error: %v", err)
	}
	if !strings.Contains(discounts.String(), "R;04;1;20;Kabelrabatt;") {
		t.Fatalf("unexpected discounts:\n%s", discounts.String())
	}
}

func TestEncodePriceOnRequest(t *testing.T) {
	t.Parallel()

	catalog := &productdata.Catalog{
		Header: productdata.Header{Currency: productdata.EUR, Supplier: productdata.Supplier{Name: "Beispiel"}},
		Articles: []productdata.Article{
			{
				SupplierArticleNumber: "A-1",
				Unit:                  productdata.NormalizeUnit("PCE"),
				Texts:                 []productdata.Text{productdata.NewText(productdata.TextShort, "Kabel")},
				Prices: []productdata.Price{
					{Type: productdata.PriceOnRequest, Amount: productdata.NewAmount(productdata.Zero(), productdata.EUR)},
				},
			},
		},
	}

	var out strings.Builder
	if err := NewEncoder().Encode(context.Background(), &out, catalog, productdata.EncodeOptions{}); err != nil {
		t.Fatalf("Encode returned error: %v", err)
	}
	file, err := Parse(strings.NewReader(out.String()))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	articles := file.Articles()
	if len(articles) != 1 {
		t.Fatalf("articles = %d, want 1", len(articles))
	}
	if articles[0].PriceTypeRaw != "9" || articles[0].PriceRaw != "" {
		t.Fatalf("price fields = %q/%q, want 9/empty", articles[0].PriceTypeRaw, articles[0].PriceRaw)
	}
	parsed, err := file.Catalog()
	if err != nil {
		t.Fatalf("Catalog returned error: %v", err)
	}
	price, ok := parsed.Articles[0].Price(productdata.PriceOnRequest)
	if !ok || !price.Amount.IsZero() {
		t.Fatalf("price on request = %+v, %v", price, ok)
	}
}

func TestEncodeRejectsSemicolon(t *testing.T) {
	t.Parallel()

	catalog := sampleCatalog()
	catalog.Articles[0].Texts = []productdata.Text{
		productdata.NewText(productdata.TextShort, "Kabel; NYM-J"),
	}
	var out strings.Builder
	err := NewEncoder().Encode(context.Background(), &out, catalog, productdata.EncodeOptions{})
	if !errors.Is(err, ErrEncodingFailed) {
		t.Fatalf("error = %v, want ErrEncodingFailed", err)
	}
	if out.Len() != 0 {
		t.Fatalf("encoder wrote %d bytes before failing", out.Len())
	}
}

func TestWrapText(t *testing.T) {
	t.Parallel()

	lines, err := wrapText("Dies ist ein langer Beispieltext, der über die vierzig Zeichen hinausgeht.", 40)
	if err != nil {
		t.Fatalf("wrapText returned error: %v", err)
	}
	if len(lines) < 2 {
		t.Fatalf("lines = %v", lines)
	}
	for _, line := range lines {
		if len(line) > 40 {
			t.Fatalf("line too long: %q", line)
		}
	}
}

func TestWrapTextTooManyLines(t *testing.T) {
	t.Parallel()

	text := strings.Repeat("x", 40*100)
	if _, err := wrapText(text, 40); err == nil {
		t.Fatal("expected ErrTooManyTextLines")
	}
}

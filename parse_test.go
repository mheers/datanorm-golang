package datanorm

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mheers/datanorm-golang/charset"
	"github.com/mheers/datanorm-golang/productdata"
)

func join(fields ...string) string { return strings.Join(fields, ";") }

func aRecord(values map[int]string) string {
	fields := make([]string, 30)
	fields[0] = "A"
	for position, value := range values {
		if position >= 1 && position <= 30 {
			fields[position-1] = value
		}
	}
	return strings.Join(fields, ";")
}

func headerA(date string) string {
	return join("V", "050", "A", date, "EUR", "Beispiel GmbH", "Preisliste 2026", "", "", "", "", "", "", "", "", "")
}

func fullArticleRecord() string {
	return join(
		"A", "N", "A-1000", "Beispiel Kabel NYM-J", "3x1,5 mm²", "MTR",
		"1", "100", "129900", "04", "ELE", "KABEL", "NYM-J",
		"HER", "ALT-1000", "HER", "M-4711", "NYM-J 3x1.5", "4006381333931",
		"7", "10", "42", "6", "T-1000", "L", "REF", "R-77", "1", "02", "50",
	)
}

func TestParseMinimal(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		headerA("20260115"),
		aRecord(map[int]string{2: "N", 3: "A-1", 4: "Beispiel Kabel", 6: "PCE", 7: "1", 8: "1", 9: "100", 19: "4006381333931", 23: "0"}),
		"E;3;Beispiel GmbH;",
	}, "\r\n")

	file, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if file.Header == nil {
		t.Fatal("missing header")
	}
	if got, want := file.Header.Version, "050"; got != want {
		t.Fatalf("version = %q, want %q", got, want)
	}
	if got, want := file.Header.Currency, "EUR"; got != want {
		t.Fatalf("currency = %q, want %q", got, want)
	}
	if file.End == nil || file.End.Count != 3 {
		t.Fatalf("end record = %+v", file.End)
	}
	if got := len(file.Records); got != 3 {
		t.Fatalf("records = %d, want 3", got)
	}
	if got := len(file.Articles()); got != 1 {
		t.Fatalf("articles = %d, want 1", got)
	}
	if file.Diagnostics.HasErrors() {
		t.Fatalf("unexpected errors: %v", file.Diagnostics)
	}
}

func TestParseFullArticle(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		headerA("20260115"),
		fullArticleRecord(),
		"E;3;Beispiel GmbH;",
	}, "\n")

	file, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	articles := file.Articles()
	if len(articles) != 1 {
		t.Fatalf("articles = %d, want 1", len(articles))
	}
	a := articles[0]

	if got, want := a.ProcessingFlag, FlagNew; got != want {
		t.Errorf("ProcessingFlag = %q, want %q", got, want)
	}
	if got, want := a.Number, "A-1000"; got != want {
		t.Errorf("Number = %q, want %q", got, want)
	}
	if got, want := a.ShortText1, "Beispiel Kabel NYM-J"; got != want {
		t.Errorf("ShortText1 = %q, want %q", got, want)
	}
	if got, want := a.ShortText2, "3x1,5 mm²"; got != want {
		t.Errorf("ShortText2 = %q, want %q", got, want)
	}
	if got, want := a.Unit, "MTR"; got != want {
		t.Errorf("Unit = %q, want %q", got, want)
	}
	if got, want := a.DiscountGroup, "04"; got != want {
		t.Errorf("DiscountGroup = %q, want %q", got, want)
	}
	if got, want := a.MainGroup, "ELE"; got != want {
		t.Errorf("MainGroup = %q, want %q", got, want)
	}
	if got, want := a.ProductGroup, "KABEL"; got != want {
		t.Errorf("ProductGroup = %q, want %q", got, want)
	}
	if got, want := a.Matchcode, "NYM-J"; got != want {
		t.Errorf("Matchcode = %q, want %q", got, want)
	}
	if got, want := a.AlternativeArticleNumber, "ALT-1000"; got != want {
		t.Errorf("AlternativeArticleNumber = %q, want %q", got, want)
	}
	if got, want := a.ManufacturerNumber, "M-4711"; got != want {
		t.Errorf("ManufacturerNumber = %q, want %q", got, want)
	}
	if got, want := a.ManufacturerType, "NYM-J 3x1.5"; got != want {
		t.Errorf("ManufacturerType = %q, want %q", got, want)
	}
	if got, want := a.GTIN, "4006381333931"; got != want {
		t.Errorf("GTIN = %q, want %q", got, want)
	}
	if got, want := a.MediaReference, "7"; got != want {
		t.Errorf("MediaReference = %q, want %q", got, want)
	}
	if got, want := a.MinOrderQuantityRaw, "10"; got != want {
		t.Errorf("MinOrderQuantityRaw = %q, want %q", got, want)
	}
	if got, want := a.CatalogPage, "42"; got != want {
		t.Errorf("CatalogPage = %q, want %q", got, want)
	}
	if got, want := string(a.TextMarker), "6"; got != want {
		t.Errorf("TextMarker = %q, want %q", got, want)
	}
	if got, want := a.LongTextKey, "T-1000"; got != want {
		t.Errorf("LongTextKey = %q, want %q", got, want)
	}
	if got, want := a.StorageFlag, "L"; got != want {
		t.Errorf("StorageFlag = %q, want %q", got, want)
	}
	if got, want := a.ReferenceNumber, "R-77"; got != want {
		t.Errorf("ReferenceNumber = %q, want %q", got, want)
	}
	if got, want := a.VATMarker, "1"; got != want {
		t.Errorf("VATMarker = %q, want %q", got, want)
	}
	if got, want := a.CostType, "02"; got != want {
		t.Errorf("CostType = %q, want %q", got, want)
	}
	if got, want := a.PackagingQuantity, "50"; got != want {
		t.Errorf("PackagingQuantity = %q, want %q", got, want)
	}
	if got, want := len(a.Fields()), 30; got != want {
		t.Fatalf("fields = %d, want %d", got, want)
	}
	if got, want := a.Field(10), "04"; got != want {
		t.Fatalf("Field(10) = %q, want %q", got, want)
	}

	amount, err := a.Amount(productdata.EUR)
	if err != nil {
		t.Fatalf("Amount returned error: %v", err)
	}
	if got, want := amount.String(), "1299.00 EUR"; got != want {
		t.Fatalf("Amount = %q, want %q", got, want)
	}
}

func TestParsePriceRepresentations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		raw  string
		want string
	}{
		{raw: "129900", want: "1299.00 EUR"},
		{raw: "1299,00", want: "1299.00 EUR"},
		{raw: "1299.00", want: "1299.00 EUR"},
		{raw: "0", want: "0.00 EUR"},
		{raw: "5", want: "0.05 EUR"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.raw, func(t *testing.T) {
			t.Parallel()
			got, err := ParsePrice(tt.raw, productdata.EUR)
			if err != nil {
				t.Fatalf("ParsePrice returned error: %v", err)
			}
			if got.String() != tt.want {
				t.Fatalf("ParsePrice = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseTexts(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		headerA("20260115"),
		aRecord(map[int]string{2: "N", 3: "A-1", 4: "Kabel", 6: "PCE", 7: "1", 8: "1", 9: "100", 13: "Kabel", 23: "4", 24: "T-1"}),
		"T;N;T-1;1;02;zweite Zeile;",
		"T;N;T-1;1;01;erste Zeile;",
		"D;N;A-1;02;F;Dimension zwei;;",
		"D;N;A-1;01;F;Dimension eins;;",
		"E;6;Beispiel;",
	}, "\n")

	catalog, err := Decode(context.Background(), strings.NewReader(input), productdata.DecodeOptions{})
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if len(catalog.Articles) != 1 {
		t.Fatalf("articles = %d, want 1", len(catalog.Articles))
	}
	a := catalog.Articles[0]
	if got, want := a.Text(productdata.TextLong), "erste Zeile\nzweite Zeile"; got != want {
		t.Fatalf("long text = %q, want %q", got, want)
	}
	if got, want := a.Text(productdata.TextDimension), "Dimension eins\nDimension zwei"; got != want {
		t.Fatalf("dimension text = %q, want %q", got, want)
	}
	if got, want := a.ShortText(), "Kabel"; got != want {
		t.Fatalf("short text = %q, want %q", got, want)
	}
}

func TestDimensionTextInsertion(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		headerA("20260115"),
		aRecord(map[int]string{2: "N", 3: "A-1", 4: "Kabel", 6: "PCE", 7: "1", 8: "1", 9: "100", 13: "Kabel", 23: "5"}),
		"T;N;T-9;1;01;Baustein;",
		"D;N;A-1;01;F;Zeile eins;;",
		"D;N;A-1;02;T;;T-9;",
		"E;5;Beispiel;",
	}, "\n")

	catalog, err := Decode(context.Background(), strings.NewReader(input), productdata.DecodeOptions{})
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if got, want := catalog.Articles[0].Text(productdata.TextDimension), "Zeile eins\nBaustein"; got != want {
		t.Fatalf("dimension text = %q, want %q", got, want)
	}
}

func TestParseGroupsAndDiscounts(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		headerA("20260115"),
		aRecord(map[int]string{2: "N", 3: "A-1", 4: "Kabel", 6: "PCE", 7: "1", 8: "1", 9: "100", 10: "04", 11: "ELE", 12: "KABEL", 13: "Kabel", 23: "4", 24: "T-1"}),
		"S;ELE;;Elektro;",
		"S;ELE;KABEL;Kabel und Leitungen;",
		"R;04;1;20;Kabelrabatt;",
		"E;5;Beispiel;",
	}, "\n")

	catalog, err := Decode(context.Background(), strings.NewReader(input), productdata.DecodeOptions{})
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if len(catalog.Groups) != 2 {
		t.Fatalf("groups = %d, want 2", len(catalog.Groups))
	}
	if group, ok := catalog.FindGroup(productdata.GroupProduct, "KABEL"); !ok || group.Name != "Kabel und Leitungen" || group.ParentID != "ELE" {
		t.Fatalf("product group = %+v, %v", group, ok)
	}
	if discount, ok := catalog.FindDiscountGroup("04"); !ok || len(discount.Discounts) != 1 {
		t.Fatalf("discount = %+v, %v", discount, ok)
	} else if got, want := discount.Discounts[0].Value.String(), "20"; got != want {
		t.Fatalf("discount value = %q, want %q", got, want)
	}
	a := catalog.Articles[0]
	if group, ok := a.Group(productdata.GroupDiscount); !ok || group.ID != "04" {
		t.Fatalf("article discount group = %+v, %v", group, ok)
	}
	if group, ok := a.Group(productdata.GroupMain); !ok || group.Name != "Elektro" {
		t.Fatalf("article main group = %+v, %v", group, ok)
	}
}

func TestParseVersion4Rejected(t *testing.T) {
	t.Parallel()

	input := "V 170126Beispiel                                                         04EUR\r\n"
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrVersion4) {
		t.Fatalf("error = %v, want ErrVersion4", err)
	}
}

func TestParseUnsupportedVersion(t *testing.T) {
	t.Parallel()

	input := "V;040;A;20260115;EUR;Beispiel\n"
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("error = %v, want ErrUnsupportedVersion", err)
	}
}

func TestParseEndCountMismatch(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		headerA("20260115"),
		aRecord(map[int]string{2: "N", 3: "A-1", 4: "Kabel", 6: "PCE", 7: "1", 8: "1", 9: "100", 19: "4006381333931", 23: "0"}),
		"E;99;Beispiel;",
	}, "\n")
	file, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if !hasDiagnostic(file.Diagnostics, "dn.end-count") {
		t.Fatalf("missing dn.end-count: %v", file.Diagnostics)
	}
}

func TestParseBadPrice(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		headerA("20260115"),
		aRecord(map[int]string{2: "N", 3: "A-1", 4: "Kabel", 6: "PCE", 7: "1", 8: "1", 9: "12x3", 19: "4006381333931", 23: "0"}),
		"E;3;Beispiel;",
	}, "\n")
	catalog, err := Decode(context.Background(), strings.NewReader(input), productdata.DecodeOptions{})
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if !hasDiagnostic(catalog.Diagnostics, "dn.price") {
		t.Fatalf("missing dn.price: %v", catalog.Diagnostics)
	}
	if len(catalog.Articles[0].Prices) != 0 {
		t.Fatalf("prices = %d, want 0", len(catalog.Articles[0].Prices))
	}
}

func TestParseEANChecksum(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		headerA("20260115"),
		aRecord(map[int]string{2: "N", 3: "A-1", 4: "Kabel", 6: "PCE", 7: "1", 8: "1", 9: "100", 19: "4006381333932", 23: "0"}),
		"E;3;Beispiel;",
	}, "\n")
	catalog, err := Decode(context.Background(), strings.NewReader(input), productdata.DecodeOptions{})
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if !hasDiagnostic(catalog.Diagnostics, "dn.ean-checksum") {
		t.Fatalf("missing dn.ean-checksum: %v", catalog.Diagnostics)
	}
}

func TestParseCP850(t *testing.T) {
	t.Parallel()

	text := strings.Join([]string{
		headerA("20260115"),
		aRecord(map[int]string{2: "N", 3: "A-1", 4: "Zubehör für Größe", 6: "PCE", 7: "1", 8: "1", 9: "100", 19: "4006381333931", 23: "0"}),
		"E;3;Beispiel;",
	}, "\r\n")
	data, err := charset.Encode(text, charset.CP850)
	if err != nil {
		t.Fatalf("Encode returned error: %v", err)
	}
	file, err := Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if got, want := file.Articles()[0].ShortText1, "Zubehör für Größe"; got != want {
		t.Fatalf("ShortText1 = %q, want %q", got, want)
	}
	if got, want := file.Charset, charset.CP850; got != want {
		t.Fatalf("charset = %q, want %q", got, want)
	}
}

func TestPriceOnRequestWithoutAmount(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		headerA("20260115"),
		aRecord(map[int]string{2: "N", 3: "A-1", 4: "Kabel", 6: "PCE", 7: "9", 8: "1", 9: "", 23: "0"}),
		"E;3;Beispiel;",
	}, "\n")
	catalog, err := Decode(context.Background(), strings.NewReader(input), productdata.DecodeOptions{})
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if hasDiagnostic(catalog.Diagnostics, "dn.price") {
		t.Fatalf("unexpected dn.price: %v", catalog.Diagnostics)
	}
	price, ok := catalog.Articles[0].Price(productdata.PriceOnRequest)
	if !ok || !price.Amount.IsZero() {
		t.Fatalf("price on request = %+v, %v", price, ok)
	}
}

func TestParseDuplicateArticle(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		headerA("20260115"),
		aRecord(map[int]string{2: "N", 3: "A-1", 4: "Kabel", 6: "PCE", 7: "1", 8: "1", 9: "100", 19: "4006381333931", 23: "0"}),
		aRecord(map[int]string{2: "N", 3: "A-1", 4: "Kabel 2", 6: "PCE", 7: "1", 8: "1", 9: "100", 19: "4006381333931", 23: "0"}),
		"E;4;Beispiel;",
	}, "\n")
	file, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if !hasDiagnostic(file.Diagnostics, "dn.duplicate-article") {
		t.Fatalf("missing dn.duplicate-article: %v", file.Diagnostics)
	}
}

func TestScan(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		headerA("20260115"),
		aRecord(map[int]string{2: "N", 3: "A-1", 4: "Kabel", 6: "PCE", 7: "1", 8: "1", 9: "100", 19: "4006381333931", 23: "0"}),
		"E;3;Beispiel;",
	}, "\n")
	var count int
	if err := Scan(strings.NewReader(input), func(record Record) error {
		count++
		return nil
	}); err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if count != 3 {
		t.Fatalf("count = %d, want 3", count)
	}
}

func TestArticleFieldSpecs(t *testing.T) {
	t.Parallel()

	specs := FieldSpecs(KindArticle)
	if len(specs) != 30 {
		t.Fatalf("specs = %d, want 30", len(specs))
	}
	if got, want := specs[9].Name, "DiscountGroup"; got != want {
		t.Fatalf("field 10 = %q, want %q", got, want)
	}
	if got, want := specs[22].Name, "TextMarker"; got != want {
		t.Fatalf("field 23 = %q, want %q", got, want)
	}
	if got, want := specs[23].Name, "LongTextKey"; got != want {
		t.Fatalf("field 24 = %q, want %q", got, want)
	}
}

func hasDiagnostic(ds productdata.Diagnostics, code string) bool {
	for _, d := range ds {
		if d.Code == code {
			return true
		}
	}
	return false
}

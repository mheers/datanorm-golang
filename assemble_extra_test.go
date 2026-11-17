package datanorm

import (
	"context"
	"strings"
	"testing"

	"github.com/mheers/datanorm-golang/productdata"
)

func TestAssembleMissingHeader(t *testing.T) {
	t.Parallel()

	if _, err := (&File{}).Catalog(); err != ErrMissingHeader {
		t.Fatalf("error = %v, want ErrMissingHeader", err)
	}
}

func TestAssembleTextAndDimensionEdgeCases(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		headerA("20260115"),
		aRecord(map[int]string{2: "N", 3: "A-1", 4: "Kabel", 23: "6", 24: "T-1"}),
		"T;N;T-1;1;02;zweite;",
		"T;N;T-1;1;01;erste;",
		"T;N;T-1;1;01;doppelt;",
		"T;N;T-2;1;01;geloescht;",
		"T;L;T-2;1;01;;",
		"D;N;A-1;01;F;Dim eins;;",
		"D;N;A-1;01;F;Dim doppelt;;",
		"D;N;A-1;02;T;;FEHLT;",
		"D;N;A-1;03;T;;;",
		"D;N;A-1;04;F;Dim vier;;",
		"D;L;A-1;05;F;;;",
		"D;N;;01;F;Ohne Artikel;;",
		"E;15;Beispiel;",
	}, "\n")

	catalog, err := Decode(context.Background(), strings.NewReader(input), productdata.DecodeOptions{})
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	article := catalog.Articles[0]
	if got, want := article.Text(productdata.TextLong), "erste\nzweite"; got != want {
		t.Fatalf("long text = %q, want %q", got, want)
	}
	if got, want := article.Text(productdata.TextDimension), "Dim eins\nDim vier"; got != want {
		t.Fatalf("dimension text = %q, want %q", got, want)
	}
	for _, code := range []string{"dn.text-line-duplicate", "dn.missing-text", "dn.dimension-article"} {
		if !hasDiagnostic(catalog.Diagnostics, code) {
			t.Errorf("missing %s in %v", code, catalog.Diagnostics)
		}
	}
}

func TestAssembleMedia(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		headerA("20260115"),
		aRecord(map[int]string{2: "N", 3: "A-1", 4: "Kabel"}),
		"G;N;A-1;7;1;Bild;BILD.GIF;",
		"G;A-1;8;2;CAD;PLAN.DXF;",
		"G;N;A-1;9;3;Video;CLIP.MP4;",
		"G;N;A-1;10;4;Audio;TON.MP3;",
		"G;N;A-1;11;5;Doku;DOK.PDF;",
		"G;N;A-1;12;X;Sonstig;DATEI.BIN;",
		"G;N;;13;1;Ohne Artikel;X.GIF;",
		"E;9;Beispiel;",
	}, "\n")
	catalog, err := Decode(context.Background(), strings.NewReader(input), productdata.DecodeOptions{})
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	media := catalog.Articles[0].Media
	if len(media) != 6 {
		t.Fatalf("media = %d, want 6: %+v", len(media), media)
	}
	wantTypes := []productdata.MediaType{
		productdata.MediaImage, productdata.MediaDrawing, productdata.MediaVideo,
		productdata.MediaAudio, productdata.MediaDocument, productdata.MediaOther,
	}
	for i, want := range wantTypes {
		if media[i].Type != want {
			t.Errorf("media[%d].Type = %q, want %q", i, media[i].Type, want)
		}
	}
}

func TestMediaTypeHelper(t *testing.T) {
	t.Parallel()

	tests := map[string]productdata.MediaType{
		"1":     productdata.MediaImage,
		"B":     productdata.MediaImage,
		"JPG":   productdata.MediaImage,
		"2":     productdata.MediaDrawing,
		"CAD":   productdata.MediaDrawing,
		"3":     productdata.MediaVideo,
		"MP4":   productdata.MediaVideo,
		"4":     productdata.MediaAudio,
		"MP3":   productdata.MediaAudio,
		"5":     productdata.MediaDocument,
		"PDF":   productdata.MediaDocument,
		"video": productdata.MediaVideo,
		"":      productdata.MediaOther,
		"XYZ":   productdata.MediaOther,
	}
	for raw, want := range tests {
		if got := mediaType(raw); got != want {
			t.Errorf("mediaType(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestAssembleDiscounts(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		headerA("20260115"),
		"R;04;1;20;Prozent;",
		"R;04;2;0.8;Faktor;",
		"R;05;3;5;Zuschlag;",
		"R;06;1;;Leer;",
		"R;;1;10;Ohne Gruppe;",
		"R;07;1;abc;Ungueltig;",
		"E;7;Beispiel;",
	}, "\n")
	catalog, err := Decode(context.Background(), strings.NewReader(input), productdata.DecodeOptions{})
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if len(catalog.DiscountGroups) != 4 {
		t.Fatalf("discount groups = %d, want 4: %+v", len(catalog.DiscountGroups), catalog.DiscountGroups)
	}
	group, ok := catalog.FindDiscountGroup("04")
	if !ok || len(group.Discounts) != 2 {
		t.Fatalf("group 04 = %+v, %v", group, ok)
	}
	if group.Discounts[0].Kind != productdata.DiscountPercent || group.Discounts[1].Kind != productdata.DiscountFactor {
		t.Fatalf("kinds = %+v", group.Discounts)
	}
	empty, ok := catalog.FindDiscountGroup("06")
	if !ok || len(empty.Discounts) != 1 || !empty.Discounts[0].Value.IsZero() {
		t.Fatalf("group 06 = %+v, %v", empty, ok)
	}
	invalid, ok := catalog.FindDiscountGroup("07")
	if !ok || len(invalid.Discounts) != 0 {
		t.Fatalf("group 07 = %+v, %v", invalid, ok)
	}
	if !hasDiagnostic(catalog.Diagnostics, "dn.discount-value") {
		t.Fatalf("missing dn.discount-value: %v", catalog.Diagnostics)
	}
}

func TestAssembleArticleEdgeCases(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		headerA("20260115"),
		aRecord(map[int]string{
			2: "N", 3: "A-9", 4: "Kabel", 6: "MTR", 7: "9", 8: "1", 9: "",
			10: "04", 11: "ELE", 12: "KABEL", 21: "10", 22: "42", 25: "L",
			29: "02", 30: "50",
		}),
		aRecord(map[int]string{2: "A", 3: "A-2", 4: "Dose", 7: "1", 8: "x", 9: "100"}),
		aRecord(map[int]string{2: "L", 3: "A-3", 4: "Alt"}),
		aRecord(map[int]string{2: "X", 3: "A-4", 4: "Neu"}),
		aRecord(map[int]string{2: "N", 3: "A-1", 4: "Ersetzt"}),
		aRecord(map[int]string{2: "N", 3: "A-5", 4: "BadPrice", 9: "12x"}),
		"E;7;Beispiel;",
	}, "\n")
	catalog, err := Decode(context.Background(), strings.NewReader(input), productdata.DecodeOptions{})
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if len(catalog.Articles) != 6 {
		t.Fatalf("articles = %d, want 6", len(catalog.Articles))
	}
	replaced, ok := catalog.FindArticle("A-1")
	if !ok || replaced.ShortText() != "Ersetzt" {
		t.Fatalf("A-1 = %+v, %v", replaced, ok)
	}
	first, ok := catalog.FindArticle("A-9")
	if !ok {
		t.Fatal("missing A-9")
	}
	if price, ok := first.Price(productdata.PriceOnRequest); !ok || !price.Amount.IsZero() {
		t.Fatalf("on-request price = %+v, %v", price, ok)
	}
	if first.MinOrderQuantity == nil || first.MinOrderQuantity.Value.String() != "10" {
		t.Fatalf("min order = %+v", first.MinOrderQuantity)
	}
	if first.PackagingQuantity == nil || first.PackagingQuantity.Value.String() != "50" {
		t.Fatalf("packaging = %+v", first.PackagingQuantity)
	}
	if first.Logistics.StorageFlag != "L" {
		t.Fatalf("storage flag = %q", first.Logistics.StorageFlag)
	}
	if got := first.Extra["datanorm.catalog_page"]; got != "42" {
		t.Fatalf("catalog page = %q", got)
	}
	if got := first.Extra["datanorm.cost_type"]; got != "02" {
		t.Fatalf("cost type = %q", got)
	}
	if article, _ := catalog.FindArticle("A-2"); article.Status != productdata.StatusChanged {
		t.Fatalf("A-2 status = %q", article.Status)
	}
	if article, _ := catalog.FindArticle("A-3"); article.Status != productdata.StatusDeleted {
		t.Fatalf("A-3 status = %q", article.Status)
	}
	if article, _ := catalog.FindArticle("A-4"); article.Status != productdata.StatusChanged {
		t.Fatalf("A-4 status = %q", article.Status)
	}
	if !hasDiagnostic(catalog.Diagnostics, "dn.price-unit") {
		t.Fatalf("missing dn.price-unit: %v", catalog.Diagnostics)
	}
	if !hasDiagnostic(catalog.Diagnostics, "dn.price") {
		t.Fatalf("missing dn.price: %v", catalog.Diagnostics)
	}
}

func TestAssembleChanges(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		headerA("20260115"),
		aRecord(map[int]string{2: "N", 3: "A-1", 4: "Kabel"}),
		aRecord(map[int]string{2: "N", 3: "A-2", 4: "Dose"}),
		aRecord(map[int]string{2: "N", 3: "A-3", 4: "Alt"}),
		"B;A;A-3;A-9;;",
		"B;L;A-4;;;",
		"B;X;A-5;A-6;;",
		"B;X;A-2;A-1;;",
		"B;X;;A-7;;",
		"E;8;Beispiel;",
	}, "\n")
	catalog, err := Decode(context.Background(), strings.NewReader(input), productdata.DecodeOptions{})
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if article, ok := catalog.FindArticle("A-3"); !ok || article.Status != productdata.StatusDiscontinued {
		t.Fatalf("A-3 = %+v, %v", article, ok)
	}
	if article, ok := catalog.FindArticle("A-3"); !ok || article.Replacement == nil || article.Replacement.SupplierArticleNumber != "A-9" {
		t.Fatalf("replacement = %+v", article.Replacement)
	}
	if article, ok := catalog.FindArticle("A-4"); !ok || article.Status != productdata.StatusDeleted {
		t.Fatalf("A-4 = %+v, %v", article, ok)
	}
	if article, ok := catalog.FindArticle("A-6"); !ok || article.Status != productdata.StatusChanged {
		t.Fatalf("A-6 = %+v, %v", article, ok)
	}
	if article, ok := catalog.FindArticle("A-7"); !ok || article.Status != productdata.StatusChanged {
		t.Fatalf("A-7 = %+v, %v", article, ok)
	}
	if article, ok := catalog.FindArticle("A-2"); !ok || article.Status != productdata.StatusDeleted {
		t.Fatalf("A-2 = %+v, %v", article, ok)
	}
	if !hasDiagnostic(catalog.Diagnostics, "dn.renumber-collision") {
		t.Fatalf("missing dn.renumber-collision: %v", catalog.Diagnostics)
	}
}

func TestAssemblePriceRecords(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		headerA("20260115"),
		aRecord(map[int]string{2: "N", 3: "A-1", 4: "Kabel", 10: "04"}),
		"P;A-1;2;10;99900;05;",
		"P;A-1;1;1;99900;",
		"P;A-1;9;1;;",
		"P;A-1;X;1;100;",
		"P;A-1;2;10;bad;",
		"P;NEW;1;100;500;",
		"E;7;Beispiel;",
	}, "\n")
	catalog, err := Decode(context.Background(), strings.NewReader(input), productdata.DecodeOptions{})
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	article, ok := catalog.FindArticle("A-1")
	if !ok {
		t.Fatal("missing A-1")
	}
	if price, ok := article.Price(productdata.PriceNet); !ok {
		t.Fatal("missing net price")
	} else if price.Per == nil || price.Per.Value.String() != "10" {
		t.Fatalf("net price per = %+v", price.Per)
	}
	if price, ok := article.Price(productdata.PriceOnRequest); !ok || !price.Amount.IsZero() {
		t.Fatalf("on-request = %+v, %v", price, ok)
	}
	if price, ok := article.Price(productdata.PriceUnknown); !ok || price.Amount.String() != "1.00 EUR" {
		t.Fatalf("unknown price type = %+v, %v", price, ok)
	}
	for _, code := range []string{"dn.price", "dn.price-type"} {
		if !hasDiagnostic(catalog.Diagnostics, code) {
			t.Errorf("missing %s: %v", code, catalog.Diagnostics)
		}
	}
	newArticle, ok := catalog.FindArticle("NEW")
	if !ok || len(newArticle.Prices) != 1 || newArticle.Prices[0].Amount.String() != "5.00 EUR" {
		t.Fatalf("NEW = %+v, %v", newArticle, ok)
	}
}

func TestAssembleHeaderFallbacks(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		join("V", "050", "A", "not-a-date", "EURO", "Beispiel GmbH", "", "", "", "", "", "", "", "", "", ""),
		"E;2;Beispiel;",
	}, "\n")
	file, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	catalog, err := file.Catalog()
	if err != nil {
		t.Fatalf("Catalog returned error: %v", err)
	}
	if catalog.Header.CreatedAt != nil || catalog.Header.PriceDate != nil {
		t.Fatalf("unexpected dates: %+v", catalog.Header)
	}
	if catalog.Header.Currency != productdata.EUR {
		t.Fatalf("currency = %q", catalog.Header.Currency)
	}
	if !hasDiagnostic(catalog.Diagnostics, "dn.header-date") {
		t.Fatalf("missing dn.header-date: %v", catalog.Diagnostics)
	}
}

func TestArticleStatusAndDiscountKindHelpers(t *testing.T) {
	t.Parallel()

	statuses := map[ProcessingFlag]productdata.ArticleStatus{
		FlagNew:      productdata.StatusActive,
		FlagChange:   productdata.StatusChanged,
		FlagDelete:   productdata.StatusDeleted,
		FlagRenumber: productdata.StatusChanged,
		FlagUnknown:  productdata.StatusActive,
	}
	for flag, want := range statuses {
		if got := articleStatus(flag); got != want {
			t.Errorf("articleStatus(%q) = %q, want %q", flag, got, want)
		}
	}
	kinds := map[string]productdata.DiscountKind{
		"2": productdata.DiscountFactor,
		"3": productdata.DiscountSurcharge,
		"1": productdata.DiscountPercent,
		"":  productdata.DiscountPercent,
	}
	for raw, want := range kinds {
		if got := discountKind(raw); got != want {
			t.Errorf("discountKind(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestTextHelpers(t *testing.T) {
	t.Parallel()

	if got := joinTextLines(nil); got != "" {
		t.Fatalf("joinTextLines(nil) = %q", got)
	}
	lines := []textLine{{number: 1, text: "  "}, {number: 2, text: "a"}, {number: 3, text: "b  "}, {number: 4, text: " "}}
	if got := joinTextLines(lines); got != "a\nb" {
		t.Fatalf("joinTextLines = %q", got)
	}
	if got := blockLines(map[string][]textLine{"k": nil}, "k"); got != nil {
		t.Fatalf("blockLines = %v", got)
	}
	if got := textLines("", " ", "x"); len(got) != 1 || got[0] != "x" {
		t.Fatalf("textLines = %v", got)
	}
	extra := map[string]string{}
	putExtra(extra, "a", " ")
	putExtra(extra, "b", "x")
	if len(extra) != 1 || extra["b"] != "x" {
		t.Fatalf("extra = %v", extra)
	}
	if got := ensureExtra(nil); got == nil {
		t.Fatal("ensureExtra(nil) = nil")
	}
	existing := map[string]string{"a": "b"}
	if got := ensureExtra(existing); len(got) != 1 {
		t.Fatalf("ensureExtra(existing) = %v", got)
	}
}

func TestSortAndDedupeEmpty(t *testing.T) {
	t.Parallel()

	var diagnostics productdata.Diagnostics
	if got := sortAndDedupeTextLines(nil, "T", "k", &diagnostics); len(got) != 0 {
		t.Fatalf("result = %v", got)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %v", diagnostics)
	}
}

func TestGroupProductWithoutParent(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		headerA("20260115"),
		"S;;KABEL;Kabel;",
		"E;2;Beispiel;",
	}, "\n")
	catalog, err := Decode(context.Background(), strings.NewReader(input), productdata.DecodeOptions{})
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	// The S record has no main group, so it is ignored.
	if len(catalog.Groups) != 0 {
		t.Fatalf("groups = %+v", catalog.Groups)
	}
}

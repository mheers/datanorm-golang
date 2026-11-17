package datanorm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/mheers/datanorm-golang/productdata"
)

func TestKindMethods(t *testing.T) {
	t.Parallel()

	kinds := []Kind{
		KindHeader, KindArticle, KindChange, KindText, KindDimension, KindEnd,
		KindPrice, KindGroup, KindDiscount, KindSurcharge, KindSet, KindService,
		KindMedia, KindControl,
	}
	wantNames := []string{
		"header", "article", "change", "long text", "dimension text", "end",
		"price change", "product group", "discount group", "surcharge/tier",
		"set", "service", "media reference", "control",
	}
	for i, kind := range kinds {
		if !kind.Known() {
			t.Errorf("%q should be known", kind)
		}
		if got := kind.Name(); got != wantNames[i] {
			t.Errorf("Kind(%q).Name() = %q, want %q", kind, got, wantNames[i])
		}
		if got := kind.String(); got != string(byte(kind)) {
			t.Errorf("Kind(%q).String() = %q", kind, got)
		}
	}
	if KindUnknown.Known() {
		t.Fatal("unknown kind should not be known")
	}
	if got := KindUnknown.String(); got != "?" {
		t.Fatalf("KindUnknown.String() = %q", got)
	}
	if got := KindUnknown.Name(); got != "unknown" {
		t.Fatalf("KindUnknown.Name() = %q", got)
	}
}

func TestParseProcessingFlag(t *testing.T) {
	t.Parallel()

	tests := map[string]ProcessingFlag{
		"N": FlagNew,
		"A": FlagChange,
		"L": FlagDelete,
		"X": FlagRenumber,
		"P": FlagPriceRequest,
		"n": FlagNew,
		"":  FlagUnknown,
		"Z": FlagUnknown,
	}
	for raw, want := range tests {
		if got := ParseProcessingFlag(raw); got != want {
			t.Errorf("ParseProcessingFlag(%q) = %q, want %q", raw, got, want)
		}
	}
	if FlagNew.Valid() != true || FlagUnknown.Valid() != false {
		t.Fatal("Valid mismatch")
	}
	if got := FlagDelete.String(); got != "L" {
		t.Fatalf("String = %q", got)
	}
}

func TestFieldSpecsAllKinds(t *testing.T) {
	t.Parallel()

	kinds := []Kind{
		KindHeader, KindArticle, KindChange, KindText, KindDimension, KindEnd,
		KindPrice, KindGroup, KindDiscount, KindSurcharge, KindMedia, KindControl,
	}
	for _, kind := range kinds {
		specs := FieldSpecs(kind)
		if len(specs) == 0 {
			t.Errorf("FieldSpecs(%q) is empty", kind)
			continue
		}
		for i, spec := range specs {
			if spec.Position != i+1 {
				t.Errorf("%q spec %d has position %d", kind, i+1, spec.Position)
			}
			if spec.Name == "" {
				t.Errorf("%q spec %d has no name", kind, i+1)
			}
		}
		specs[0].Name = "mutated"
		if FieldSpecs(kind)[0].Name == "mutated" {
			t.Errorf("%q FieldSpecs returned an aliased slice", kind)
		}
	}
	for _, kind := range []Kind{KindSet, KindService, KindUnknown} {
		if specs := FieldSpecs(kind); specs != nil {
			t.Errorf("FieldSpecs(%q) = %v, want nil", kind, specs)
		}
	}
}

func TestParseErrorFormatting(t *testing.T) {
	t.Parallel()

	tests := []ParseError{
		{},
		{Line: 7},
		{File: "a.001", Line: 7},
		{Line: 7, RecordType: 'A'},
		{Line: 7, RecordType: 'A', Field: 3},
		{Line: 7, RecordType: 'A', Field: 3, Value: "x"},
		{Err: ErrVersion4},
	}
	for _, parseErr := range tests {
		parseErr := parseErr
		text := parseErr.Error()
		if text == "" {
			t.Fatalf("empty error string for %+v", parseErr)
		}
	}
	inner := errors.New("inner")
	parseErr := &ParseError{File: "f", Line: 2, RecordType: 'T', Field: 6, Value: "v", Err: inner}
	if !errors.Is(parseErr, inner) {
		t.Fatal("Unwrap did not expose the inner error")
	}
	if got := parseErr.Error(); !strings.Contains(got, "inner") || !strings.Contains(got, "f:2") {
		t.Fatalf("Error() = %q", got)
	}
}

func TestHeaderDataTypeAndSupplier(t *testing.T) {
	t.Parallel()

	markers := map[string]string{
		"A": "article data",
		"S": "product groups",
		"R": "discount groups",
		"P": "price changes",
		"J": "sets",
		"T": "texts",
		"":  "unknown",
		"X": "unknown",
	}
	for marker, want := range markers {
		header := &HeaderRecord{DataMarker: marker}
		if got := header.DataType(); got != want {
			t.Errorf("DataType(%q) = %q, want %q", marker, got, want)
		}
	}

	tests := []struct {
		header HeaderRecord
		want   string
	}{
		{HeaderRecord{Description: "Beispiel GmbH"}, "Beispiel GmbH"},
		{HeaderRecord{Creator: "Beispiel AG"}, "Beispiel AG"},
		{HeaderRecord{DataContent: "Beispiel KG"}, "Beispiel KG"},
		{HeaderRecord{Copyright: "Beispiel SE"}, "Beispiel SE"},
		{HeaderRecord{Description: "Preisliste"}, ""},
	}
	for _, tt := range tests {
		if got := tt.header.SupplierName(); got != tt.want {
			t.Errorf("SupplierName(%+v) = %q, want %q", tt.header, got, tt.want)
		}
	}
}

func TestArticleRecordHelpers(t *testing.T) {
	t.Parallel()

	record := parseSingleRecord(t, aRecord(map[int]string{
		2: "N", 3: "A-1", 6: "MTR", 7: "2", 8: "100", 9: "129900",
	}))
	article, ok := record.(*ArticleRecord)
	if !ok {
		t.Fatalf("record = %T", record)
	}
	if got := article.PriceType(); got != "2" {
		t.Fatalf("PriceType = %q, want 2", got)
	}
	if per, ok := article.PricePer(); !ok || per != 100 {
		t.Fatalf("PricePer = %d, %v", per, ok)
	}
	amount, err := article.Amount(productdata.EUR)
	if err != nil {
		t.Fatalf("Amount returned error: %v", err)
	}
	if got := amount.String(); got != "1299.00 EUR" {
		t.Fatalf("Amount = %q", got)
	}

	bad := parseSingleRecord(t, aRecord(map[int]string{2: "N", 3: "A-1", 8: "x"}))
	badArticle := bad.(*ArticleRecord)
	if _, ok := badArticle.PricePer(); ok {
		t.Fatal("PricePer accepted an invalid unit")
	}
	zero := parseSingleRecord(t, aRecord(map[int]string{2: "N", 3: "A-1", 8: "0"}))
	if _, ok := zero.(*ArticleRecord).PricePer(); ok {
		t.Fatal("PricePer accepted zero")
	}
	empty := parseSingleRecord(t, aRecord(map[int]string{2: "N", 3: "A-1"}))
	if _, ok := empty.(*ArticleRecord).PricePer(); ok {
		t.Fatal("PricePer accepted an empty unit")
	}
}

func TestPriceRecordAmount(t *testing.T) {
	t.Parallel()

	record := parseSingleRecord(t, "P;A-1;2;10;129900;04;")
	price, ok := record.(*PriceRecord)
	if !ok {
		t.Fatalf("record = %T", record)
	}
	amount, err := price.Amount(productdata.EUR)
	if err != nil {
		t.Fatalf("Amount returned error: %v", err)
	}
	if got := amount.String(); got != "1299.00 EUR" {
		t.Fatalf("Amount = %q", got)
	}
}

func TestRecordAccessors(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		headerA("20260115"),
		aRecord(map[int]string{2: "N", 3: "A-1", 4: "Kabel"}),
		"E;3;Beispiel;",
	}, "\n")
	file, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	counts := file.KindCount()
	if counts[KindHeader] != 1 || counts[KindArticle] != 1 || counts[KindEnd] != 1 {
		t.Fatalf("KindCount = %v", counts)
	}
	if got := len(file.RecordsOfKind(KindArticle)); got != 1 {
		t.Fatalf("RecordsOfKind = %d", got)
	}
	record := file.Records[1]
	if record.Kind() != KindArticle || record.Line() != 2 || !strings.Contains(record.Raw(), "A;N;A-1") {
		t.Fatalf("record = %+v", record)
	}
	if got := record.Field(1); got != "A" {
		t.Fatalf("Field(1) = %q", got)
	}
	if got := record.Field(0); got != "" {
		t.Fatalf("Field(0) = %q", got)
	}
	if got := record.Field(99); got != "" {
		t.Fatalf("Field(99) = %q", got)
	}
	if got := record.Fields()[2]; got != "A-1" {
		t.Fatalf("Fields()[2] = %q", got)
	}
}

func TestUnknownRecordKind(t *testing.T) {
	t.Parallel()

	record := parseSingleRecord(t, "Q;value;")
	unknown, ok := record.(*UnknownRecord)
	if !ok {
		t.Fatalf("record = %T", record)
	}
	if unknown.Kind() != Kind('Q') {
		t.Fatalf("Kind = %q", unknown.Kind())
	}
}

func TestFormatInterface(t *testing.T) {
	format, ok := productdata.Get("datanorm")
	if !ok {
		t.Fatal("datanorm format not registered")
	}
	if format.Name() != "datanorm" || format.Description() == "" {
		t.Fatalf("format = %q/%q", format.Name(), format.Description())
	}
	if len(format.Extensions()) == 0 {
		t.Fatal("no extensions")
	}
	caps := format.Capabilities()
	if !caps.Read || !caps.Write || !caps.Streaming || !caps.Batch || !caps.Prices || !caps.Texts || !caps.Media {
		t.Fatalf("capabilities = %+v", caps)
	}
	decoder, err := format.NewDecoder()
	if err != nil {
		t.Fatalf("NewDecoder: %v", err)
	}
	encoder, err := format.NewEncoder()
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	if decoder == nil || encoder == nil {
		t.Fatal("nil decoder or encoder")
	}
}

func TestDecodeGeneric(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		headerA("20260115"),
		aRecord(map[int]string{2: "N", 3: "A-1", 4: "Kabel"}),
		"E;3;Beispiel;",
	}, "\n")
	catalog, err := Decode(context.Background(), strings.NewReader(input), productdata.DecodeOptions{})
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if len(catalog.Articles) != 1 {
		t.Fatalf("articles = %d", len(catalog.Articles))
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Decode(ctx, strings.NewReader(input), productdata.DecodeOptions{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}

	if _, err := Decode(context.Background(), strings.NewReader("dies ist keine datanorm-datei\n"), productdata.DecodeOptions{}); err == nil {
		t.Fatal("expected an error for non-DATANORM input")
	}

	strictInput := strings.Join([]string{
		headerA("20260115"),
		"Q;x;",
		"E;3;Beispiel;",
	}, "\n")
	if _, err := Decode(context.Background(), strings.NewReader(strictInput), productdata.DecodeOptions{Strict: true}); err == nil {
		t.Fatal("strict mode accepted an unknown record")
	}
	if _, err := Decode(context.Background(), strings.NewReader(strictInput), productdata.DecodeOptions{Charset: "klingon"}); err == nil {
		t.Fatal("accepted an unknown charset")
	}
}

func TestOpenMissingFile(t *testing.T) {
	t.Parallel()

	if _, err := Open("/does/not/exist/DATANORM.001"); err == nil {
		t.Fatal("Open accepted a missing file")
	}
}

func TestOptions(t *testing.T) {
	t.Parallel()

	o := defaultOptions()
	WithStrict(true)(&o)
	if !o.strict {
		t.Fatal("WithStrict did not apply")
	}
	WithMaxLineBytes(0)(&o)
	if o.maxLineBytes == 0 {
		t.Fatal("WithMaxLineBytes(0) should be ignored")
	}
	WithMaxLineBytes(128)(&o)
	if o.maxLineBytes != 128 {
		t.Fatalf("maxLineBytes = %d", o.maxLineBytes)
	}
	WithoutValidation()(&o)
	if o.validate {
		t.Fatal("WithoutValidation did not apply")
	}
	WithLineEnding("\r")(&o)
	if o.lineEnding != "\r\n" {
		t.Fatalf("invalid line ending changed the value to %q", o.lineEnding)
	}
	WithLineEnding("\n")(&o)
	if o.lineEnding != "\n" {
		t.Fatalf("lineEnding = %q", o.lineEnding)
	}
	WithCharset("cp850")(&o)
	cs, err := o.resolveCharset()
	if err != nil || cs.String() != "cp850" {
		t.Fatalf("resolveCharset = %v, %v", cs, err)
	}
	WithCharset("klingon")(&o)
	if _, err := o.resolveCharset(); err == nil {
		t.Fatal("resolveCharset accepted an unknown charset")
	}
}

func TestParsePriceErrors(t *testing.T) {
	t.Parallel()

	if _, err := ParsePrice("", productdata.EUR); !errors.Is(err, ErrNoPrice) {
		t.Fatalf("empty price error = %v", err)
	}
	if _, err := ParsePrice("12", "EURO"); err == nil {
		t.Fatal("accepted an invalid currency")
	}
	if _, err := ParsePrice("12x", productdata.EUR); !errors.Is(err, ErrInvalidPrice) {
		t.Fatalf("invalid price error = %v", err)
	}
	amount, err := ParsePrice("1.234,56", productdata.EUR)
	if err != nil {
		t.Fatalf("explicit decimal returned error: %v", err)
	}
	if got := amount.String(); got != "1234.56 EUR" {
		t.Fatalf("amount = %q", got)
	}
}

func TestNormalizeHeaderCurrencyFallback(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		join("V", "050", "A", "20260115", "EURO", "Beispiel", "", "", "", "", "", "", "", "", "", ""),
		aRecord(map[int]string{2: "N", 3: "A-1"}),
		"E;3;Beispiel;",
	}, "\n")
	file, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	catalog, err := file.Catalog()
	if err != nil {
		t.Fatalf("Catalog returned error: %v", err)
	}
	if catalog.Header.Currency != productdata.EUR {
		t.Fatalf("currency = %q, want EUR", catalog.Header.Currency)
	}
}

func TestFieldSpecConfidence(t *testing.T) {
	t.Parallel()

	for _, spec := range FieldSpecs(KindArticle) {
		switch spec.Confidence {
		case ConfidenceSpecified, ConfidenceObserved, ConfidenceInferred, ConfidenceUnknown:
		default:
			t.Errorf("field %d has invalid confidence %q", spec.Position, spec.Confidence)
		}
	}
}

func TestParseRecordKindName(t *testing.T) {
	t.Parallel()

	if got := fmt.Sprintf("%c", byte(KindArticle)); got != "A" {
		t.Fatalf("byte conversion = %q", got)
	}
}

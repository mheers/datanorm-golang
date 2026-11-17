package datanorm

import (
	"errors"
	"strings"
	"testing"

	"github.com/mheers/datanorm-golang/productdata"
)

func parseSingleRecord(t *testing.T, line string) Record {
	t.Helper()
	file, err := Parse(strings.NewReader(joinLines(headerA("20260115"), line, "E;3;Beispiel;")))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(file.Records) != 3 {
		t.Fatalf("records = %d, want 3", len(file.Records))
	}
	return file.Records[1]
}

func TestPriceRecordVariants(t *testing.T) {
	t.Parallel()

	t.Run("no flag", func(t *testing.T) {
		t.Parallel()
		record := parseSingleRecord(t, "P;A-1;2;10;129900;04;;")
		price, ok := record.(*PriceRecord)
		if !ok {
			t.Fatalf("record = %T", record)
		}
		if price.ProcessingFlag != FlagUnknown {
			t.Errorf("flag = %q", price.ProcessingFlag)
		}
		if price.ArticleNumber != "A-1" || price.PriceTypeRaw != "2" || price.PriceUnitRaw != "10" || price.PriceRaw != "129900" {
			t.Errorf("price = %+v", price)
		}
		if price.DiscountGroup != "04" {
			t.Errorf("discount group = %q", price.DiscountGroup)
		}
	})

	t.Run("version 4 flag", func(t *testing.T) {
		t.Parallel()
		record := parseSingleRecord(t, "P;A;A-1;2;10;129900;04;")
		price, ok := record.(*PriceRecord)
		if !ok {
			t.Fatalf("record = %T", record)
		}
		if price.ProcessingFlag != FlagChange {
			t.Errorf("flag = %q", price.ProcessingFlag)
		}
		if price.ArticleNumber != "A-1" || price.PriceTypeRaw != "2" || price.PriceUnitRaw != "10" || price.PriceRaw != "129900" {
			t.Errorf("price = %+v", price)
		}
	})
}

func TestChangeRecord(t *testing.T) {
	t.Parallel()

	record := parseSingleRecord(t, "B;X;A-1;A-2;;")
	change, ok := record.(*ChangeRecord)
	if !ok {
		t.Fatalf("record = %T", record)
	}
	if change.ProcessingFlag != FlagRenumber || change.ArticleNumber != "A-1" || change.NewArticleNumber != "A-2" {
		t.Fatalf("change = %+v", change)
	}
}

func TestGroupAndDiscountRecords(t *testing.T) {
	t.Parallel()

	groupRecord := parseSingleRecord(t, "S;ELE;KABEL;Kabel;")
	group, ok := groupRecord.(*GroupRecord)
	if !ok {
		t.Fatalf("record = %T", groupRecord)
	}
	if group.MainGroup != "ELE" || group.ProductGroup != "KABEL" || group.Description != "Kabel" {
		t.Fatalf("group = %+v", group)
	}

	discountRecord := parseSingleRecord(t, "R;04;1;20;Kabelrabatt;")
	discount, ok := discountRecord.(*DiscountRecord)
	if !ok {
		t.Fatalf("record = %T", discountRecord)
	}
	if discount.Group != "04" || discount.KindRaw != "1" || discount.ValueRaw != "20" {
		t.Fatalf("discount = %+v", discount)
	}
}

func TestSurchargeRecord(t *testing.T) {
	t.Parallel()

	record := parseSingleRecord(t, "Z;N;A-1;1;1;1;Staffel;1;129900;10;100;")
	surcharge, ok := record.(*SurchargeRecord)
	if !ok {
		t.Fatalf("record = %T", record)
	}
	if surcharge.ProcessingFlag != FlagNew || surcharge.ArticleNumber != "A-1" {
		t.Fatalf("surcharge = %+v", surcharge)
	}
	if surcharge.ProcessingMarker != "1" || surcharge.BasisMarker != "1" || surcharge.PriceRaw != "129900" {
		t.Fatalf("surcharge = %+v", surcharge)
	}
	if surcharge.FromRaw != "10" || surcharge.ToRaw != "100" {
		t.Fatalf("surcharge range = %+v", surcharge)
	}
}

func TestTextMarkerHelpers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		marker    TextMarker
		short1    bool
		short2    bool
		long      bool
		dimension bool
	}{
		{"0", true, true, false, false},
		{"1", true, false, true, false},
		{"2", true, false, false, true},
		{"3", false, false, true, true},
		{"4", true, true, true, false},
		{"5", true, true, false, true},
		{"6", true, true, true, true},
	}
	for _, tt := range tests {
		if got := tt.marker.WantsShort1(); got != tt.short1 {
			t.Errorf("%s WantsShort1 = %v", tt.marker, got)
		}
		if got := tt.marker.WantsShort2(); got != tt.short2 {
			t.Errorf("%s WantsShort2 = %v", tt.marker, got)
		}
		if got := tt.marker.WantsLong(); got != tt.long {
			t.Errorf("%s WantsLong = %v", tt.marker, got)
		}
		if got := tt.marker.WantsDimension(); got != tt.dimension {
			t.Errorf("%s WantsDimension = %v", tt.marker, got)
		}
	}
	if TextMarker("7").Valid() {
		t.Fatal("marker 7 should be invalid")
	}
}

func TestScanRejectsVersion4(t *testing.T) {
	t.Parallel()

	err := Scan(strings.NewReader("V 170126Beispiel                                                         04EUR\r\n"), func(Record) error { return nil })
	if !errors.Is(err, ErrVersion4) {
		t.Fatalf("error = %v, want ErrVersion4", err)
	}
}

func TestScanRejectsMissingHeader(t *testing.T) {
	t.Parallel()

	err := Scan(strings.NewReader(""), func(Record) error { return nil })
	if !errors.Is(err, ErrMissingHeader) {
		t.Fatalf("error = %v, want ErrMissingHeader", err)
	}
}

func TestScanRejectsNonDatanorm(t *testing.T) {
	t.Parallel()

	err := Scan(strings.NewReader(aRecord(map[int]string{2: "N", 3: "A-1"})), func(Record) error { return nil })
	if !errors.Is(err, ErrNotDatanorm) {
		t.Fatalf("error = %v, want ErrNotDatanorm", err)
	}
}

func TestEndCountVariants(t *testing.T) {
	t.Parallel()

	variants := []string{"3", "2", "1"}
	for _, count := range variants {
		count := count
		t.Run(count, func(t *testing.T) {
			t.Parallel()
			input := strings.Join([]string{
				headerA("20260115"),
				aRecord(map[int]string{2: "N", 3: "A-1", 19: "4006381333931"}),
				"E;" + count + ";Beispiel;",
			}, "\n")
			file, err := Parse(strings.NewReader(input))
			if err != nil {
				t.Fatalf("Parse returned error: %v", err)
			}
			if hasDiagnostic(file.Diagnostics, "dn.end-count") {
				t.Fatalf("unexpected dn.end-count for %s: %v", count, file.Diagnostics)
			}
		})
	}
}

func TestHeaderSupplierName(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		join("V", "050", "A", "20260115", "EUR", "Beispiel GmbH", "Preisliste", "", "", "", "", "", "", "", "", ""),
		aRecord(map[int]string{2: "N", 3: "A-1"}),
		"E;3;Beispiel;",
	}, "\n")
	file, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if got, want := file.Header.SupplierName(), "Beispiel GmbH"; got != want {
		t.Fatalf("SupplierName = %q, want %q", got, want)
	}
}

func TestFormatDetection(t *testing.T) {
	t.Parallel()

	format, ok := productdata.Get("datanorm")
	if !ok {
		t.Fatal("datanorm format not registered")
	}
	if !format.Detect("DATANORM.001", nil) {
		t.Fatal("expected filename detection")
	}
	if !format.Detect("catalog.099", nil) {
		t.Fatal("expected numeric extension detection")
	}
	if !format.Detect("", []byte("V;050;A;20260115;EUR;Beispiel")) {
		t.Fatal("expected content detection")
	}
	if !format.Detect("", []byte("V 170126Beispiel")) {
		t.Fatal("expected version 4 detection")
	}
	if format.Detect("other.txt", []byte("hello")) {
		t.Fatal("unexpected detection")
	}
}

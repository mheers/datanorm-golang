package datanorm

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/mheers/datanorm-golang/productdata"
)

func TestRoleOfMarkers(t *testing.T) {
	t.Parallel()

	tests := map[string]DeliveryRole{
		"A": RoleArticles,
		"P": RolePrices,
		"S": RoleGroups,
		"R": RoleDiscounts,
		"J": RoleSets,
		"T": RoleTexts,
	}
	for marker, want := range tests {
		header := &HeaderRecord{DataMarker: marker}
		if got := RoleOf(&File{Header: header}); got != want {
			t.Errorf("RoleOf(marker %q) = %q, want %q", marker, got, want)
		}
	}
	if got := RoleOf(&File{Header: &HeaderRecord{DataMarker: "X"}, Path: "DATANORM.001"}); got != RoleArticles {
		t.Fatalf("fallback RoleOf = %q, want articles", got)
	}
	if got := RoleOf(nil); got != RoleUnknown {
		t.Fatalf("RoleOf(nil) = %q", got)
	}
}

func TestNewLoaderAndLoad(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeFile(t, dir, "DATANORM.001", joinLines(
		headerMarker("A", "20260115"),
		aRecord(map[int]string{2: "N", 3: "A-1", 4: "Kabel"}),
		"E;3;Beispiel;",
	))
	catalog, err := NewLoader().Load(context.Background(), path)
	if err != nil {
		t.Fatalf("Loader.Load returned error: %v", err)
	}
	if len(catalog.Articles) != 1 {
		t.Fatalf("articles = %d", len(catalog.Articles))
	}
	if _, err := Load(context.Background(), []string{path}, WithCharset("cp850")); err != nil {
		t.Fatalf("Load with option returned error: %v", err)
	}
}

func TestLoadErrors(t *testing.T) {
	t.Parallel()

	if _, err := Load(context.Background(), []string{"/does/not/exist"}); err == nil {
		t.Fatal("Load accepted a missing file")
	}
	dir := t.TempDir()
	empty := writeFile(t, dir, "DATANORM.001", "")
	if _, err := Load(context.Background(), []string{empty}); err == nil {
		t.Fatal("Load accepted an empty file")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Load(ctx, []string{empty}); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestSniffRoleBranches(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cases := []struct {
		name string
		data string
		want DeliveryRole
	}{
		{name: "price", data: "V;050;P;20260115;EUR;X;\n", want: RolePrices},
		{name: "groups", data: "V;050;S;20260115;EUR;X;\n", want: RoleGroups},
		{name: "discounts", data: "V;050;R;20260115;EUR;X;\n", want: RoleDiscounts},
		{name: "sets", data: "V;050;J;20260115;EUR;X;\n", want: RoleSets},
		{name: "texts", data: "V;050;T;20260115;EUR;X;\n", want: RoleTexts},
		{name: "articles", data: "V;050;A;20260115;EUR;X;\n", want: RoleArticles},
		{name: "unknown marker", data: "V;050;X;20260115;EUR;X;\n", want: RoleUnknown},
		{name: "no header", data: "garbage\n", want: RoleUnknown},
		{name: "empty", data: "", want: RoleUnknown},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := writeFile(t, dir, tt.name+".dat", tt.data)
			if got := sniffRole(path); got != tt.want {
				t.Fatalf("sniffRole = %q, want %q", got, tt.want)
			}
		})
	}
	if got := sniffRole(filepath.Join(dir, "DATANORM.RAB")); got != RoleDiscounts {
		t.Fatalf("fallback sniffRole = %q", got)
	}
	if got := sniffRole(filepath.Join(dir, "missing.RAB")); got != RoleDiscounts {
		t.Fatalf("missing file sniffRole = %q", got)
	}
}

func TestMergeArticleBranches(t *testing.T) {
	t.Parallel()

	quantity := &productdata.Quantity{Value: productdata.NewDecimal(1, 0)}
	base := productdata.Article{
		SupplierArticleNumber:     "A-1",
		Manufacturer:              "Hersteller",
		ManufacturerArticleNumber: "M-1",
		Type:                      "Typ",
		Matchcode:                 "MC",
		GTIN:                      "4006381333931",
		Unit:                      productdata.NormalizeUnit("MTR"),
		Texts: []productdata.Text{
			productdata.NewText(productdata.TextShort, "Kabel"),
			productdata.NewText(productdata.TextLong, "Lang"),
		},
		Groups:            []productdata.GroupRef{{Level: productdata.GroupMain, ID: "ELE"}},
		MinOrderQuantity:  quantity,
		PackagingQuantity: quantity,
		Status:            productdata.StatusActive,
		Replacement:       &productdata.ArticleRef{SupplierArticleNumber: "A-2"},
		Media:             []productdata.Media{{Type: productdata.MediaImage, Filename: "a.gif"}},
		Features:          []productdata.Feature{{Name: "Farbe", Value: "rot"}},
		Classifications:   []productdata.Classification{{System: "ETIM", Code: "EC000001"}},
		Associations:      []productdata.Association{{Kind: productdata.AssociationAlternative, Article: productdata.ArticleRef{SupplierArticleNumber: "A-3"}}},
		Logistics: productdata.Logistics{
			Weight:              quantity,
			Length:              quantity,
			Width:               quantity,
			Height:              quantity,
			PackagingQuantity:   quantity,
			CountryOfOrigin:     "DE",
			CustomsTariffNumber: "123",
			StorageFlag:         "L",
		},
		Extra:  map[string]string{"a": "1"},
		Source: productdata.Source{RecordType: "A", File: "a.001"},
		Raw:    "raw",
		Prices: []productdata.Price{{Type: productdata.PriceList}},
	}
	next := productdata.Article{
		Status:          productdata.StatusChanged,
		Texts:           []productdata.Text{productdata.NewText(productdata.TextShort, "Kabel neu")},
		Groups:          []productdata.GroupRef{{Level: productdata.GroupMain, ID: "ELE2"}},
		Media:           []productdata.Media{{Type: productdata.MediaDocument, Filename: "b.pdf"}},
		Features:        []productdata.Feature{{Name: "Farbe", Value: "blau"}},
		Classifications: []productdata.Classification{{System: "ETIM", Code: "EC000002"}},
		Associations:    []productdata.Association{{Kind: productdata.AssociationAccessory, Article: productdata.ArticleRef{SupplierArticleNumber: "A-4"}}},
		Extra:           map[string]string{"b": "2"},
		Source:          productdata.Source{RecordType: "P"},
		Prices:          []productdata.Price{{Type: productdata.PriceNet, Per: &productdata.Quantity{Value: productdata.NewDecimal(10, 0)}}},
	}

	merged := mergeArticle(base, next)
	if merged.SupplierArticleNumber != "A-1" || merged.Manufacturer != "Hersteller" ||
		merged.ManufacturerArticleNumber != "M-1" || merged.Type != "Typ" ||
		merged.Matchcode != "MC" || merged.GTIN != "4006381333931" {
		t.Fatalf("identity not merged: %+v", merged)
	}
	if merged.Unit.Code != "MTR" {
		t.Fatalf("unit = %+v", merged.Unit)
	}
	if len(merged.Texts) != 2 {
		t.Fatalf("texts = %+v", merged.Texts)
	}
	if len(merged.Groups) != 1 || merged.Groups[0].ID != "ELE2" {
		t.Fatalf("groups = %+v", merged.Groups)
	}
	if len(merged.Media) != 2 || len(merged.Classifications) != 2 || len(merged.Associations) != 2 {
		t.Fatalf("lists not merged: %+v", merged)
	}
	if len(merged.Features) != 1 || merged.Features[0].Value != "blau" {
		t.Fatalf("features = %+v", merged.Features)
	}
	if merged.MinOrderQuantity == nil || merged.PackagingQuantity == nil || merged.Replacement == nil {
		t.Fatalf("inherited values missing: %+v", merged)
	}
	if merged.Status != productdata.StatusChanged {
		t.Fatalf("status = %q", merged.Status)
	}
	if merged.Source.RecordType != "A" || merged.Source.File != "a.001" {
		t.Fatalf("source = %+v", merged.Source)
	}
	if merged.Logistics.Weight == nil || merged.Logistics.StorageFlag != "L" {
		t.Fatalf("logistics = %+v", merged.Logistics)
	}
	if len(merged.Prices) != 2 {
		t.Fatalf("prices = %+v", merged.Prices)
	}
	if merged.Prices[0].Per == nil || merged.Prices[0].Per.Unit.Code != "MTR" {
		t.Fatalf("price unit = %+v", merged.Prices[0].Per)
	}
	if merged.Extra["a"] != "1" || merged.Extra["b"] != "2" {
		t.Fatalf("extra = %+v", merged.Extra)
	}
	if merged.Raw != "raw" {
		t.Fatalf("raw = %v", merged.Raw)
	}
}

func TestRoleRankAndDiagnosticPrefix(t *testing.T) {
	t.Parallel()

	ranks := map[DeliveryRole]int{
		RoleDiscounts: 0,
		RoleGroups:    1,
		RoleArticles:  2,
		RolePrices:    3,
		RoleTexts:     4,
		RoleSets:      5,
		RoleUnknown:   6,
	}
	for role, want := range ranks {
		if got := roleRank(role); got != want {
			t.Errorf("roleRank(%q) = %d, want %d", role, got, want)
		}
	}
	diagnostics := productdata.Diagnostics{{Code: "a"}, {Code: "b", File: "kept"}}
	prefixed := prefixDiagnostics(diagnostics, "file.001")
	if prefixed[0].File != "file.001" || prefixed[1].File != "kept" {
		t.Fatalf("prefixed = %+v", prefixed)
	}
}

func TestFindDeliveryFilesErrorAndFiltering(t *testing.T) {
	t.Parallel()

	if _, err := FindDeliveryFiles("/does/not/exist"); err == nil {
		t.Fatal("expected an error for a missing directory")
	}
	dir := t.TempDir()
	writeFile(t, dir, "DATANORM.001", "")
	writeFile(t, dir, "DATPREIS.001", "")
	writeFile(t, dir, "DATASETS.001", "")
	writeFile(t, dir, "DATATEXT.001", "")
	writeFile(t, dir, "DATANORM.RAB", "")
	writeFile(t, dir, "DATANORM.WRG", "")
	writeFile(t, dir, "readme.txt", "")
	if err := os.Mkdir(filepath.Join(dir, "DATANORM.099"), 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	paths, err := FindDeliveryFiles(dir)
	if err != nil {
		t.Fatalf("FindDeliveryFiles: %v", err)
	}
	if len(paths) != 6 {
		t.Fatalf("paths = %d, want 6: %v", len(paths), paths)
	}
}

func TestIsDeliveryFileName(t *testing.T) {
	t.Parallel()

	valid := []string{"datanorm.001", "datanorm_komplettdatei.001", "datpreis.001", "datasets.001", "datatext.001", "x.rab", "x.wrg"}
	for _, name := range valid {
		if !isDeliveryFileName(name) {
			t.Errorf("%q should be a delivery file", name)
		}
	}
	invalid := []string{"readme.txt", "datanorm.txt", "archive.7z.001", "datanorm.01", "datpreis"}
	for _, name := range invalid {
		if isDeliveryFileName(name) {
			t.Errorf("%q should not be a delivery file", name)
		}
	}
}

func TestLoadOnlyPrices(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeFile(t, dir, "DATPREIS.001", joinLines(
		headerMarker("P", "20260115"),
		"P;A-1;2;1;99900;04;",
		"E;3;Beispiel;",
	))
	catalog, err := Load(context.Background(), []string{path})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(catalog.Articles) != 1 {
		t.Fatalf("articles = %d", len(catalog.Articles))
	}
	article := catalog.Articles[0]
	if article.Status != "" {
		t.Fatalf("status = %q, want empty", article.Status)
	}
	if article.Prices[0].Per != nil {
		t.Fatalf("unexpected per: %+v", article.Prices[0].Per)
	}
	if group, ok := article.Group(productdata.GroupDiscount); !ok || group.ID != "04" {
		t.Fatalf("discount group = %+v, %v", group, ok)
	}
}

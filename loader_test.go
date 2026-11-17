package datanorm

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mheers/datanorm-golang/productdata"
)

func headerMarker(marker, date string) string {
	return join("V", "050", marker, date, "EUR", "Beispiel GmbH", "Testfile", "", "", "", "", "", "", "", "", "")
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
	return path
}

func TestLoadDelivery(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	articleFile := writeFile(t, dir, "DATANORM.001", joinLines(
		headerMarker("A", "20260115"),
		aRecord(map[int]string{
			2: "N", 3: "A-1", 4: "Kabel", 6: "PCE", 7: "1", 8: "1", 9: "100",
			10: "04", 11: "ELE", 12: "KABEL", 23: "4", 24: "T-1",
		}),
		"T;N;T-1;1;01;Langtext;",
		"E;4;Beispiel;",
	))
	rabFile := writeFile(t, dir, "DATANORM.RAB", joinLines(
		headerMarker("R", "20260115"),
		"R;04;1;20;Kabelrabatt;",
		"E;3;Beispiel;",
	))
	wrgFile := writeFile(t, dir, "DATANORM.WRG", joinLines(
		headerMarker("S", "20260115"),
		"S;ELE;;Elektro;",
		"S;ELE;KABEL;Kabel und Leitungen;",
		"E;4;Beispiel;",
	))
	priceFile := writeFile(t, dir, "DATPREIS.001", joinLines(
		headerMarker("P", "20260116"),
		"P;A-1;2;1;8000;04;",
		"E;3;Beispiel;",
	))

	catalog, err := Load(context.Background(), []string{priceFile, articleFile, wrgFile, rabFile})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got := len(catalog.Articles); got != 1 {
		t.Fatalf("articles = %d, want 1", got)
	}
	article := catalog.Articles[0]
	if got := len(article.Prices); got != 2 {
		t.Fatalf("prices = %d, want 2", got)
	}
	if price, ok := article.Price(productdata.PriceNet); !ok || price.Amount.String() != "80.00 EUR" {
		t.Fatalf("net price = %v, %v", price, ok)
	}
	if price, ok := article.Price(productdata.PriceList); !ok || price.Amount.String() != "1.00 EUR" {
		t.Fatalf("list price = %v, %v", price, ok)
	}
	if group, ok := catalog.FindGroup(productdata.GroupProduct, "KABEL"); !ok || group.Name != "Kabel und Leitungen" {
		t.Fatalf("product group = %+v, %v", group, ok)
	}
	if group, ok := article.Group(productdata.GroupProduct); !ok || group.Name != "Kabel und Leitungen" {
		t.Fatalf("article product group = %+v, %v", group, ok)
	}
	if discount, ok := catalog.FindDiscountGroup("04"); !ok || len(discount.Discounts) != 1 {
		t.Fatalf("discount group = %+v, %v", discount, ok)
	}
	if got, want := catalog.Header.Description, "Beispiel GmbH"; got != want {
		t.Fatalf("description = %q, want %q", got, want)
	}
	if catalog.Diagnostics.HasErrors() {
		t.Fatalf("unexpected errors: %v", catalog.Diagnostics)
	}
}

func TestLoadDeletionAndRenumber(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	articleFile := writeFile(t, dir, "DATANORM.001", joinLines(
		headerMarker("A", "20260115"),
		aRecord(map[int]string{2: "N", 3: "A-1", 4: "Kabel", 6: "PCE", 7: "1", 8: "1", 9: "100", 23: "0"}),
		aRecord(map[int]string{2: "N", 3: "A-2", 4: "Dose", 6: "PCE", 7: "1", 8: "1", 9: "200", 23: "0"}),
		"E;4;Beispiel;",
	))
	deletionFile := writeFile(t, dir, "DATANORM.099", joinLines(
		headerMarker("A", "20260115"),
		"B;L;A-1;;;",
		"E;3;Beispiel;",
	))
	renumberFile := writeFile(t, dir, "DATANORM.098", joinLines(
		headerMarker("A", "20260115"),
		"B;X;A-2;A-20;;",
		"E;3;Beispiel;",
	))

	catalog, err := Load(context.Background(), []string{articleFile, renumberFile, deletionFile})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(catalog.Articles) != 2 {
		t.Fatalf("articles = %d, want 2: %+v", len(catalog.Articles), catalog.Articles)
	}
	if article, ok := catalog.FindArticle("A-1"); !ok || article.Status != productdata.StatusDeleted {
		t.Fatalf("A-1 = %+v, %v", article, ok)
	}
	if article, ok := catalog.FindArticle("A-20"); !ok || article.Status != productdata.StatusChanged {
		t.Fatalf("A-20 = %+v, %v", article, ok)
	}
	if _, ok := catalog.FindArticle("A-2"); ok {
		t.Fatal("A-2 should have been renumbered")
	}
}

func TestLoadRenamedFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	articleFile := writeFile(t, dir, "a.dat", joinLines(
		headerMarker("A", "20260115"),
		aRecord(map[int]string{2: "N", 3: "A-1", 4: "Kabel", 6: "PCE", 7: "1", 8: "1", 9: "100", 10: "04", 11: "ELE", 12: "KABEL"}),
		"E;3;Beispiel;",
	))
	discountFile := writeFile(t, dir, "b.dat", joinLines(
		headerMarker("R", "20260115"),
		"R;04;1;20;Kabelrabatt;",
		"E;3;Beispiel;",
	))
	groupFile := writeFile(t, dir, "c.dat", joinLines(
		headerMarker("S", "20260115"),
		"S;ELE;KABEL;Kabel und Leitungen;",
		"E;3;Beispiel;",
	))
	priceFile := writeFile(t, dir, "d.dat", joinLines(
		headerMarker("P", "20260116"),
		"P;A-1;2;1;8000;",
		"E;3;Beispiel;",
	))

	catalog, err := Load(context.Background(), []string{priceFile, articleFile, groupFile, discountFile})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(catalog.Articles) != 1 {
		t.Fatalf("articles = %d, want 1", len(catalog.Articles))
	}
	article := catalog.Articles[0]
	if price, ok := article.Price(productdata.PriceNet); !ok || price.Amount.String() != "80.00 EUR" {
		t.Fatalf("net price = %+v, %v", price, ok)
	}
	if group, ok := article.Group(productdata.GroupProduct); !ok || group.Name != "Kabel und Leitungen" {
		t.Fatalf("product group = %+v, %v", group, ok)
	}
	if group, ok := article.Group(productdata.GroupDiscount); !ok || group.Name != "Kabelrabatt" {
		t.Fatalf("discount group = %+v, %v", group, ok)
	}
}

func TestLoadRenumberInArticleFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeFile(t, dir, "DATANORM.002", joinLines(
		headerMarker("A", "20260115"),
		aRecord(map[int]string{2: "N", 3: "A-1", 4: "Kabel", 6: "PCE", 7: "1", 8: "1", 9: "100"}),
		"B;X;A-1;A-2;;",
		"E;4;Beispiel;",
	))
	catalog, err := Load(context.Background(), []string{path})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(catalog.Articles) != 1 {
		t.Fatalf("articles = %d, want 1: %+v", len(catalog.Articles), catalog.Articles)
	}
	if article := catalog.Articles[0]; article.SupplierArticleNumber != "A-2" {
		t.Fatalf("article = %q, want A-2", article.SupplierArticleNumber)
	}
}

func TestFindDeliveryFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, dir, "DATANORM.001", "")
	writeFile(t, dir, "DATANORM.RAB", "")
	writeFile(t, dir, "DATANORM.WRG", "")
	writeFile(t, dir, "DATPREIS.001", "")
	writeFile(t, dir, "readme.txt", "")

	paths, err := FindDeliveryFiles(dir)
	if err != nil {
		t.Fatalf("FindDeliveryFiles returned error: %v", err)
	}
	if got, want := len(paths), 4; got != want {
		t.Fatalf("paths = %d, want %d: %v", got, want, paths)
	}
	order := []string{"DATANORM.RAB", "DATANORM.WRG", "DATANORM.001", "DATPREIS.001"}
	for i, want := range order {
		if got := filepath.Base(paths[i]); got != want {
			t.Fatalf("paths[%d] = %q, want %q", i, got, want)
		}
	}
}

func TestRoleOf(t *testing.T) {
	t.Parallel()

	tests := map[string]DeliveryRole{
		"DATANORM.001": RoleArticles,
		"DATANORM.RAB": RoleDiscounts,
		"DATANORM.WRG": RoleGroups,
		"DATPREIS.001": RolePrices,
		"DATASETS.001": RoleSets,
		"DATATEXT.001": RoleTexts,
		"unknown.dat":  RoleUnknown,
	}
	for name, want := range tests {
		if got := RoleFromName(name); got != want {
			t.Errorf("RoleFromName(%q) = %q, want %q", name, got, want)
		}
	}
}

func joinLines(lines ...string) string {
	out := ""
	for _, line := range lines {
		out += line + "\r\n"
	}
	return out
}

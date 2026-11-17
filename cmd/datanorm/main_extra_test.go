package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mheers/datanorm-golang"
	"github.com/mheers/datanorm-golang/productdata"
)

func cmdArticleRecord(number string) string {
	fields := make([]string, 30)
	fields[0] = "A"
	fields[1] = "N"
	fields[2] = number
	fields[3] = "Beispiel Kabel"
	fields[5] = "PCE"
	fields[6] = "1"
	fields[7] = "100"
	fields[8] = "129900"
	fields[9] = "04"
	fields[10] = "ELE"
	fields[11] = "KABEL"
	fields[18] = "4006381333931"
	fields[22] = "0"
	return strings.Join(fields, ";")
}

func writeDelivery(t *testing.T) []string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"DATANORM.001": strings.Join([]string{
			"V;050;A;20260115;EUR;Beispiel GmbH;Preisliste;",
			cmdArticleRecord("A-1"),
			"E;3;Beispiel;",
		}, "\r\n") + "\r\n",
		"DATANORM.RAB": "V;050;R;20260115;EUR;Beispiel GmbH;\r\nR;04;1;20;Kabelrabatt;\r\nE;3;Beispiel;\r\n",
		"DATANORM.WRG": "V;050;S;20260115;EUR;Beispiel GmbH;\r\nS;ELE;;Elektro;\r\nS;ELE;KABEL;Kabel;\r\nE;4;Beispiel;\r\n",
		"DATPREIS.001": "V;050;P;20260116;EUR;Beispiel GmbH;\r\nP;A-1;2;1;99900;04;\r\nE;3;Beispiel;\r\n",
	}
	var paths []string
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		paths = append(paths, path)
	}
	return paths
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestRunHelpAndErrors(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{{"help"}, {"-h"}, {"--help"}} {
		var out, errOut bytes.Buffer
		if code := run(args, &out, &errOut); code != 0 {
			t.Errorf("run(%v) = %d", args, code)
		}
		if !strings.Contains(out.String(), "Usage:") {
			t.Errorf("run(%v) output = %q", args, out.String())
		}
	}
	var out, errOut bytes.Buffer
	if code := run([]string{"nope", "x"}, &out, &errOut); code != 2 {
		t.Fatalf("unknown command code = %d", code)
	}
	if !strings.Contains(errOut.String(), "unknown command") {
		t.Fatalf("stderr = %q", errOut.String())
	}
	if code := run([]string{"info"}, &out, &errOut); code != 2 {
		t.Fatalf("no paths code = %d", code)
	}
	if code := run([]string{"info", "-nope"}, &out, &errOut); code != 2 {
		t.Fatalf("bad flag code = %d", code)
	}
}

func TestRunInfoDelivery(t *testing.T) {
	t.Parallel()

	paths := writeDelivery(t)
	args := append([]string{"info", "-delivery"}, paths...)
	var out, errOut bytes.Buffer
	if code := run(args, &out, &errOut); code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, errOut.String())
	}
	for _, want := range []string{"Delivery:    4 file(s)", "Supplier:    Beispiel GmbH", "Price date:  2026-01-15", "Groups:      2", "Discounts:   1"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}

func TestRunInfoErrors(t *testing.T) {
	t.Parallel()

	if code := run([]string{"info", "/does/not/exist"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 1 {
		t.Fatalf("missing single file code = %d", code)
	}
	dir := t.TempDir()
	bad := filepath.Join(dir, "DATANORM.001")
	if err := os.WriteFile(bad, []byte("not datanorm\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if code := run([]string{"info", "-delivery", bad}, &bytes.Buffer{}, &bytes.Buffer{}); code != 1 {
		t.Fatalf("invalid delivery code = %d", code)
	}
}

func TestRunValidateBranches(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	invalid := filepath.Join(dir, "DATANORM.001")
	content := "V;050;A;20260115;EUR;Beispiel;\r\nA;N;;Kabel;;PCE;1;1;100;;;;;Kabel;;;;;;;;;;0;;;;;;;\r\nE;3;Beispiel;\r\n"
	if err := os.WriteFile(invalid, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	var out, errOut bytes.Buffer
	if code := run([]string{"validate", invalid}, &out, &errOut); code != 1 {
		t.Fatalf("code = %d, want 1; stdout = %s", code, out.String())
	}
	if !strings.Contains(out.String(), "dn.article-number") {
		t.Fatalf("stdout = %q", out.String())
	}
	if code := run([]string{"validate", "/does/not/exist"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 1 {
		t.Fatalf("missing file code = %d", code)
	}
	paths := writeDelivery(t)
	args := append([]string{"validate", "-delivery"}, paths...)
	if code := run(args, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("valid delivery code = %d", code)
	}
	if code := run(append([]string{"validate", "-strict", "-delivery"}, paths...), &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("strict valid delivery code = %d", code)
	}
}

func TestRunArticlesDelivery(t *testing.T) {
	t.Parallel()

	paths := writeDelivery(t)
	args := append([]string{"articles", "-delivery"}, paths...)
	var out, errOut bytes.Buffer
	if code := run(args, &out, &errOut); code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, errOut.String())
	}
	for _, want := range []string{"A-1", "1299.00", "999.00", "04", "ELE", "KABEL"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
	if code := run([]string{"articles", "/does/not/exist"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 1 {
		t.Fatalf("missing file code = %d", code)
	}
}

func TestRunJSON(t *testing.T) {
	t.Parallel()

	path := writeTempDatanorm(t, minimalFile())
	var out, errOut bytes.Buffer
	if code := run([]string{"json", path}, &out, &errOut); code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, errOut.String())
	}
	var decoded map[string]any
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if decoded["header"] == nil {
		t.Fatalf("missing header: %v", decoded)
	}

	paths := writeDelivery(t)
	args := append([]string{"json", "-delivery"}, paths...)
	out.Reset()
	if code := run(args, &out, &errOut); code != 0 {
		t.Fatalf("delivery code = %d, stderr = %s", code, errOut.String())
	}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid delivery JSON: %v", err)
	}
	if code := run([]string{"json", "/does/not/exist"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 1 {
		t.Fatalf("missing file code = %d", code)
	}
}

func TestRunEncodings(t *testing.T) {
	t.Parallel()

	path := writeTempDatanorm(t, minimalFile())
	var out, errOut bytes.Buffer
	if code := run([]string{"encodings", path, "/does/not/exist"}, &out, &errOut); code != 0 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(out.String(), "cp850") || !strings.Contains(out.String(), "error") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestCommandWriterErrors(t *testing.T) {
	t.Parallel()

	path := writeTempDatanorm(t, minimalFile())
	if code := run([]string{"articles", path}, errWriter{}, &bytes.Buffer{}); code != 1 {
		t.Fatalf("articles writer error code = %d", code)
	}
	if code := run([]string{"json", path}, errWriter{}, &bytes.Buffer{}); code != 1 {
		t.Fatalf("json writer error code = %d", code)
	}
}

func TestPriceGroupAndDateHelpers(t *testing.T) {
	t.Parallel()

	article := loadTestArticle(t)
	if price, currency := priceFields(article, productdata.PriceList); price != "1299.00" || currency != "EUR" {
		t.Fatalf("priceFields = %q, %q", price, currency)
	}
	if price, currency := priceFields(article, productdata.PriceRecommendedRetail); price != "" || currency != "" {
		t.Fatalf("missing priceFields = %q, %q", price, currency)
	}
	discount, main, product := groupFields(article)
	if discount != "04" || main != "ELE" || product != "KABEL" {
		t.Fatalf("groupFields = %q, %q, %q", discount, main, product)
	}
	date := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	if got := formatDate("20260102", true, date); got != "2026-01-02" {
		t.Fatalf("formatDate = %q", got)
	}
	if got := formatDate("bad", false, time.Time{}); got != "bad (invalid)" {
		t.Fatalf("formatDate = %q", got)
	}
	if got := formatDate("", false, time.Time{}); got != "" {
		t.Fatalf("formatDate = %q", got)
	}
}

func loadTestArticle(t *testing.T) productdata.Article {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "DATANORM.001")
	content := "V;050;A;20260115;EUR;Beispiel GmbH;\r\n" +
		"A;N;A-1;Kabel;;PCE;1;100;129900;04;ELE;KABEL;;;;;;;;;4006381333931;;;;0;;;;;;;\r\n" +
		"E;3;Beispiel;\r\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	file, err := datanorm.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	catalog, err := file.Catalog()
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}
	return catalog.Articles[0]
}

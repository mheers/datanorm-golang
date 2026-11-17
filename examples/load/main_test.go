package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func articleLine(number string) string {
	fields := make([]string, 30)
	fields[0] = "A"
	fields[1] = "N"
	fields[2] = number
	fields[3] = "Beispiel Kabel"
	fields[5] = "PCE"
	fields[6] = "1"
	fields[7] = "1"
	fields[8] = "129900"
	fields[18] = "4006381333931"
	fields[22] = "0"
	return strings.Join(fields, ";")
}

func writeDelivery(t *testing.T, dir string) {
	t.Helper()
	content := strings.Join([]string{
		"V;050;A;20260115;EUR;Beispiel GmbH;Preisliste;",
		articleLine("A-1"),
		"E;3;Beispiel;",
	}, "\r\n") + "\r\n"
	path := filepath.Join(dir, "DATANORM.001")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestRunLoadExplicitFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeDelivery(t, dir)
	var out, errOut bytes.Buffer
	if code := run([]string{filepath.Join(dir, "DATANORM.001")}, dir, &out, &errOut); code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, errOut.String())
	}
	for _, want := range []string{"files:     1", "articles:  1", "currency:  EUR", "price date: 2026-01-15"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}

func TestRunLoadDiscoversFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeDelivery(t, dir)
	var out, errOut bytes.Buffer
	if code := run(nil, dir, &out, &errOut); code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "files:     1") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestRunLoadErrors(t *testing.T) {
	t.Parallel()

	var out, errOut bytes.Buffer
	if code := run(nil, t.TempDir(), &out, &errOut); code != 1 {
		t.Fatalf("empty dir code = %d", code)
	}
	if !strings.Contains(errOut.String(), "no DATANORM files found") {
		t.Fatalf("stderr = %q", errOut.String())
	}
	errOut.Reset()
	if code := run(nil, "/does/not/exist", &out, &errOut); code != 1 {
		t.Fatalf("missing dir code = %d", code)
	}
	errOut.Reset()
	if code := run([]string{"/does/not/exist"}, ".", &out, &errOut); code != 1 {
		t.Fatalf("missing file code = %d", code)
	}
}

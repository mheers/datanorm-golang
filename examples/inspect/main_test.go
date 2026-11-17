package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mheers/datanorm-golang"
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

func writeSample(t *testing.T, articles int) string {
	t.Helper()
	lines := []string{"V;050;A;20260115;EUR;Beispiel GmbH;Preisliste;"}
	for i := 0; i < articles; i++ {
		lines = append(lines, articleLine("A-"+string(rune('0'+i))))
	}
	lines = append(lines, "E;3;Beispiel;")
	path := filepath.Join(t.TempDir(), "DATANORM.001")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\r\n")+"\r\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func TestRunInspect(t *testing.T) {
	t.Parallel()

	path := writeSample(t, 1)
	var out, errOut bytes.Buffer
	if code := run([]string{path}, &out, &errOut); code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, errOut.String())
	}
	for _, want := range []string{"encoding:", "version:", "articles: 1", "A-0", "1299.00 EUR"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}

func TestRunInspectLimitsArticles(t *testing.T) {
	t.Parallel()

	path := writeSample(t, 11)
	var out, errOut bytes.Buffer
	if code := run([]string{path}, &out, &errOut); code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, errOut.String())
	}
	if strings.Contains(out.String(), "A-:") {
		t.Fatalf("eleventh article was printed:\n%s", out.String())
	}
}

func TestRunInspectErrors(t *testing.T) {
	t.Parallel()

	var out, errOut bytes.Buffer
	if code := run(nil, &out, &errOut); code != 2 {
		t.Fatalf("no args code = %d", code)
	}
	if !strings.Contains(errOut.String(), "usage") {
		t.Fatalf("stderr = %q", errOut.String())
	}
	if code := run([]string{"/does/not/exist"}, &out, &errOut); code != 1 {
		t.Fatalf("missing file code = %d", code)
	}
	invalid := filepath.Join(t.TempDir(), "invalid.001")
	if err := os.WriteFile(invalid, []byte("not datanorm\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if code := run([]string{invalid}, &out, &errOut); code != 1 {
		t.Fatalf("invalid file code = %d", code)
	}
}

func TestReportFileError(t *testing.T) {
	t.Parallel()

	var out, errOut bytes.Buffer
	if code := reportFile(&datanorm.File{}, &out, &errOut); code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "error") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

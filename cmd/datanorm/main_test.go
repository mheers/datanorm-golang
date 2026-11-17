package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempDatanorm(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "DATANORM.001")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func minimalFile() string {
	return fileWithEAN("4006381333931")
}

func fileWithEAN(ean string) string {
	fields := make([]string, 30)
	fields[0] = "A"
	fields[1] = "N"
	fields[2] = "A-1"
	fields[3] = "Beispiel Kabel"
	fields[5] = "PCE"
	fields[6] = "1"
	fields[7] = "1"
	fields[8] = "100"
	fields[18] = ean
	fields[22] = "0"
	return strings.Join([]string{
		"V;050;A;20260115;EUR;Beispiel GmbH;Preisliste;",
		strings.Join(fields, ";"),
		"E;3;Beispiel;",
	}, "\r\n") + "\r\n"
}

func TestRunVersion(t *testing.T) {
	t.Parallel()

	var out, errOut bytes.Buffer
	if code := run([]string{"version"}, &out, &errOut); code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "datanorm") || !strings.Contains(out.String(), "5.0") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestRunInfo(t *testing.T) {
	t.Parallel()

	path := writeTempDatanorm(t, minimalFile())
	var out, errOut bytes.Buffer
	if code := run([]string{"info", path}, &out, &errOut); code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, errOut.String())
	}
	for _, want := range []string{"DATANORM 5.0", "Articles", "Records"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}

func TestRunValidateStrict(t *testing.T) {
	t.Parallel()

	content := fileWithEAN("4006381333932")
	path := writeTempDatanorm(t, content)

	var out, errOut bytes.Buffer
	if code := run([]string{"validate", "-strict", path}, &out, &errOut); code != 1 {
		t.Fatalf("code = %d, want 1; stdout = %s; stderr = %s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "dn.ean-checksum") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestRunArticles(t *testing.T) {
	t.Parallel()

	path := writeTempDatanorm(t, minimalFile())
	var out, errOut bytes.Buffer
	if code := run([]string{"articles", path}, &out, &errOut); code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, errOut.String())
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines = %d, want 2:\n%s", len(lines), out.String())
	}
	if !strings.HasPrefix(lines[0], "article_number,") {
		t.Fatalf("header = %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "A-1,") {
		t.Fatalf("article = %q", lines[1])
	}
}

func TestRunUsage(t *testing.T) {
	t.Parallel()

	var out, errOut bytes.Buffer
	if code := run(nil, &out, &errOut); code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(errOut.String(), "Usage:") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

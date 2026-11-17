package charset

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestCharsetString(t *testing.T) {
	t.Parallel()

	if got := CP850.String(); got != "cp850" {
		t.Fatalf("String = %q", got)
	}
}

func TestParseAliases(t *testing.T) {
	t.Parallel()

	aliases := map[string]Charset{
		"detect":         Auto,
		"unicode":        UTF8,
		"dos437":         CP437,
		"dos850":         CP850,
		"oem":            CP850,
		"1252":           Windows1252,
		"win1252":        Windows1252,
		"latin1-windows": Windows1252,
		"latin9":         ISO885915,
		"latin-9":        ISO885915,
		"latin-1":        ISO88591,
	}
	for input, want := range aliases {
		got, err := Parse(input)
		if err != nil {
			t.Errorf("Parse(%q): %v", input, err)
			continue
		}
		if got != want {
			t.Errorf("Parse(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestDecodeErrors(t *testing.T) {
	t.Parallel()

	if _, err := Decode([]byte("x"), Charset("klingon")); err == nil {
		t.Fatal("Decode accepted an unknown charset")
	}
	if _, err := Encode("x", Charset("klingon")); err == nil {
		t.Fatal("Encode accepted an unknown charset")
	}
}

func TestDecodeUTF8BOM(t *testing.T) {
	t.Parallel()

	text, err := Decode(append([]byte{0xEF, 0xBB, 0xBF}, []byte("Preis")...), UTF8)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if text != "Preis" {
		t.Fatalf("Decode = %q", text)
	}
	if got := Detect(append([]byte{0xEF, 0xBB, 0xBF}, []byte("Preis")...)); got != UTF8 {
		t.Fatalf("Detect BOM = %q", got)
	}
}

func TestDetectBranches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data []byte
		want Charset
	}{
		{name: "empty", data: nil, want: CP850},
		{name: "ascii", data: []byte("Preisliste"), want: CP850},
		{name: "utf8", data: []byte("Größe"), want: UTF8},
		{name: "cp850 automated", data: []byte{0x5A, 0x75, 0x62, 0x65, 0x68, 0x94, 0x72}, want: CP850},
		{name: "cp437 vs cp850", data: append([]byte("Preis "), 0x9B), want: CP850},
		{name: "windows", data: append([]byte("Preis f"), 0xFC, 0x72), want: Windows1252},
		{name: "iso15 euro", data: append([]byte("Preis "), 0xA4), want: ISO885915},
		{name: "invalid high bytes", data: []byte{0x80, 0x81, 0x82, 0x83}, want: CP850},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Detect(tt.data); got != tt.want {
				t.Fatalf("Detect = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDetectInvalidUTF8WithoutWords(t *testing.T) {
	t.Parallel()

	// Bytes 0x80-0x83 are printable Windows-1252 characters, so that
	// encoding wins over the DOS code pages.
	data := []byte{0x80, 0x80, 0x80, 0x80}
	if got := Detect(data); got != Windows1252 {
		t.Fatalf("Detect = %q, want Windows1252", got)
	}
}

func TestScoreTextBranches(t *testing.T) {
	t.Parallel()

	if scoreText("") != 0 {
		t.Fatal("empty text should score zero")
	}
	if scoreText("der die das und für mit von") <= 0 {
		t.Fatal("German words should score positive")
	}
	if scoreText("Ã¤Ã¶Ã¼") >= scoreText("äöü") {
		t.Fatal("mojibake should score lower than correct umlauts")
	}
	if scoreText("€") <= scoreText("¤") {
		t.Fatal("euro should score higher than currency sign")
	}
	if scoreText("\x00\x01") >= 0 {
		t.Fatal("control characters should score negative")
	}
	if scoreText("┌─┐") >= 0 {
		t.Fatal("box drawing should score negative")
	}
	if scoreText("αβγ") >= 0 {
		t.Fatal("Greek letters should score negative")
	}
	if scoreText("\u00AD") >= 0 {
		t.Fatal("soft hyphen should score negative")
	}
	if scoreText("ß") <= 0 {
		t.Fatal("sharp s should score positive")
	}
	if scoreText("é") <= 0 {
		t.Fatal("Latin-1 supplement should score positive")
	}
	if scoreText("☃") >= 0 {
		t.Fatal("unrelated symbols should score negative")
	}
}

func TestNewReaderExplicitEncodings(t *testing.T) {
	t.Parallel()

	for _, cs := range []Charset{UTF8, CP437, CP850, Windows1252, ISO885915, ISO88591} {
		cs := cs
		t.Run(cs.String(), func(t *testing.T) {
			t.Parallel()
			encoded, err := Encode("Preis für Größe", cs)
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			reader, err := NewReader(bytes.NewReader(encoded), cs)
			if err != nil {
				t.Fatalf("NewReader: %v", err)
			}
			decoded, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("ReadAll: %v", err)
			}
			if string(decoded) != "Preis für Größe" {
				t.Fatalf("decoded = %q", decoded)
			}
		})
	}
	if _, err := NewReader(strings.NewReader("x"), Charset("klingon")); err == nil {
		t.Fatal("NewReader accepted an unknown charset")
	}
}

func TestNewReaderAutoDetect(t *testing.T) {
	t.Parallel()

	encoded, err := Encode("Zubehör für Größe", CP850)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	reader, err := NewReader(bytes.NewReader(encoded), Auto)
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}
	decoded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(decoded) != "Zubehör für Größe" {
		t.Fatalf("decoded = %q", decoded)
	}
}

func TestNewWriter(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	writer, err := NewWriter(&buf, CP850)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if _, err := writer.Write([]byte("Zubehör")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	decoded, err := Decode(buf.Bytes(), CP850)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if decoded != "Zubehör" {
		t.Fatalf("decoded = %q", decoded)
	}

	buf.Reset()
	writer, err = NewWriter(&buf, UTF8)
	if err != nil {
		t.Fatalf("NewWriter UTF8: %v", err)
	}
	if _, err := writer.Write([]byte("Zubehör")); err != nil {
		t.Fatalf("Write UTF8: %v", err)
	}
	if buf.String() != "Zubehör" {
		t.Fatalf("UTF8 bytes = %q", buf.String())
	}

	buf.Reset()
	writer, err = NewWriter(&buf, Auto)
	if err != nil {
		t.Fatalf("NewWriter Auto: %v", err)
	}
	if _, err := writer.Write([]byte("Preis")); err != nil {
		t.Fatalf("Write Auto: %v", err)
	}
	if _, err := NewWriter(&buf, Charset("klingon")); err == nil {
		t.Fatal("NewWriter accepted an unknown charset")
	}
}

func TestDecodeEncodeErrorsPropagate(t *testing.T) {
	t.Parallel()

	// A rune that CP850 cannot represent must produce an error, not silent
	// substitution.
	if _, err := Encode("€", CP850); err == nil {
		t.Fatal("Encode silently substituted an unrepresentable rune")
	}
	if _, err := Encode("☃", ISO88591); err == nil {
		t.Fatal("Encode silently substituted an unrepresentable rune")
	}
}

func TestEncodeAutoUsesCP850(t *testing.T) {
	t.Parallel()

	data, err := Encode("Zubehör", Auto)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.Contains(data, []byte{0x94}) {
		t.Fatalf("Auto encoding did not use CP850: % X", data)
	}
}

func TestDecodeAuto(t *testing.T) {
	t.Parallel()

	encoded, err := Encode("Größe", CP850)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	decoded, err := Decode(encoded, Auto)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if decoded != "Größe" {
		t.Fatalf("decoded = %q", decoded)
	}
}

func TestErrorsAreWrapped(t *testing.T) {
	t.Parallel()

	_, err := Encode("€", CP850)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "charset") {
		t.Fatalf("error = %v", err)
	}
	if errors.Is(err, io.EOF) {
		t.Fatal("unexpected EOF match")
	}
}

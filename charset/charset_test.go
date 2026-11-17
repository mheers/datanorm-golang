package charset

import (
	"strings"
	"testing"
)

// Byte sequences generated with iconv/Python codecs from synthetic text.
var (
	cp850Zubehoer = []byte{0x5A, 0x75, 0x62, 0x65, 0x68, 0x94, 0x72, 0x20, 0x66, 0x81, 0x72, 0x20, 0x47, 0x72, 0x94, 0xE1, 0x65} // Zubehör für Größe
	cp850Stueck   = []byte{0x53, 0x74, 0x81, 0x63, 0x6B, 0x20, 0x99, 0x6C, 0x20, 0x84, 0x94, 0x81, 0x8E, 0x99, 0x9A, 0xE1}       // Stück Öl äöüÄÖÜß
	win1252Text   = []byte{0x50, 0x72, 0x65, 0x69, 0x73, 0x20, 0x66, 0xFC, 0x72, 0x20, 0x47, 0x72, 0xF6, 0xDF, 0x65}             // Preis für Größe
	utf8Text      = []byte("Preis für Größe")
	iso15Euro     = []byte{0x31, 0x32, 0x2C, 0x35, 0x30, 0x20, 0xA4} // 12,50 €
)

func TestDetect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data []byte
		want Charset
	}{
		{name: "utf8", data: utf8Text, want: UTF8},
		{name: "cp850", data: cp850Zubehoer, want: CP850},
		{name: "cp850-umlauts", data: cp850Stueck, want: CP850},
		{name: "win1252", data: win1252Text, want: Windows1252},
		{name: "iso8859-15-euro", data: iso15Euro, want: ISO885915},
		{name: "ascii", data: []byte("V;050;A;20260115;EUR;Example"), want: CP850},
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

func TestDecode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data []byte
		cs   Charset
		want string
	}{
		{name: "cp850", data: cp850Zubehoer, cs: CP850, want: "Zubehör für Größe"},
		{name: "cp850-all", data: cp850Stueck, cs: CP850, want: "Stück Öl äöüÄÖÜß"},
		{name: "win1252", data: win1252Text, cs: Windows1252, want: "Preis für Größe"},
		{name: "utf8", data: utf8Text, cs: UTF8, want: "Preis für Größe"},
		{name: "iso15", data: iso15Euro, cs: ISO885915, want: "12,50 €"},
		{name: "auto-cp850", data: cp850Zubehoer, cs: Auto, want: "Zubehör für Größe"},
		{name: "auto-utf8", data: utf8Text, cs: Auto, want: "Preis für Größe"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Decode(tt.data, tt.cs)
			if err != nil {
				t.Fatalf("Decode returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("Decode = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDecodeRejectsUnknown(t *testing.T) {
	t.Parallel()

	if _, err := Decode([]byte("x"), Charset("klingon")); err == nil {
		t.Fatal("expected error for unknown charset")
	}
}

func TestEncodeRoundTrip(t *testing.T) {
	t.Parallel()

	for _, cs := range []Charset{CP437, CP850, Windows1252, ISO885915, UTF8} {
		cs := cs
		t.Run(string(cs), func(t *testing.T) {
			t.Parallel()
			encoded, err := Encode("Zubehör für Größe", cs)
			if err != nil {
				t.Fatalf("Encode returned error: %v", err)
			}
			decoded, err := Decode(encoded, cs)
			if err != nil {
				t.Fatalf("Decode returned error: %v", err)
			}
			if decoded != "Zubehör für Größe" {
				t.Fatalf("round trip = %q", decoded)
			}
		})
	}
}

func TestParse(t *testing.T) {
	t.Parallel()

	tests := map[string]Charset{
		"cp850":        CP850,
		"IBM850":       CP850,
		"850":          CP850,
		"cp437":        CP437,
		"windows-1252": Windows1252,
		"cp1252":       Windows1252,
		"iso-8859-15":  ISO885915,
		"latin9":       ISO885915,
		"iso-8859-1":   ISO88591,
		"latin1":       ISO88591,
		"utf-8":        UTF8,
		"utf8":         UTF8,
		"auto":         Auto,
	}
	for input, want := range tests {
		got, err := Parse(input)
		if err != nil {
			t.Errorf("Parse(%q) returned error: %v", input, err)
			continue
		}
		if got != want {
			t.Errorf("Parse(%q) = %q, want %q", input, got, want)
		}
	}
	if _, err := Parse("nope"); err == nil {
		t.Fatal("expected error for unknown name")
	}
}

func TestNewReaderDetectsAuto(t *testing.T) {
	t.Parallel()

	reader, err := NewReader(strings.NewReader(string(cp850Zubehoer)), Auto)
	if err != nil {
		t.Fatalf("NewReader returned error: %v", err)
	}
	out := new(strings.Builder)
	buf := make([]byte, 64)
	for {
		n, readErr := reader.Read(buf)
		out.Write(buf[:n])
		if readErr != nil {
			break
		}
	}
	if got, want := out.String(), "Zubehör für Größe"; got != want {
		t.Fatalf("NewReader = %q, want %q", got, want)
	}
}

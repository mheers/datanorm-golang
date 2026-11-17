// Package charset handles the legacy character encodings used by DATANORM and
// similar formats. It can detect an encoding from a byte sample, decode to
// UTF-8, encode from UTF-8, and wrap streams.
package charset

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// Charset identifies a character encoding.
type Charset string

// Supported encodings.
const (
	Auto        Charset = "auto"
	UTF8        Charset = "utf-8"
	CP437       Charset = "cp437"
	CP850       Charset = "cp850"
	Windows1252 Charset = "windows-1252"
	ISO885915   Charset = "iso-8859-15"
	ISO88591    Charset = "iso-8859-1"
)

// sampleSize is the number of bytes used for automatic detection.
const sampleSize = 32 * 1024

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// String returns the canonical name.
func (c Charset) String() string {
	return string(c)
}

// Parse accepts a canonical name or a common alias.
func Parse(name string) (Charset, error) {
	key := strings.ToLower(strings.TrimSpace(name))
	switch key {
	case "", "auto", "detect":
		return Auto, nil
	case "utf-8", "utf8", "unicode":
		return UTF8, nil
	case "cp437", "ibm437", "437", "dos437":
		return CP437, nil
	case "cp850", "ibm850", "850", "dos850", "oem":
		return CP850, nil
	case "windows-1252", "cp1252", "1252", "win1252", "latin1-windows":
		return Windows1252, nil
	case "iso-8859-15", "iso8859-15", "latin9", "latin-9":
		return ISO885915, nil
	case "iso-8859-1", "iso8859-1", "latin1", "latin-1":
		return ISO88591, nil
	default:
		return "", fmt.Errorf("charset: unknown encoding %q", name)
	}
}

func decoderFor(cs Charset) (transform.Transformer, error) {
	switch cs {
	case UTF8:
		return nil, nil
	case CP437:
		return charmap.CodePage437.NewDecoder(), nil
	case CP850:
		return charmap.CodePage850.NewDecoder(), nil
	case Windows1252:
		return charmap.Windows1252.NewDecoder(), nil
	case ISO885915:
		return charmap.ISO8859_15.NewDecoder(), nil
	case ISO88591:
		return charmap.ISO8859_1.NewDecoder(), nil
	default:
		return nil, fmt.Errorf("charset: unknown encoding %q", cs)
	}
}

func encoderFor(cs Charset) (transform.Transformer, error) {
	switch cs {
	case UTF8:
		return nil, nil
	case CP437:
		return charmap.CodePage437.NewEncoder(), nil
	case CP850:
		return charmap.CodePage850.NewEncoder(), nil
	case Windows1252:
		return charmap.Windows1252.NewEncoder(), nil
	case ISO885915:
		return charmap.ISO8859_15.NewEncoder(), nil
	case ISO88591:
		return charmap.ISO8859_1.NewEncoder(), nil
	default:
		return nil, fmt.Errorf("charset: unknown encoding %q", cs)
	}
}

// Decode converts bytes in the given encoding to UTF-8. With Auto, the encoding
// is detected from the bytes.
func Decode(data []byte, cs Charset) (string, error) {
	if cs == Auto {
		cs = Detect(data)
	}
	if cs == UTF8 {
		return string(bytes.TrimPrefix(data, utf8BOM)), nil
	}
	decoder, err := decoderFor(cs)
	if err != nil {
		return "", err
	}
	out, _, err := transform.Bytes(decoder, data)
	if err != nil {
		return "", fmt.Errorf("charset: decode %s: %w", cs, err)
	}
	return string(out), nil
}

// Encode converts UTF-8 text to the given encoding. With Auto, CP850 is used,
// because that is the DATANORM default.
func Encode(text string, cs Charset) ([]byte, error) {
	if cs == Auto {
		cs = CP850
	}
	if cs == UTF8 {
		return []byte(text), nil
	}
	encoder, err := encoderFor(cs)
	if err != nil {
		return nil, err
	}
	out, _, err := transform.Bytes(encoder, []byte(text))
	if err != nil {
		return nil, fmt.Errorf("charset: encode %s: %w", cs, err)
	}
	return out, nil
}

// NewReader returns a reader that produces UTF-8. With Auto, it peeks at the
// input to detect the encoding. The peeked bytes are not lost.
func NewReader(r io.Reader, cs Charset) (io.Reader, error) {
	buffered := bufio.NewReaderSize(r, sampleSize)
	if cs == Auto {
		sample, _ := buffered.Peek(sampleSize)
		cs = Detect(sample)
	}
	if cs == UTF8 {
		if bom, _ := buffered.Peek(len(utf8BOM)); bytes.Equal(bom, utf8BOM) {
			_, _ = buffered.Discard(len(utf8BOM))
		}
		return buffered, nil
	}
	decoder, err := decoderFor(cs)
	if err != nil {
		return nil, err
	}
	return transform.NewReader(buffered, decoder), nil
}

// NewWriter returns a writer that accepts UTF-8 and writes the given encoding.
// With Auto, CP850 is used.
func NewWriter(w io.Writer, cs Charset) (io.Writer, error) {
	if cs == Auto {
		cs = CP850
	}
	if cs == UTF8 {
		return w, nil
	}
	encoder, err := encoderFor(cs)
	if err != nil {
		return nil, err
	}
	return transform.NewWriter(w, encoder), nil
}

// Detect guesses the encoding of a byte sample.
//
// Pure ASCII is reported as CP850 because that is the DATANORM default and the
// choice cannot change the decoded text. CP437 and CP850 usually decode German
// umlauts identically; CP850 wins ties because the standard names it.
func Detect(sample []byte) Charset {
	if len(sample) == 0 {
		return CP850
	}
	if bytes.HasPrefix(sample, utf8BOM) {
		return UTF8
	}
	if utf8.Valid(sample) && hasNonASCII(sample) {
		return UTF8
	}
	if !hasNonASCII(sample) {
		return CP850
	}

	candidates := []Charset{CP850, CP437, Windows1252, ISO885915, ISO88591}
	best := CP850
	bestScore := -1 << 30
	for _, candidate := range candidates {
		decoded, err := Decode(sample, candidate)
		if err != nil {
			continue
		}
		score := scoreText(decoded)
		if score > bestScore {
			best, bestScore = candidate, score
		}
	}
	if bestScore <= 0 {
		return CP850
	}
	return best
}

func hasNonASCII(data []byte) bool {
	for _, b := range data {
		if b >= 0x80 {
			return true
		}
	}
	return false
}

func scoreText(text string) int {
	score := 0
	lower := strings.ToLower(text)
	for _, word := range []string{
		" der ", " die ", " das ", " und ", " für ", " mit ", " von ",
		" den ", " ein ", " eine ", " ist ", " im ", " zu ", " auf ",
		" oder ", " nicht ", " preis ", " stück ", " artikel ", " menge ",
		" liefer", " gmbh", " katalog ", " bestell",
	} {
		score += 12 * strings.Count(lower, word)
	}
	for _, bad := range []string{"Ã¤", "Ã¶", "Ã¼", "Ã„", "Ã–", "Ãœ", "ÃŸ"} {
		score -= 20 * strings.Count(text, bad)
	}

	for _, r := range text {
		switch {
		case r == utf8.RuneError:
			score -= 50
		case r == '\r' || r == '\n' || r == '\t':
			// Neutral.
		case r < 0x20 || r == 0x7F:
			score -= 20
		case r >= 0x2500 && r <= 0x259F: // box drawing, block elements
			score -= 15
		case r >= 0x0370 && r <= 0x03FF: // Greek letters; CP437 mis-decodes
			score -= 8
		case r == 'ä' || r == 'ö' || r == 'ü' || r == 'Ä' || r == 'Ö' || r == 'Ü' || r == 'ß':
			score += 8
		case r == '€':
			score += 10
		case r == '¢' || r == '£' || r == '¤' || r == '¥' || r == '₧' || r == '⌐':
			score -= 6
		case r == '\u00AD': // soft hyphen
			score -= 6
		case r >= 0x80 && r <= 0x024F: // Latin-1 supplement / extended
			score += 2
		case r >= 0x20 && r <= 0x7E:
			score += 1
		default:
			score -= 2
		}
	}
	return score
}

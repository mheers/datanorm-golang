package productdata

import "strings"

// TextKind describes the role of a text.
type TextKind string

// Text kinds used across the supported formats.
const (
	TextShort      TextKind = "short"
	TextLong       TextKind = "long"
	TextDimension  TextKind = "dimension"
	TextMarketing  TextKind = "marketing"
	TextAdditional TextKind = "additional"
	TextTender     TextKind = "tender"
	TextNote       TextKind = "note"
)

// Text is a multi-line text with a kind and an optional language.
//
// DATANORM has hard 40-character lines; IDS Connect and OMD have flowing text.
// The model preserves the lines exactly as delivered. Use Joined to obtain a
// single string.
type Text struct {
	Kind     TextKind `json:"kind"`
	Language string   `json:"language,omitempty"`
	Lines    []string `json:"lines"`
}

// NewText returns a text with the given kind and lines.
func NewText(kind TextKind, lines ...string) Text {
	return Text{Kind: kind, Lines: append([]string(nil), lines...)}
}

// Joined joins the lines with the given separator.
func (t Text) Joined(separator string) string {
	return strings.Join(t.Lines, separator)
}

// IsEmpty reports whether every line is empty or whitespace.
func (t Text) IsEmpty() bool {
	for _, line := range t.Lines {
		if strings.TrimSpace(line) != "" {
			return false
		}
	}
	return true
}

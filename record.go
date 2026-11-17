package datanorm

import (
	"strings"
)

// Record is one parsed DATANORM record.
//
// Fields returns the raw, semicolon-separated values including the record type
// at index 0. The returned slice must not be modified.
type Record interface {
	Kind() Kind
	Line() int
	Raw() string
	Fields() []string
	Field(position int) string
}

type base struct {
	kind   Kind
	line   int
	raw    string
	fields []string
}

func newBase(kind Kind, line int, raw string, fields []string) base {
	return base{kind: kind, line: line, raw: raw, fields: fields}
}

// Kind returns the record type.
func (b base) Kind() Kind { return b.kind }

// Line returns the 1-based physical line number.
func (b base) Line() int { return b.line }

// Raw returns the original line without line ending.
func (b base) Raw() string { return b.raw }

// Fields returns the raw fields. The slice must not be modified.
func (b base) Fields() []string { return b.fields }

// Field returns the trimmed value at a 1-based field position, or "".
func (b base) Field(position int) string {
	if position < 1 || position > len(b.fields) {
		return ""
	}
	return strings.TrimSpace(b.fields[position-1])
}

// fieldCount returns the number of fields including the record type.
func (b base) fieldCount() int { return len(b.fields) }

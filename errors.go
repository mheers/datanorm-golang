package datanorm

import (
	"errors"
	"fmt"
)

// Sentinel errors. Use errors.Is to test for them.
var (
	// ErrNotDatanorm means the input does not look like a DATANORM file.
	ErrNotDatanorm = errors.New("datanorm: not a DATANORM file")
	// ErrMissingHeader means the first record is not a V record.
	ErrMissingHeader = errors.New("datanorm: missing V header record")
	// ErrVersion4 means the input is a DATANORM 4 file, which is not supported.
	ErrVersion4 = errors.New("datanorm: DATANORM 4 is not supported; this library reads version 5 (V;050;)")
	// ErrUnsupportedVersion means the file declares a version other than 050.
	ErrUnsupportedVersion = errors.New("datanorm: unsupported DATANORM version")
	// ErrLineTooLong means a physical line exceeds the configured maximum.
	ErrLineTooLong = errors.New("datanorm: line exceeds maximum length")
)

// ParseError adds location information to a parsing error.
type ParseError struct {
	File       string
	Line       int
	RecordType byte
	Field      int
	Value      string
	Err        error
}

// Error implements error.
func (e *ParseError) Error() string {
	location := ""
	if e.File != "" {
		location = e.File
	}
	if e.Line > 0 {
		if location != "" {
			location += ":"
		}
		location += fmt.Sprintf("%d", e.Line)
	}
	if location != "" {
		location = " at " + location
	}
	record := ""
	if e.RecordType != 0 {
		record = fmt.Sprintf(" (record %c", e.RecordType)
		if e.Field > 0 {
			record += fmt.Sprintf(", field %d", e.Field)
		}
		if e.Value != "" {
			record += fmt.Sprintf(", value %q", e.Value)
		}
		record += ")"
	}
	return fmt.Sprintf("%v%s%s", e.Err, location, record)
}

// Unwrap returns the underlying error.
func (e *ParseError) Unwrap() error { return e.Err }

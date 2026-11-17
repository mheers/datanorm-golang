package datanorm

import "strings"

// ProcessingFlag is the Verarbeitungskennzeichen of a record.
type ProcessingFlag string

// Processing flags.
const (
	FlagNew          ProcessingFlag = "N"
	FlagChange       ProcessingFlag = "A"
	FlagDelete       ProcessingFlag = "L"
	FlagRenumber     ProcessingFlag = "X"
	FlagPriceRequest ProcessingFlag = "P"
	FlagUnknown      ProcessingFlag = ""
)

// ParseProcessingFlag maps a raw value to a flag. Unknown values return
// FlagUnknown and keep the original in the record.
func ParseProcessingFlag(raw string) ProcessingFlag {
	switch ProcessingFlag(strings.ToUpper(strings.TrimSpace(raw))) {
	case FlagNew:
		return FlagNew
	case FlagChange:
		return FlagChange
	case FlagDelete:
		return FlagDelete
	case FlagRenumber:
		return FlagRenumber
	case FlagPriceRequest:
		return FlagPriceRequest
	default:
		return FlagUnknown
	}
}

// Valid reports whether f is a known processing flag.
func (f ProcessingFlag) Valid() bool {
	return f != FlagUnknown
}

// String returns the raw flag.
func (f ProcessingFlag) String() string {
	return string(f)
}

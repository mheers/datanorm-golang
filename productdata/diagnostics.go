package productdata

// Severity is the severity of a diagnostic.
type Severity string

// Diagnostic severities.
const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// Diagnostic is a recoverable problem or note found while reading or writing
// data. Fatal problems are returned as errors instead.
type Diagnostic struct {
	Severity   Severity `json:"severity"`
	Code       string   `json:"code"`
	Message    string   `json:"message"`
	File       string   `json:"file,omitempty"`
	Line       int      `json:"line,omitempty"`
	RecordType string   `json:"recordType,omitempty"`
	Field      int      `json:"field,omitempty"`
	Value      string   `json:"value,omitempty"`
}

// String renders a diagnostic in a compact, log-friendly form.
func (d Diagnostic) String() string {
	out := string(d.Severity)
	if d.File != "" {
		out += " " + d.File
	}
	if d.Line > 0 {
		out += ":" + itoa(d.Line)
	}
	if d.Code != "" {
		out += " " + d.Code
	}
	if d.Message != "" {
		out += ": " + d.Message
	}
	return out
}

// Diagnostics is an ordered collection of diagnostics.
type Diagnostics []Diagnostic

// Add appends a diagnostic with a severity, a stable code, and a message.
func (d *Diagnostics) Add(severity Severity, code, message string) {
	*d = append(*d, Diagnostic{Severity: severity, Code: code, Message: message})
}

// AddAt appends a pre-built diagnostic.
func (d *Diagnostics) AddAt(diagnostic Diagnostic) {
	*d = append(*d, diagnostic)
}

// HasErrors reports whether any diagnostic has severity error.
func (d Diagnostics) HasErrors() bool {
	return d.Count(SeverityError) > 0
}

// Count returns the number of diagnostics with the given severity.
func (d Diagnostics) Count(severity Severity) int {
	count := 0
	for _, diagnostic := range d {
		if diagnostic.Severity == severity {
			count++
		}
	}
	return count
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

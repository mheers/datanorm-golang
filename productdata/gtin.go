package productdata

import "strings"

// ValidateGTIN reports whether s is a syntactically valid GTIN-8, GTIN-12,
// GTIN-13, or GTIN-14 with a correct GS1 check digit.
func ValidateGTIN(s string) bool {
	s = strings.TrimSpace(s)
	switch len(s) {
	case 8, 12, 13, 14:
	default:
		return false
	}
	sum := 0
	weight := 3
	for i := len(s) - 2; i >= 0; i-- {
		c := s[i]
		if c < '0' || c > '9' {
			return false
		}
		sum += int(c-'0') * weight
		if weight == 3 {
			weight = 1
		} else {
			weight = 3
		}
	}
	check := (10 - sum%10) % 10
	last := s[len(s)-1]
	if last < '0' || last > '9' {
		return false
	}
	return int(last-'0') == check
}

// NormalizeGTIN trims surrounding whitespace without changing leading zeros.
func NormalizeGTIN(s string) string {
	return strings.TrimSpace(s)
}

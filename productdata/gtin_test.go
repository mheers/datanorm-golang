package productdata

import "testing"

func TestValidateGTIN(t *testing.T) {
	t.Parallel()

	valid := []string{
		"4006381333931",
		"4006381333931",
		"96385074",
		"036000291452",
		"00012345600012",
	}
	for _, gtin := range valid {
		if !ValidateGTIN(gtin) {
			t.Errorf("ValidateGTIN(%q) = false, want true", gtin)
		}
	}
	invalid := []string{
		"",
		"4006381333932",
		"123",
		"400638133393A",
		"40063813339311",
	}
	for _, gtin := range invalid {
		if ValidateGTIN(gtin) {
			t.Errorf("ValidateGTIN(%q) = true, want false", gtin)
		}
	}
}

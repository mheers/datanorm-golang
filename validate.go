package datanorm

import (
	"fmt"

	"github.com/mheers/datanorm-golang/productdata"
)

func validateFile(file *File) productdata.Diagnostics {
	var diagnostics productdata.Diagnostics

	if file.Header != nil && !file.Header.DateValid {
		diagnostics.AddAt(productdata.Diagnostic{
			Severity:   productdata.SeverityWarning,
			Code:       "dn.header-date",
			Message:    fmt.Sprintf("header date %q is not a valid YYYYMMDD date", file.Header.DateRaw),
			File:       file.Path,
			Line:       file.Header.Line(),
			RecordType: "V",
			Field:      4,
			Value:      file.Header.DateRaw,
		})
	}

	seenArticles := make(map[string]int)
	for _, record := range file.Records {
		switch typed := record.(type) {
		case *ArticleRecord:
			if typed.Number == "" {
				diagnostics.AddAt(productdata.Diagnostic{
					Severity:   productdata.SeverityError,
					Code:       "dn.article-number",
					Message:    "article number is empty",
					File:       file.Path,
					Line:       typed.Line(),
					RecordType: "A",
					Field:      3,
				})
			} else if utf8RuneCount(typed.Number) > 15 {
				diagnostics.AddAt(productdata.Diagnostic{
					Severity:   productdata.SeverityWarning,
					Code:       "dn.article-number",
					Message:    "article number is longer than 15 characters",
					File:       file.Path,
					Line:       typed.Line(),
					RecordType: "A",
					Field:      3,
					Value:      typed.Number,
				})
			}
			if previous, exists := seenArticles[typed.Number]; exists {
				diagnostics.AddAt(productdata.Diagnostic{
					Severity:   productdata.SeverityWarning,
					Code:       "dn.duplicate-article",
					Message:    fmt.Sprintf("article %q already appeared on line %d", typed.Number, previous),
					File:       file.Path,
					Line:       typed.Line(),
					RecordType: "A",
					Field:      3,
					Value:      typed.Number,
				})
			} else {
				seenArticles[typed.Number] = typed.Line()
			}
			if utf8RuneCount(typed.ShortText1) > 40 {
				diagnostics.AddAt(productdata.Diagnostic{
					Severity:   productdata.SeverityWarning,
					Code:       "dn.short-text",
					Message:    "short text 1 is longer than 40 characters",
					File:       file.Path,
					Line:       typed.Line(),
					RecordType: "A",
					Field:      4,
				})
			}
			if utf8RuneCount(typed.ShortText2) > 40 {
				diagnostics.AddAt(productdata.Diagnostic{
					Severity:   productdata.SeverityWarning,
					Code:       "dn.short-text",
					Message:    "short text 2 is longer than 40 characters",
					File:       file.Path,
					Line:       typed.Line(),
					RecordType: "A",
					Field:      5,
				})
			}
			if utf8RuneCount(typed.Unit) > 4 && !productdata.NormalizeUnit(typed.Unit).Known() {
				diagnostics.AddAt(productdata.Diagnostic{
					Severity:   productdata.SeverityWarning,
					Code:       "dn.unit",
					Message:    "quantity unit is longer than 4 characters and is not a known alias",
					File:       file.Path,
					Line:       typed.Line(),
					RecordType: "A",
					Field:      6,
					Value:      typed.Unit,
				})
			}
			if typed.TextMarker != "" && !typed.TextMarker.Valid() {
				diagnostics.AddAt(productdata.Diagnostic{
					Severity:   productdata.SeverityWarning,
					Code:       "dn.text-marker",
					Message:    "text marker is not in the range 0-6",
					File:       file.Path,
					Line:       typed.Line(),
					RecordType: "A",
					Field:      23,
					Value:      string(typed.TextMarker),
				})
			}
			if typed.fieldCount() > 30 {
				diagnostics.AddAt(productdata.Diagnostic{
					Severity:   productdata.SeverityWarning,
					Code:       "dn.field-count",
					Message:    fmt.Sprintf("article record has %d fields, expected at most 30", typed.fieldCount()),
					File:       file.Path,
					Line:       typed.Line(),
					RecordType: "A",
				})
			}
			if typed.ProcessingFlag == FlagUnknown && typed.Field(2) != "" {
				diagnostics.AddAt(productdata.Diagnostic{
					Severity:   productdata.SeverityWarning,
					Code:       "dn.processing-flag",
					Message:    "unknown processing flag",
					File:       file.Path,
					Line:       typed.Line(),
					RecordType: "A",
					Field:      2,
					Value:      typed.Field(2),
				})
			}
			hasType := typed.PriceTypeRaw != ""
			hasUnit := typed.PriceUnitRaw != ""
			hasPrice := typed.PriceRaw != ""
			anyPresent := hasType || hasUnit || hasPrice
			allPresent := hasType && hasUnit && hasPrice
			if typed.PriceTypeRaw != "9" && anyPresent && !allPresent {
				diagnostics.AddAt(productdata.Diagnostic{
					Severity:   productdata.SeverityWarning,
					Code:       "dn.price-fields",
					Message:    "price type, price unit, and price should be supplied together",
					File:       file.Path,
					Line:       typed.Line(),
					RecordType: "A",
				})
			}
			if hasType && productdata.PriceTypeFromCode(typed.PriceTypeRaw) == productdata.PriceUnknown {
				diagnostics.AddAt(productdata.Diagnostic{
					Severity:   productdata.SeverityWarning,
					Code:       "dn.price-type",
					Message:    fmt.Sprintf("unknown price type %q", typed.PriceTypeRaw),
					File:       file.Path,
					Line:       typed.Line(),
					RecordType: "A",
					Field:      7,
					Value:      typed.PriceTypeRaw,
				})
			}
		case *TextRecord:
			if typed.Key == "" {
				diagnostics.AddAt(productdata.Diagnostic{
					Severity:   productdata.SeverityWarning,
					Code:       "dn.text-key",
					Message:    "text record has no key",
					File:       file.Path,
					Line:       typed.Line(),
					RecordType: "T",
					Field:      3,
				})
			}
			if utf8RuneCount(typed.Text) > 40 {
				diagnostics.AddAt(productdata.Diagnostic{
					Severity:   productdata.SeverityWarning,
					Code:       "dn.text-length",
					Message:    "text line is longer than 40 characters",
					File:       file.Path,
					Line:       typed.Line(),
					RecordType: "T",
					Field:      6,
				})
			}
		case *DimensionRecord:
			if typed.ArticleNumber == "" {
				diagnostics.AddAt(productdata.Diagnostic{
					Severity:   productdata.SeverityWarning,
					Code:       "dn.dimension-article",
					Message:    "dimension record has no article number",
					File:       file.Path,
					Line:       typed.Line(),
					RecordType: "D",
					Field:      3,
				})
			}
			if utf8RuneCount(typed.Text) > 40 {
				diagnostics.AddAt(productdata.Diagnostic{
					Severity:   productdata.SeverityWarning,
					Code:       "dn.text-length",
					Message:    "dimension text line is longer than 40 characters",
					File:       file.Path,
					Line:       typed.Line(),
					RecordType: "D",
				})
			}
		case *EndRecord:
			if !typed.CountValid {
				diagnostics.AddAt(productdata.Diagnostic{
					Severity:   productdata.SeverityWarning,
					Code:       "dn.end-count",
					Message:    "E record count is not numeric",
					File:       file.Path,
					Line:       typed.Line(),
					RecordType: "E",
					Field:      2,
					Value:      typed.CountRaw,
				})
			}
		}
	}
	return diagnostics
}

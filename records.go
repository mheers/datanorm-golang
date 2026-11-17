package datanorm

import (
	"strconv"
	"strings"
	"time"

	"github.com/mheers/datanorm-golang/productdata"
)

// TextMarker is the DATANORM Textkennzeichen. It selects which text kinds an
// application should print.
type TextMarker string

// Text markers defined by DATANORM.
const (
	TextMarkerShortOnly          TextMarker = "0"
	TextMarkerShort1AndLong      TextMarker = "1"
	TextMarkerShort1AndDimension TextMarker = "2"
	TextMarkerLongAndDimension   TextMarker = "3"
	TextMarkerShortAndLong       TextMarker = "4"
	TextMarkerShortAndDimension  TextMarker = "5"
	TextMarkerAll                TextMarker = "6"
)

// Valid reports whether the marker is 0–6.
func (m TextMarker) Valid() bool {
	_, ok := m.Code()
	return ok
}

// Code returns the numeric marker and whether it is valid.
func (m TextMarker) Code() (int, bool) {
	n, err := strconv.Atoi(string(m))
	if err != nil || n < 0 || n > 6 {
		return 0, false
	}
	return n, true
}

// WantsShort1 reports whether short text line 1 should be printed.
func (m TextMarker) WantsShort1() bool {
	n, ok := m.Code()
	return ok && n != 3
}

// WantsShort2 reports whether short text line 2 should be printed.
func (m TextMarker) WantsShort2() bool {
	n, ok := m.Code()
	return ok && (n == 0 || n == 4 || n == 5 || n == 6)
}

// WantsLong reports whether a long text should be printed.
func (m TextMarker) WantsLong() bool {
	n, ok := m.Code()
	return ok && (n == 1 || n == 3 || n == 4 || n == 6)
}

// WantsDimension reports whether a dimension text should be printed.
func (m TextMarker) WantsDimension() bool {
	n, ok := m.Code()
	return ok && (n == 2 || n == 3 || n == 5 || n == 6)
}

// HeaderRecord is a V record.
type HeaderRecord struct {
	base

	Version      string
	DataMarker   string
	DateRaw      string
	Date         time.Time
	DateValid    bool
	Currency     string
	Description  string
	Info         []string
	DataContent  string
	Copyright    string
	Creator      string
	AddressLines []string
}

// DataType returns a human-readable name for the data marker.
func (h *HeaderRecord) DataType() string {
	switch strings.ToUpper(h.DataMarker) {
	case "A":
		return "article data"
	case "S":
		return "product groups"
	case "R":
		return "discount groups"
	case "P":
		return "price changes"
	case "J":
		return "sets"
	case "T":
		return "texts"
	default:
		return "unknown"
	}
}

// SupplierName extracts a plausible supplier name from the header. DATANORM
// does not specify a dedicated field consistently; this is a best effort.
func (h *HeaderRecord) SupplierName() string {
	for _, candidate := range []string{h.Description, h.Creator, h.DataContent, h.Copyright} {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		lower := strings.ToLower(candidate)
		for _, suffix := range []string{" gmbh", " ag", " kg", " se", " e.k.", " co.", " & co"} {
			if strings.Contains(lower, suffix) {
				return candidate
			}
		}
	}
	return ""
}

// ArticleRecord is an A record. The 30 fields are documented in SPEC.md §6.2.
type ArticleRecord struct {
	base

	ProcessingFlag                  ProcessingFlag
	Number                          string
	ShortText1                      string
	ShortText2                      string
	Unit                            string
	PriceTypeRaw                    string
	PriceUnitRaw                    string
	PriceRaw                        string
	DiscountGroup                   string
	MainGroup                       string
	ProductGroup                    string
	Matchcode                       string
	AlternativeArticleNumberCreator string
	AlternativeArticleNumber        string
	ManufacturerNumberCreator       string
	ManufacturerNumber              string
	ManufacturerType                string
	GTIN                            string
	MediaReference                  string
	MinOrderQuantityRaw             string
	CatalogPage                     string
	TextMarker                      TextMarker
	LongTextKey                     string
	StorageFlag                     string
	ReferenceNumberCreator          string
	ReferenceNumber                 string
	VATMarker                       string
	CostType                        string
	PackagingQuantity               string
}

// PriceType returns the parsed price type.
func (a *ArticleRecord) PriceType() string {
	return strings.TrimSpace(a.PriceTypeRaw)
}

// PricePer returns the price unit as an integer. Zero or invalid values return
// 0 and false.
func (a *ArticleRecord) PricePer() (int64, bool) {
	raw := strings.TrimSpace(a.PriceUnitRaw)
	if raw == "" {
		return 0, false
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0, false
	}
	return value, true
}

// Amount parses the article price in the given currency.
func (a *ArticleRecord) Amount(currency productdata.Currency) (productdata.Amount, error) {
	return ParsePrice(a.PriceRaw, currency)
}

// ChangeRecord is a B record.
type ChangeRecord struct {
	base

	ProcessingFlag   ProcessingFlag
	ArticleNumber    string
	NewArticleNumber string
	Successor        string
	DateRaw          string
	Extra            []string
}

// TextRecord is a T record.
type TextRecord struct {
	base

	ProcessingFlag ProcessingFlag
	Key            string
	TextType       string
	LineNumberRaw  string
	LineNumber     int
	Text           string
}

// DimensionRecord is a D record.
type DimensionRecord struct {
	base

	ProcessingFlag ProcessingFlag
	ArticleNumber  string
	LineNumberRaw  string
	LineNumber     int
	Type           string
	FreeText       string
	Text           string
	Key            string
}

// IsInsertion reports whether the record inserts a T block.
func (d *DimensionRecord) IsInsertion() bool {
	return strings.EqualFold(strings.TrimSpace(d.Type), "T")
}

// EndRecord is an E record.
type EndRecord struct {
	base

	CountRaw   string
	Count      int
	CountValid bool
	Creator    string
}

// PriceRecord is a P record.
type PriceRecord struct {
	base

	ProcessingFlag ProcessingFlag
	ArticleNumber  string
	PriceTypeRaw   string
	PriceUnitRaw   string
	PriceRaw       string
	DiscountGroup  string
	Extra          []string
}

// Amount returns the parsed amount.
func (p *PriceRecord) Amount(currency productdata.Currency) (productdata.Amount, error) {
	return ParsePrice(p.PriceRaw, currency)
}

// GroupRecord is an S record.
type GroupRecord struct {
	base

	MainGroup            string
	ProductGroup         string
	Description          string
	MainGroupDescription string
}

// DiscountRecord is an R record.
type DiscountRecord struct {
	base

	Group       string
	KindRaw     string
	ValueRaw    string
	Description string
}

// MediaRecord is a G record. The exact field order is not publicly specified;
// the typed fields are heuristics and Heuristic is always true.
type MediaRecord struct {
	base

	Heuristic       bool
	ArticleNumber   string
	ReferenceNumber string
	MediaTypeRaw    string
	Description     string
	Filename        string
}

// SurchargeRecord is a Z record. The version 5 layout is a family of
// sub-types; the typed fields follow the version 4 staffel/surcharge layout and
// are therefore heuristic. All fields remain available through Record.Fields.
type SurchargeRecord struct {
	base

	ProcessingFlag   ProcessingFlag
	ArticleNumber    string
	SetNumberRaw     string
	ProcessingMarker string
	BasisMarker      string
	BasisDescription string
	PriceTypeRaw     string
	PriceRaw         string
	FromRaw          string
	ToRaw            string
	Extra            []string
}

// SetRecord is a J record.
type SetRecord struct {
	base

	SetArticleNumber       string
	ComponentArticleNumber string
	SubTypeRaw             string
}

// ServiceRecord is a C record.
type ServiceRecord struct {
	base

	ArticleNumber string
	SubTypeRaw    string
}

// ControlRecord is a K record.
type ControlRecord struct {
	base

	CustomerNumber string
}

// UnknownRecord preserves a record with an unknown type.
type UnknownRecord struct {
	base

	RawKind byte
}

// Kind returns the raw first byte.
func (u *UnknownRecord) Kind() Kind { return Kind(u.RawKind) }

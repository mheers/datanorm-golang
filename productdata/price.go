package productdata

import (
	"strings"
	"time"
)

// PriceType describes what a price means commercially. It is not the same as a
// VAT rate and it is not the same as a price unit.
type PriceType string

// Price types used by DATANORM, IDS Connect, and Open Masterdata.
const (
	PriceList              PriceType = "list"
	PriceNet               PriceType = "net"
	PriceRoute             PriceType = "route"
	PriceRecommendedRetail PriceType = "recommended-retail"
	PriceGuide             PriceType = "guide"
	PriceOnRequest         PriceType = "on-request"
	PriceUnknown           PriceType = "unknown"
)

var priceTypeByCode = map[string]PriceType{
	"1": PriceList,
	"2": PriceNet,
	"3": PriceRoute,
	"4": PriceRecommendedRetail,
	"5": PriceGuide,
	"9": PriceOnRequest,

	"list":                      PriceList,
	"listenpreis":               PriceList,
	"brutto":                    PriceList,
	"net":                       PriceNet,
	"nettopreis":                PriceNet,
	"netto":                     PriceNet,
	"route":                     PriceRoute,
	"streckenpreis":             PriceRoute,
	"recommended-retail":        PriceRecommendedRetail,
	"recommendedretail":         PriceRecommendedRetail,
	"empfohlener verkaufspreis": PriceRecommendedRetail,
	"uvp":                       PriceRecommendedRetail,
	"rrp":                       PriceRecommendedRetail,
	"guide":                     PriceGuide,
	"richtpreis":                PriceGuide,
	"on-request":                PriceOnRequest,
	"onrequest":                 PriceOnRequest,
	"preis auf anfrage":         PriceOnRequest,
	"anfrage":                   PriceOnRequest,
}

var priceTypeToCode = map[PriceType]string{
	PriceList:              "1",
	PriceNet:               "2",
	PriceRoute:             "3",
	PriceRecommendedRetail: "4",
	PriceGuide:             "5",
	PriceOnRequest:         "9",
	PriceUnknown:           "",
}

// PriceTypeFromCode maps a DATANORM price-type code or a textual name to a
// PriceType. Unknown values return PriceUnknown.
func PriceTypeFromCode(code string) PriceType {
	key := strings.ToLower(strings.TrimSpace(code))
	if value, ok := priceTypeByCode[key]; ok {
		return value
	}
	return PriceUnknown
}

// Code returns the DATANORM price-type code, or an empty string.
func (p PriceType) Code() string {
	return priceTypeToCode[p]
}

// String returns a human-readable name.
func (p PriceType) String() string {
	switch p {
	case PriceList:
		return "list price"
	case PriceNet:
		return "net price"
	case PriceRoute:
		return "route price"
	case PriceRecommendedRetail:
		return "recommended retail price"
	case PriceGuide:
		return "guide price"
	case PriceOnRequest:
		return "price on request"
	default:
		return "unknown price"
	}
}

// PriceTier is a quantity-dependent price.
type PriceTier struct {
	MinQuantity Quantity `json:"minQuantity"`
	Amount      Amount   `json:"amount"`
}

// Price is one price for one article. An article may have several prices of
// different types at the same time.
type Price struct {
	Type        PriceType   `json:"type"`
	Amount      Amount      `json:"amount"`
	Per         *Quantity   `json:"per,omitempty"`
	MinQuantity *Quantity   `json:"minQuantity,omitempty"`
	Tiers       []PriceTier `json:"tiers,omitempty"`
	ValidFrom   *time.Time  `json:"validFrom,omitempty"`
	ValidTo     *time.Time  `json:"validTo,omitempty"`
}

// PerQuantity returns the price base quantity. A nil Per means one base unit.
func (p Price) PerQuantity() Quantity {
	if p.Per == nil {
		return Quantity{Value: NewDecimal(1, 0)}
	}
	return *p.Per
}

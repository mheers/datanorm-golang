package productdata

import "strings"

// ArticleStatus describes the lifecycle state of an article as delivered.
type ArticleStatus string

// Article statuses.
const (
	StatusActive       ArticleStatus = "active"
	StatusNew          ArticleStatus = "new"
	StatusChanged      ArticleStatus = "changed"
	StatusDiscontinued ArticleStatus = "discontinued"
	StatusDeleted      ArticleStatus = "deleted"
)

// ArticleRef identifies an article without embedding its full data.
type ArticleRef struct {
	SupplierArticleNumber     string `json:"supplierArticleNumber,omitempty"`
	Manufacturer              string `json:"manufacturer,omitempty"`
	ManufacturerArticleNumber string `json:"manufacturerArticleNumber,omitempty"`
	GTIN                      string `json:"gtin,omitempty"`
}

// AssociationKind describes a relation between articles.
type AssociationKind string

// Association kinds.
const (
	AssociationAlternative AssociationKind = "alternative"
	AssociationSuccessor   AssociationKind = "successor"
	AssociationAccessory   AssociationKind = "accessory"
	AssociationSparePart   AssociationKind = "spare-part"
	AssociationSimilar     AssociationKind = "similar"
	AssociationComponent   AssociationKind = "component"
)

// Association links one article to another.
type Association struct {
	Kind     AssociationKind `json:"kind"`
	Article  ArticleRef      `json:"article"`
	Quantity *Quantity       `json:"quantity,omitempty"`
	Note     string          `json:"note,omitempty"`
}

// Logistics groups physical and customs data.
type Logistics struct {
	Weight              *Quantity `json:"weight,omitempty"`
	Length              *Quantity `json:"length,omitempty"`
	Width               *Quantity `json:"width,omitempty"`
	Height              *Quantity `json:"height,omitempty"`
	PackagingQuantity   *Quantity `json:"packagingQuantity,omitempty"`
	CountryOfOrigin     string    `json:"countryOfOrigin,omitempty"`
	CustomsTariffNumber string    `json:"customsTariffNumber,omitempty"`
	StorageFlag         string    `json:"storageFlag,omitempty"`
}

// Source records where an article came from. It is useful for diagnostics and
// for explaining a value to a user.
type Source struct {
	Format     string `json:"format,omitempty"`
	Version    string `json:"version,omitempty"`
	File       string `json:"file,omitempty"`
	Line       int    `json:"line,omitempty"`
	RecordType string `json:"recordType,omitempty"`
}

// Article is the canonical product record.
//
// Prices and texts are slices because a single article can carry a list price,
// a net price, a recommended retail price, short texts, long texts, and
// marketing texts at the same time.
type Article struct {
	SupplierArticleNumber     string            `json:"supplierArticleNumber"`
	Manufacturer              string            `json:"manufacturer,omitempty"`
	ManufacturerArticleNumber string            `json:"manufacturerArticleNumber,omitempty"`
	Type                      string            `json:"type,omitempty"`
	Matchcode                 string            `json:"matchcode,omitempty"`
	GTIN                      string            `json:"gtin,omitempty"`
	Unit                      Unit              `json:"unit"`
	Prices                    []Price           `json:"prices,omitempty"`
	Texts                     []Text            `json:"texts,omitempty"`
	Groups                    []GroupRef        `json:"groups,omitempty"`
	MinOrderQuantity          *Quantity         `json:"minOrderQuantity,omitempty"`
	PackagingQuantity         *Quantity         `json:"packagingQuantity,omitempty"`
	Status                    ArticleStatus     `json:"status,omitempty"`
	Replacement               *ArticleRef       `json:"replacement,omitempty"`
	Media                     []Media           `json:"media,omitempty"`
	Features                  []Feature         `json:"features,omitempty"`
	Classifications           []Classification  `json:"classifications,omitempty"`
	Logistics                 Logistics         `json:"logistics,omitempty"`
	Associations              []Association     `json:"associations,omitempty"`
	Extra                     map[string]string `json:"extra,omitempty"`
	Source                    Source            `json:"source,omitempty"`
	Raw                       any               `json:"-"`
}

// AddText replaces a text of the same kind and language, or appends it.
func (a *Article) AddText(text Text) {
	for i := range a.Texts {
		if a.Texts[i].Kind == text.Kind && a.Texts[i].Language == text.Language {
			a.Texts[i] = text
			return
		}
	}
	a.Texts = append(a.Texts, text)
}

// Text returns the joined lines of the first text with the given kind. Long and
// dimension texts are joined with newlines; short texts are joined with spaces.
func (a Article) Text(kind TextKind) string {
	for _, text := range a.Texts {
		if text.Kind != kind {
			continue
		}
		if kind == TextShort {
			return text.Joined(" ")
		}
		return text.Joined("\n")
	}
	return ""
}

// ShortText returns the short text lines joined with a space.
func (a Article) ShortText() string {
	return a.Text(TextShort)
}

// LongText returns the long text, or the dimension text if no long text exists.
func (a Article) LongText() string {
	if text := a.Text(TextLong); text != "" {
		return text
	}
	return a.Text(TextDimension)
}

// Price returns the first price of the given type.
func (a Article) Price(kind PriceType) (Price, bool) {
	for _, price := range a.Prices {
		if price.Type == kind {
			return price, true
		}
	}
	return Price{}, false
}

// Group returns the first group reference of the given level.
func (a Article) Group(level GroupLevel) (GroupRef, bool) {
	for _, group := range a.Groups {
		if group.Level == level {
			return group, true
		}
	}
	return GroupRef{}, false
}

// DisplayName returns a human-readable name: the short text, the long text, the
// manufacturer type, or the article number, in that order.
func (a Article) DisplayName() string {
	if text := strings.TrimSpace(a.ShortText()); text != "" {
		return text
	}
	if text := strings.TrimSpace(a.Text(TextLong)); text != "" {
		return text
	}
	if a.Type != "" {
		return a.Type
	}
	return a.SupplierArticleNumber
}

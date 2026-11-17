package productdata

import "time"

// Supplier is the sender of a delivery.
type Supplier struct {
	Name       string `json:"name,omitempty"`
	Number     string `json:"number,omitempty"`
	Street     string `json:"street,omitempty"`
	PostalCode string `json:"postalCode,omitempty"`
	City       string `json:"city,omitempty"`
	Country    string `json:"country,omitempty"`
	Website    string `json:"website,omitempty"`
	Email      string `json:"email,omitempty"`
	Phone      string `json:"phone,omitempty"`
	Contact    string `json:"contact,omitempty"`
}

// Header describes a catalogue or message.
type Header struct {
	Format        string     `json:"format"`
	FormatVersion string     `json:"formatVersion,omitempty"`
	Encoding      string     `json:"encoding,omitempty"`
	Description   string     `json:"description,omitempty"`
	Info          []string   `json:"info,omitempty"`
	CreatedAt     *time.Time `json:"createdAt,omitempty"`
	PriceDate     *time.Time `json:"priceDate,omitempty"`
	Currency      Currency   `json:"currency,omitempty"`
	Supplier      Supplier   `json:"supplier,omitempty"`
	RecordCount   int        `json:"recordCount,omitempty"`
	Source        string     `json:"source,omitempty"`
}

// Set is a set or bill of materials.
type Set struct {
	ID          string    `json:"id"`
	Description string    `json:"description,omitempty"`
	Items       []SetItem `json:"items,omitempty"`
}

// SetItem is one component of a set.
type SetItem struct {
	ArticleNumber string   `json:"articleNumber"`
	Quantity      Quantity `json:"quantity"`
	Optional      bool     `json:"optional,omitempty"`
	Kind          string   `json:"kind,omitempty"`
	Text          string   `json:"text,omitempty"`
}

// Surcharge is a raw-material or special-cost surcharge.
type Surcharge struct {
	Type          string   `json:"type,omitempty"`
	ArticleNumber string   `json:"articleNumber,omitempty"`
	Material      string   `json:"material,omitempty"`
	Amount        *Amount  `json:"amount,omitempty"`
	Percent       *Decimal `json:"percent,omitempty"`
	Note          string   `json:"note,omitempty"`
}

// Catalog is the canonical, format-neutral result of reading a batch of
// product data. A catalogue may contain articles, groups, discount groups,
// sets, and surcharges.
type Catalog struct {
	Header         Header          `json:"header"`
	Articles       []Article       `json:"articles,omitempty"`
	Groups         []Group         `json:"groups,omitempty"`
	DiscountGroups []DiscountGroup `json:"discountGroups,omitempty"`
	Sets           []Set           `json:"sets,omitempty"`
	Surcharges     []Surcharge     `json:"surcharges,omitempty"`
	Diagnostics    Diagnostics     `json:"diagnostics,omitempty"`
}

// AddArticle appends an article.
func (c *Catalog) AddArticle(article Article) {
	c.Articles = append(c.Articles, article)
}

// AddGroup appends a group.
func (c *Catalog) AddGroup(group Group) {
	c.Groups = append(c.Groups, group)
}

// AddDiscountGroup appends a discount group.
func (c *Catalog) AddDiscountGroup(group DiscountGroup) {
	c.DiscountGroups = append(c.DiscountGroups, group)
}

// FindArticle returns the first article with the given supplier article number.
func (c *Catalog) FindArticle(number string) (*Article, bool) {
	for i := range c.Articles {
		if c.Articles[i].SupplierArticleNumber == number {
			return &c.Articles[i], true
		}
	}
	return nil, false
}

// FindGroup returns the first group with the given level and ID.
func (c *Catalog) FindGroup(level GroupLevel, id string) (Group, bool) {
	for _, group := range c.Groups {
		if group.Level == level && group.ID == id {
			return group, true
		}
	}
	return Group{}, false
}

// FindDiscountGroup returns the discount group with the given ID.
func (c *Catalog) FindDiscountGroup(id string) (DiscountGroup, bool) {
	for _, group := range c.DiscountGroups {
		if group.ID == id {
			return group, true
		}
	}
	return DiscountGroup{}, false
}

// ArticleCount returns the number of articles.
func (c *Catalog) ArticleCount() int {
	return len(c.Articles)
}

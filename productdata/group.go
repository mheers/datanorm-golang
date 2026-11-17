package productdata

// GroupLevel describes the role of a product or commercial group.
type GroupLevel string

// Group levels.
const (
	GroupMain     GroupLevel = "main"
	GroupProduct  GroupLevel = "product"
	GroupDiscount GroupLevel = "discount"
	GroupCategory GroupLevel = "category"
	GroupCustom   GroupLevel = "custom"
)

// GroupRef is a reference to a group from an article.
type GroupRef struct {
	Level GroupLevel `json:"level"`
	ID    string     `json:"id"`
	Name  string     `json:"name,omitempty"`
}

// Group is a product group or category.
type Group struct {
	Level    GroupLevel `json:"level"`
	ID       string     `json:"id"`
	Name     string     `json:"name,omitempty"`
	ParentID string     `json:"parentId,omitempty"`
}

// DiscountKind describes how a discount value is interpreted.
type DiscountKind string

// Discount kinds.
const (
	DiscountPercent   DiscountKind = "percent"
	DiscountFactor    DiscountKind = "factor"
	DiscountSurcharge DiscountKind = "surcharge"
)

// Discount is one rate in a discount group. Multiple discounts in a group are
// applied in order and multiplicatively, as DATANORM requires.
type Discount struct {
	Kind        DiscountKind `json:"kind"`
	Value       Decimal      `json:"value"`
	Description string       `json:"description,omitempty"`
}

// DiscountGroup is a commercial group of articles with shared conditions.
type DiscountGroup struct {
	ID        string     `json:"id"`
	Name      string     `json:"name,omitempty"`
	Discounts []Discount `json:"discounts,omitempty"`
}

// Describe returns the name if present, otherwise the ID.
func (g DiscountGroup) Describe() string {
	if g.Name != "" {
		return g.Name
	}
	return g.ID
}

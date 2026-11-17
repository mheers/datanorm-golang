package productdata

import "strings"

// UnitKind is the physical or commercial kind of a unit.
type UnitKind string

// Unit kinds.
const (
	UnitPiece     UnitKind = "piece"
	UnitPair      UnitKind = "pair"
	UnitLength    UnitKind = "length"
	UnitArea      UnitKind = "area"
	UnitVolume    UnitKind = "volume"
	UnitMass      UnitKind = "mass"
	UnitTime      UnitKind = "time"
	UnitPackaging UnitKind = "packaging"
	UnitSet       UnitKind = "set"
	UnitOther     UnitKind = "other"
)

// Unit is a quantity unit. Code is the normalized canonical code when the unit
// is known, otherwise the original value. Name is an optional human-readable
// name. Kind is a coarse classification.
type Unit struct {
	Code string   `json:"code"`
	Name string   `json:"name,omitempty"`
	Kind UnitKind `json:"kind"`
}

type unitMapping struct {
	code string
	name string
	kind UnitKind
}

var unitMappings = map[string]unitMapping{
	// Piece.
	"pce":    {"PCE", "piece", UnitPiece},
	"c62":    {"PCE", "piece", UnitPiece},
	"h87":    {"PCE", "piece", UnitPiece},
	"stk":    {"PCE", "piece", UnitPiece},
	"stck":   {"PCE", "piece", UnitPiece},
	"st":     {"PCE", "piece", UnitPiece},
	"stück":  {"PCE", "piece", UnitPiece},
	"stueck": {"PCE", "piece", UnitPiece},
	"stuek":  {"PCE", "piece", UnitPiece},
	"ea":     {"PCE", "piece", UnitPiece},
	"each":   {"PCE", "piece", UnitPiece},
	"piece":  {"PCE", "piece", UnitPiece},

	// Pair.
	"pr":   {"PR", "pair", UnitPair},
	"prs":  {"PR", "pair", UnitPair},
	"paar": {"PR", "pair", UnitPair},
	"pa":   {"PR", "pair", UnitPair},
	"pair": {"PR", "pair", UnitPair},

	// Length.
	"mtr":   {"MTR", "metre", UnitLength},
	"m":     {"MTR", "metre", UnitLength},
	"lfm":   {"MTR", "running metre", UnitLength},
	"lfdm":  {"MTR", "running metre", UnitLength},
	"lfd.m": {"MTR", "running metre", UnitLength},
	"meter": {"MTR", "metre", UnitLength},
	"metre": {"MTR", "metre", UnitLength},
	"mmt":   {"MMT", "millimetre", UnitLength},
	"mm":    {"MMT", "millimetre", UnitLength},
	"cmt":   {"CMT", "centimetre", UnitLength},
	"cm":    {"CMT", "centimetre", UnitLength},
	"km":    {"KMT", "kilometre", UnitLength},

	// Area.
	"mtk":          {"MTK", "square metre", UnitArea},
	"m2":           {"MTK", "square metre", UnitArea},
	"m²":           {"MTK", "square metre", UnitArea},
	"qm":           {"MTK", "square metre", UnitArea},
	"quadratmeter": {"MTK", "square metre", UnitArea},
	"sqm":          {"MTK", "square metre", UnitArea},

	// Volume.
	"mtq":        {"MTQ", "cubic metre", UnitVolume},
	"m3":         {"MTQ", "cubic metre", UnitVolume},
	"m³":         {"MTQ", "cubic metre", UnitVolume},
	"cbm":        {"MTQ", "cubic metre", UnitVolume},
	"kubikmeter": {"MTQ", "cubic metre", UnitVolume},
	"ltr":        {"LTR", "litre", UnitVolume},
	"l":          {"LTR", "litre", UnitVolume},
	"liter":      {"LTR", "litre", UnitVolume},
	"mlt":        {"MLT", "millilitre", UnitVolume},
	"ml":         {"MLT", "millilitre", UnitVolume},

	// Mass.
	"kgm":       {"KGM", "kilogram", UnitMass},
	"kg":        {"KGM", "kilogram", UnitMass},
	"kilogramm": {"KGM", "kilogram", UnitMass},
	"grm":       {"GRM", "gram", UnitMass},
	"g":         {"GRM", "gram", UnitMass},
	"gramm":     {"GRM", "gram", UnitMass},
	"tne":       {"TNE", "tonne", UnitMass},
	"t":         {"TNE", "tonne", UnitMass},
	"tonne":     {"TNE", "tonne", UnitMass},

	// Time.
	"hur":     {"HUR", "hour", UnitTime},
	"std":     {"HUR", "hour", UnitTime},
	"stunde":  {"HUR", "hour", UnitTime},
	"stunden": {"HUR", "hour", UnitTime},
	"h":       {"HUR", "hour", UnitTime},
	"hour":    {"HUR", "hour", UnitTime},
	"min":     {"MIN", "minute", UnitTime},
	"minute":  {"MIN", "minute", UnitTime},
	"minuten": {"MIN", "minute", UnitTime},

	// Packaging.
	"pk":      {"PK", "package", UnitPackaging},
	"pak":     {"PK", "package", UnitPackaging},
	"pck":     {"PK", "package", UnitPackaging},
	"pkg":     {"PK", "package", UnitPackaging},
	"pack":    {"PK", "package", UnitPackaging},
	"packung": {"PK", "package", UnitPackaging},
	"package": {"PK", "package", UnitPackaging},
	"karton":  {"PK", "carton", UnitPackaging},
	"ct":      {"PK", "carton", UnitPackaging},
	"box":     {"PK", "box", UnitPackaging},

	// Set.
	"set":  {"SET", "set", UnitSet},
	"satz": {"SET", "set", UnitSet},

	// Rolls are a packaging-like unit.
	"rol":   {"ROL", "roll", UnitPackaging},
	"rolle": {"ROL", "roll", UnitPackaging},
	"roll":  {"ROL", "roll", UnitPackaging},
}

// NormalizeUnit maps a unit code or a common German abbreviation to a canonical
// unit. Unknown units are returned with their original code and UnitOther.
func NormalizeUnit(code string) Unit {
	original := strings.TrimSpace(code)
	if original == "" {
		return Unit{Kind: UnitOther}
	}
	key := strings.ToLower(original)
	key = strings.TrimSuffix(key, ".")
	key = strings.ReplaceAll(key, " ", "")
	if mapping, ok := unitMappings[key]; ok {
		return Unit{Code: mapping.code, Name: mapping.name, Kind: mapping.kind}
	}
	return Unit{Code: original, Kind: UnitOther}
}

// Known reports whether the unit code is recognized.
func (u Unit) Known() bool {
	return u.Kind != "" && u.Kind != UnitOther
}

// String returns the unit code.
func (u Unit) String() string {
	return u.Code
}

// Quantity is a decimal value with a unit.
type Quantity struct {
	Value Decimal `json:"value"`
	Unit  Unit    `json:"unit"`
}

// NewQuantity returns a quantity.
func NewQuantity(value Decimal, unit Unit) Quantity {
	return Quantity{Value: value, Unit: unit}
}

// String returns "value code", for example "100 PCE".
func (q Quantity) String() string {
	if q.Unit.Code == "" {
		return q.Value.String()
	}
	return q.Value.String() + " " + q.Unit.Code
}

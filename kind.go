package datanorm

// Kind is a DATANORM record type (Satzart).
type Kind byte

// Record types defined by DATANORM 5.
const (
	KindHeader    Kind = 'V'
	KindArticle   Kind = 'A'
	KindChange    Kind = 'B'
	KindText      Kind = 'T'
	KindDimension Kind = 'D'
	KindEnd       Kind = 'E'
	KindPrice     Kind = 'P'
	KindGroup     Kind = 'S'
	KindDiscount  Kind = 'R'
	KindSurcharge Kind = 'Z'
	KindSet       Kind = 'J'
	KindService   Kind = 'C'
	KindMedia     Kind = 'G'
	KindControl   Kind = 'K'
	KindUnknown   Kind = 0
)

// String returns the one-character record type.
func (k Kind) String() string {
	if k == KindUnknown {
		return "?"
	}
	return string(byte(k))
}

// Name returns the English record name.
func (k Kind) Name() string {
	switch k {
	case KindHeader:
		return "header"
	case KindArticle:
		return "article"
	case KindChange:
		return "change"
	case KindText:
		return "long text"
	case KindDimension:
		return "dimension text"
	case KindEnd:
		return "end"
	case KindPrice:
		return "price change"
	case KindGroup:
		return "product group"
	case KindDiscount:
		return "discount group"
	case KindSurcharge:
		return "surcharge/tier"
	case KindSet:
		return "set"
	case KindService:
		return "service"
	case KindMedia:
		return "media reference"
	case KindControl:
		return "control"
	default:
		return "unknown"
	}
}

// Known reports whether k is a known DATANORM 5 record type.
func (k Kind) Known() bool {
	switch k {
	case KindHeader, KindArticle, KindChange, KindText, KindDimension, KindEnd,
		KindPrice, KindGroup, KindDiscount, KindSurcharge, KindSet, KindService,
		KindMedia, KindControl:
		return true
	default:
		return false
	}
}

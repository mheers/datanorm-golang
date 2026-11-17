package datanorm

// Confidence describes how well a field position is established.
type Confidence string

// Field confidence levels.
const (
	ConfidenceSpecified Confidence = "specified"
	ConfidenceObserved  Confidence = "observed"
	ConfidenceInferred  Confidence = "inferred"
	ConfidenceUnknown   Confidence = "unknown"
)

// FieldSpec documents one field of a record.
type FieldSpec struct {
	Position   int
	Name       string
	German     string
	MaxLength  int
	Confidence Confidence
	Note       string
}

var articleFieldSpecs = []FieldSpec{
	{1, "RecordType", "Satzart", 1, ConfidenceSpecified, "Always A"},
	{2, "ProcessingFlag", "Verarbeitungskennzeichen", 1, ConfidenceSpecified, "N, A, L"},
	{3, "ArticleNumber", "Artikelnummer", 15, ConfidenceSpecified, ""},
	{4, "ShortText1", "Kurztext 1", 40, ConfidenceSpecified, ""},
	{5, "ShortText2", "Kurztext 2", 40, ConfidenceSpecified, ""},
	{6, "Unit", "Mengeneinheit", 4, ConfidenceSpecified, "ISO-style or German abbreviation"},
	{7, "PriceType", "Preiskennzeichen", 1, ConfidenceSpecified, "1, 2, 3, 4, 5, 9"},
	{8, "PriceUnit", "Preiseinheit", 6, ConfidenceSpecified, "Direct number, not the version 4 code"},
	{9, "Price", "Preis", 10, ConfidenceSpecified, "Minor units or explicit decimal"},
	{10, "DiscountGroup", "Rabattgruppe", 4, ConfidenceSpecified, "References R field 2"},
	{11, "MainGroup", "Hauptwarengruppe", 3, ConfidenceSpecified, "References S field 2"},
	{12, "ProductGroup", "Warengruppe", 10, ConfidenceSpecified, "References S field 3"},
	{13, "Matchcode", "Matchcode", 15, ConfidenceSpecified, ""},
	{14, "AlternativeArticleNumberCreator", "Kürzel Ersteller Altern.Artikelnr.", 13, ConfidenceSpecified, ""},
	{15, "AlternativeArticleNumber", "Alternativartikelnummer", 15, ConfidenceSpecified, "Observed in public files"},
	{16, "ManufacturerNumberCreator", "Kürzel Ersteller Herstellernr.", 13, ConfidenceSpecified, ""},
	{17, "ManufacturerNumber", "Herstellernummer", 35, ConfidenceSpecified, "Observed in public files"},
	{18, "ManufacturerType", "Herstellertype", 35, ConfidenceSpecified, ""},
	{19, "GTIN", "EAN / GTIN", 18, ConfidenceSpecified, "String; leading zeros significant"},
	{20, "MediaReference", "Grafikanbindungsnummer", 15, ConfidenceSpecified, "Links to G records"},
	{21, "MinOrderQuantity", "Mindestverpackungsmenge", 5, ConfidenceSpecified, ""},
	{22, "CatalogPage", "Katalogseite", 15, ConfidenceSpecified, "Observed"},
	{23, "TextMarker", "Textkennzeichen", 1, ConfidenceSpecified, "0-6"},
	{24, "LongTextKey", "Langtextschlüssel", 18, ConfidenceSpecified, "References T field 3"},
	{25, "StorageFlag", "Lagermerker", 1, ConfidenceSpecified, "Often empty or a space"},
	{26, "ReferenceNumberCreator", "Kürzel Ersteller Referenznr.", 13, ConfidenceSpecified, ""},
	{27, "ReferenceNumber", "Referenznummer", 17, ConfidenceSpecified, ""},
	{28, "VATMarker", "MwSt-Merker", 1, ConfidenceSpecified, "Meaning not publicly documented"},
	{29, "CostType", "Kostenart", 2, ConfidenceInferred, "Version 4 B field 11"},
	{30, "PackagingQuantity", "Verpackungsmenge", 15, ConfidenceInferred, "Version 4 B field 12"},
}

var headerFieldSpecs = []FieldSpec{
	{1, "RecordType", "Satzart", 1, ConfidenceSpecified, "Always V"},
	{2, "Version", "Version", 3, ConfidenceSpecified, "050"},
	{3, "DataMarker", "Datenkennzeichen", 1, ConfidenceSpecified, "A, S, R; P/J/T inferred"},
	{4, "Date", "Datum", 8, ConfidenceSpecified, "YYYYMMDD"},
	{5, "Currency", "Währung", 3, ConfidenceSpecified, "ISO 4217"},
	{6, "Info1", "Infozeile 1", 40, ConfidenceSpecified, ""},
	{7, "Info2", "Infozeile 2", 40, ConfidenceSpecified, ""},
	{8, "Info3", "Infozeile 3", 40, ConfidenceSpecified, ""},
	{9, "DataContent", "Dateninhalt", 40, ConfidenceSpecified, ""},
	{10, "Copyright", "Copyright", 40, ConfidenceSpecified, ""},
	{11, "Creator", "Datenersteller", 13, ConfidenceSpecified, ""},
	{12, "Address1", "Anschrift 1", 30, ConfidenceSpecified, ""},
	{13, "Address2", "Anschrift 2", 30, ConfidenceSpecified, ""},
	{14, "Address3", "Anschrift 3", 30, ConfidenceSpecified, ""},
	{15, "Street", "Straße", 30, ConfidenceSpecified, ""},
	{16, "Country", "Land", 3, ConfidenceSpecified, ""},
	{17, "PostalCode", "PLZ", 9, ConfidenceSpecified, ""},
	{18, "City", "Ort", 30, ConfidenceSpecified, ""},
}

var textFieldSpecs = []FieldSpec{
	{1, "RecordType", "Satzart", 1, ConfidenceSpecified, "Always T"},
	{2, "ProcessingFlag", "Verarbeitungskennzeichen", 1, ConfidenceSpecified, "N, A, L"},
	{3, "Key", "Textschlüssel", 18, ConfidenceSpecified, ""},
	{4, "TextType", "Textart", 1, ConfidenceSpecified, "1 = normal long text"},
	{5, "LineNumber", "Zeilennummer", 2, ConfidenceSpecified, "Decimal, compare numerically"},
	{6, "Text", "Textzeile", 40, ConfidenceSpecified, ""},
	{7, "Reserved", "Reserve", 0, ConfidenceUnknown, ""},
}

var dimensionFieldSpecs = []FieldSpec{
	{1, "RecordType", "Satzart", 1, ConfidenceSpecified, "Always D"},
	{2, "ProcessingFlag", "Verarbeitungskennzeichen", 1, ConfidenceSpecified, "N, A, L"},
	{3, "ArticleNumber", "Artikelnummer", 15, ConfidenceSpecified, ""},
	{4, "LineNumber", "Zeilennummer", 2, ConfidenceSpecified, ""},
	{5, "Type", "Unterkennzeichen", 1, ConfidenceSpecified, "F free text, T insert block"},
	{6, "FreeText", "Freitext", 40, ConfidenceObserved, "Vendors disagree between field 6 and 7"},
	{7, "TextOrKey", "Zeilentext / Schlüssel", 40, ConfidenceObserved, ""},
	{8, "Reserved", "Reserve", 0, ConfidenceUnknown, ""},
}

var priceFieldSpecs = []FieldSpec{
	{1, "RecordType", "Satzart", 1, ConfidenceSpecified, "Always P"},
	{2, "ArticleNumber", "Artikelnummer", 15, ConfidenceObserved, "A version 4 flag may precede it"},
	{3, "PriceType", "Preiskennzeichen", 1, ConfidenceObserved, ""},
	{4, "PriceUnit", "Preiseinheit", 6, ConfidenceObserved, ""},
	{5, "Price", "Preis", 10, ConfidenceObserved, ""},
	{6, "DiscountGroup", "Rabattgruppe", 4, ConfidenceObserved, ""},
}

var groupFieldSpecs = []FieldSpec{
	{1, "RecordType", "Satzart", 1, ConfidenceSpecified, "Always S"},
	{2, "MainGroup", "Hauptwarengruppe", 3, ConfidenceSpecified, "No processing flag"},
	{3, "ProductGroup", "Warengruppe", 10, ConfidenceSpecified, ""},
	{4, "Description", "Bezeichnung", 40, ConfidenceSpecified, ""},
	{5, "MainGroupDescription", "Bezeichnung Hauptgruppe", 40, ConfidenceObserved, ""},
}

var discountFieldSpecs = []FieldSpec{
	{1, "RecordType", "Satzart", 1, ConfidenceSpecified, "Always R"},
	{2, "Group", "Rabattgruppe", 4, ConfidenceSpecified, "No processing flag"},
	{3, "Kind", "Rabattkennzeichen", 1, ConfidenceSpecified, "1 percent, 2 factor, 3 surcharge"},
	{4, "Value", "Rabatt / Multiplikator", 15, ConfidenceSpecified, ""},
	{5, "Description", "Bezeichnung", 40, ConfidenceSpecified, ""},
}

var endFieldSpecs = []FieldSpec{
	{1, "RecordType", "Satzart", 1, ConfidenceSpecified, "Always E"},
	{2, "Count", "Satzanzahl", 10, ConfidenceSpecified, "Convention varies; see SPEC.md §6.6"},
	{3, "Creator", "Ersteller / Copyright", 40, ConfidenceSpecified, ""},
}

var changeFieldSpecs = []FieldSpec{
	{1, "RecordType", "Satzart", 1, ConfidenceSpecified, "Always B"},
	{2, "ProcessingFlag", "Verarbeitungskennzeichen", 1, ConfidenceSpecified, "L delete, X renumber, A change"},
	{3, "ArticleNumber", "Artikelnummer", 15, ConfidenceSpecified, "Old/affected number"},
	{4, "NewArticleNumber", "Neue Artikelnummer", 15, ConfidenceObserved, "For X records"},
	{5, "Successor", "Nachfolger", 15, ConfidenceObserved, "For A records"},
	{6, "Date", "Datum", 8, ConfidenceObserved, "YYYYMMDD"},
}

var mediaFieldSpecs = []FieldSpec{
	{1, "RecordType", "Satzart", 1, ConfidenceSpecified, "Always G"},
	{2, "ProcessingFlag", "Verarbeitungskennzeichen", 1, ConfidenceObserved, "May be absent"},
	{3, "ArticleNumber", "Artikelnummer", 15, ConfidenceInferred, "Heuristic"},
	{4, "ReferenceNumber", "Grafikanbindungsnummer", 15, ConfidenceInferred, "Heuristic"},
	{5, "MediaType", "Anbindungsart", 2, ConfidenceUnknown, "Heuristic"},
	{6, "Description", "Grafikbeschreibung", 40, ConfidenceInferred, "Heuristic"},
	{7, "Filename", "Grafikdateiname", 15, ConfidenceInferred, "DOS 8.3 in strict files"},
}

var surchargeFieldSpecs = []FieldSpec{
	{1, "RecordType", "Satzart", 1, ConfidenceSpecified, "Always Z"},
	{2, "ProcessingFlag", "Verarbeitungskennzeichen", 1, ConfidenceObserved, ""},
	{3, "ArticleNumber", "Artikelnummer", 15, ConfidenceInferred, "Heuristic"},
	{4, "SetNumber", "Satznummer", 15, ConfidenceUnknown, ""},
	{5, "ProcessingMarker", "Bearbeitungsmerker", 1, ConfidenceInferred, "1 tier price per NAV mapping"},
	{6, "BasisMarker", "Basismerker", 1, ConfidenceInferred, "1 delivery/purchase quantity"},
	{7, "BasisDescription", "Basisbezeichnung", 40, ConfidenceUnknown, ""},
	{8, "PriceType", "Preiskennzeichen", 1, ConfidenceInferred, ""},
	{9, "Price", "Preis", 10, ConfidenceInferred, ""},
	{10, "From", "Von", 15, ConfidenceInferred, "Tier lower bound"},
	{11, "To", "Bis", 15, ConfidenceInferred, "Tier upper bound"},
}

var controlFieldSpecs = []FieldSpec{
	{1, "RecordType", "Satzart", 1, ConfidenceSpecified, "Always K"},
	{2, "Reserved", "Reserve", 0, ConfidenceUnknown, ""},
	{3, "CustomerNumber", "Kundennummer", 15, ConfidenceInferred, "Heuristic"},
}

// FieldSpecs returns the documented field table for a record type. The
// returned slice is a copy and may be modified by the caller.
func FieldSpecs(kind Kind) []FieldSpec {
	var source []FieldSpec
	switch kind {
	case KindHeader:
		source = headerFieldSpecs
	case KindArticle:
		source = articleFieldSpecs
	case KindText:
		source = textFieldSpecs
	case KindDimension:
		source = dimensionFieldSpecs
	case KindPrice:
		source = priceFieldSpecs
	case KindGroup:
		source = groupFieldSpecs
	case KindDiscount:
		source = discountFieldSpecs
	case KindEnd:
		source = endFieldSpecs
	case KindChange:
		source = changeFieldSpecs
	case KindMedia:
		source = mediaFieldSpecs
	case KindSurcharge:
		source = surchargeFieldSpecs
	case KindControl:
		source = controlFieldSpecs
	default:
		return nil
	}
	out := make([]FieldSpec, len(source))
	copy(out, source)
	return out
}

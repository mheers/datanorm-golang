package datanorm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/mheers/datanorm-golang/charset"
	"github.com/mheers/datanorm-golang/productdata"
)

type fileRole int

const (
	fileRoleArticles fileRole = iota
	fileRoleGroups
	fileRoleDiscounts
	fileRolePrices
)

// Encoder writes DATANORM 5 files.
type Encoder struct {
	options options
}

// NewEncoder returns an encoder with the given options.
func NewEncoder(opts ...Option) *Encoder {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}
	return &Encoder{options: o}
}

// Encode writes an article file (DATANORM.001 style) from a catalog.
func (e *Encoder) Encode(ctx context.Context, w io.Writer, catalog *productdata.Catalog, opts productdata.EncodeOptions) error {
	return encodeCatalog(ctx, w, catalog, e.options, fileRoleArticles, opts)
}

func joinFields(fields ...string) string {
	return strings.Join(fields, ";")
}

// Encoding errors.
var (
	ErrEncodingFailed   = errors.New("datanorm: encoding failed")
	ErrTooManyTextLines = errors.New("datanorm: text exceeds 99 lines")
)

// EncodeGroups writes a product-group file (DATANORM.WRG style).
func (e *Encoder) EncodeGroups(ctx context.Context, w io.Writer, catalog *productdata.Catalog, opts productdata.EncodeOptions) error {
	return encodeCatalog(ctx, w, catalog, e.options, fileRoleGroups, opts)
}

// EncodeDiscounts writes a discount-group file (DATANORM.RAB style).
func (e *Encoder) EncodeDiscounts(ctx context.Context, w io.Writer, catalog *productdata.Catalog, opts productdata.EncodeOptions) error {
	return encodeCatalog(ctx, w, catalog, e.options, fileRoleDiscounts, opts)
}

// EncodePrices writes a price-change file (DATPREIS style).
func (e *Encoder) EncodePrices(ctx context.Context, w io.Writer, catalog *productdata.Catalog, opts productdata.EncodeOptions) error {
	return encodeCatalog(ctx, w, catalog, e.options, fileRolePrices, opts)
}

func encodeCatalog(ctx context.Context, w io.Writer, catalog *productdata.Catalog, base options, role fileRole, opts productdata.EncodeOptions) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if catalog == nil {
		return fmt.Errorf("%w: nil catalog", ErrEncodingFailed)
	}

	o := base
	if opts.Charset != "" {
		o.charsetName = opts.Charset
	}
	if opts.LineEnding != "" {
		o.lineEnding = opts.LineEnding
	}
	cs, err := o.resolveCharset()
	if err != nil {
		return err
	}
	if cs == charset.Auto {
		cs = charset.CP850
	}
	if o.lineEnding == "" {
		o.lineEnding = "\r\n"
	}

	currency := catalog.Header.Currency
	if !currency.Valid() {
		currency = productdata.EUR
	}
	date := time.Now()
	if catalog.Header.PriceDate != nil {
		date = *catalog.Header.PriceDate
	} else if catalog.Header.CreatedAt != nil {
		date = *catalog.Header.CreatedAt
	}

	var lines []string
	switch role {
	case fileRoleGroups:
		lines = append(lines, headerLine("S", date, currency, catalog.Header))
		for _, group := range catalog.Groups {
			lines = append(lines, groupLine(group))
		}
	case fileRoleDiscounts:
		lines = append(lines, headerLine("R", date, currency, catalog.Header))
		for _, group := range catalog.DiscountGroups {
			lines = append(lines, discountLines(group)...)
		}
	case fileRolePrices:
		lines = append(lines, headerLine("P", date, currency, catalog.Header))
		for _, article := range catalog.Articles {
			lines = append(lines, priceLines(article, currency)...)
		}
	default:
		lines = append(lines, headerLine("A", date, currency, catalog.Header))
		textKeys := make(map[string]string)
		for i := range catalog.Articles {
			article := &catalog.Articles[i]
			if len(article.SupplierArticleNumber) > 15 {
				return fmt.Errorf("%w: article number %q exceeds 15 characters", ErrEncodingFailed, article.SupplierArticleNumber)
			}
			for _, line := range shortTextLines(article) {
				if len(line) > 40 {
					return fmt.Errorf("%w: short text of article %q exceeds 40 characters", ErrEncodingFailed, article.SupplierArticleNumber)
				}
			}
			longLines, err := wrapText(article.Text(productdata.TextLong), 40)
			if err != nil {
				return fmt.Errorf("%w: article %q: %v", ErrEncodingFailed, article.SupplierArticleNumber, err)
			}
			textKey := ""
			if len(longLines) > 0 {
				textKey = uniqueTextKey(article.SupplierArticleNumber, textKeys)
				for index, line := range longLines {
					lines = append(lines, joinFields("T", "N", textKey, "1", fmt.Sprintf("%02d", index+1), line, ""))
				}
			}
			dimensionLines, err := wrapText(article.Text(productdata.TextDimension), 40)
			if err != nil {
				return fmt.Errorf("%w: article %q: %v", ErrEncodingFailed, article.SupplierArticleNumber, err)
			}
			for index, line := range dimensionLines {
				lines = append(lines, joinFields("D", "N", article.SupplierArticleNumber, fmt.Sprintf("%02d", index+1), "F", line, "", ""))
			}
			lines = append(lines, articleLine(article, textKey, currency))
		}
	}

	lines = append(lines, joinFields("E", strconv.Itoa(len(lines)+1), defaultString(catalog.Header.Supplier.Name, "datanorm-golang"), ""))
	data, err := charset.Encode(strings.Join(lines, o.lineEnding)+o.lineEnding, cs)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrEncodingFailed, err)
	}
	_, err = w.Write(data)
	return err
}

func headerLine(marker string, date time.Time, currency productdata.Currency, header productdata.Header) string {
	info := header.Description
	info2 := ""
	info3 := ""
	if len(header.Info) > 0 && info == "" {
		info = header.Info[0]
	}
	if len(header.Info) > 1 {
		info2 = header.Info[1]
	}
	if len(header.Info) > 2 {
		info3 = header.Info[2]
	}
	return joinFields("V", "050", marker, date.Format("20060102"), string(currency), info, info2, info3, "", "", "", "", "", "", "", "")
}

func groupLine(group productdata.Group) string {
	switch group.Level {
	case productdata.GroupProduct:
		return joinFields("S", group.ParentID, group.ID, group.Name, "")
	default:
		return joinFields("S", group.ID, "", group.Name, "")
	}
}

func discountLines(group productdata.DiscountGroup) []string {
	if len(group.Discounts) == 0 {
		return []string{joinFields("R", group.ID, "1", "0", group.Name, "")}
	}
	out := make([]string, 0, len(group.Discounts))
	for _, discount := range group.Discounts {
		out = append(out, joinFields("R", group.ID, discountKindCode(discount.Kind), discount.Value.String(), group.Name, ""))
	}
	return out
}

func discountKindCode(kind productdata.DiscountKind) string {
	switch kind {
	case productdata.DiscountFactor:
		return "2"
	case productdata.DiscountSurcharge:
		return "3"
	default:
		return "1"
	}
}

func priceLines(article productdata.Article, currency productdata.Currency) []string {
	out := make([]string, 0, len(article.Prices))
	for _, price := range article.Prices {
		per := "1"
		if price.Per != nil && !price.Per.Value.IsZero() {
			per = price.Per.Value.String()
		}
		discountGroup := ""
		if group, ok := article.Group(productdata.GroupDiscount); ok {
			discountGroup = group.ID
		}
		value := encodePriceValue(price.Amount)
		if price.Type == productdata.PriceOnRequest && price.Amount.IsZero() {
			value = ""
		}
		out = append(out, joinFields("P", article.SupplierArticleNumber, price.Type.Code(), per, value, discountGroup))
	}
	return out
}

func articleLine(article *productdata.Article, textKey string, currency productdata.Currency) string {
	fields := make([]string, 30)
	fields[0] = "A"
	fields[1] = processingFlagForStatus(article.Status)
	fields[2] = article.SupplierArticleNumber

	short := shortTextLines(article)
	if len(short) > 0 {
		fields[3] = short[0]
	}
	if len(short) > 1 {
		fields[4] = short[1]
	}
	fields[5] = article.Unit.Code
	if price, ok := bestPrice(article); ok {
		fields[6] = price.Type.Code()
		if price.Per != nil && !price.Per.Value.IsZero() {
			fields[7] = price.Per.Value.String()
		} else {
			fields[7] = "1"
		}
		fields[8] = encodePriceValue(price.Amount)
	}
	if group, ok := article.Group(productdata.GroupDiscount); ok {
		fields[9] = group.ID
	}
	if group, ok := article.Group(productdata.GroupMain); ok {
		fields[10] = group.ID
	}
	if group, ok := article.Group(productdata.GroupProduct); ok {
		fields[11] = group.ID
	}
	fields[12] = article.Matchcode
	fields[13] = extra(article, "datanorm.alternative_article_number_creator")
	fields[14] = extra(article, "datanorm.alternative_article_number")
	fields[15] = extra(article, "datanorm.manufacturer_number_creator")
	fields[16] = article.ManufacturerArticleNumber
	fields[17] = article.Type
	fields[18] = article.GTIN
	fields[19] = extra(article, "datanorm.media_reference")
	if article.MinOrderQuantity != nil {
		fields[20] = article.MinOrderQuantity.Value.String()
	}
	fields[21] = extra(article, "datanorm.catalog_page")
	fields[22] = textMarkerFor(article, textKey)
	fields[23] = textKey
	fields[24] = extra(article, "datanorm.storage_flag")
	fields[25] = extra(article, "datanorm.reference_number_creator")
	fields[26] = extra(article, "datanorm.reference_number")
	fields[27] = extra(article, "datanorm.vat_marker")
	fields[28] = extra(article, "datanorm.cost_type")
	if article.PackagingQuantity != nil {
		fields[29] = article.PackagingQuantity.Value.String()
	}
	return strings.Join(fields, ";")
}

func processingFlagForStatus(status productdata.ArticleStatus) string {
	switch status {
	case productdata.StatusDeleted:
		return "L"
	case productdata.StatusChanged, productdata.StatusDiscontinued:
		return "A"
	default:
		return "N"
	}
}

func shortTextLines(article *productdata.Article) []string {
	for _, text := range article.Texts {
		if text.Kind == productdata.TextShort {
			return text.Lines
		}
	}
	if article.ShortText() != "" {
		return []string{article.ShortText()}
	}
	return nil
}

func bestPrice(article *productdata.Article) (productdata.Price, bool) {
	if price, ok := article.Price(productdata.PriceList); ok {
		return price, true
	}
	if price, ok := article.Price(productdata.PriceNet); ok {
		return price, true
	}
	if len(article.Prices) > 0 {
		return article.Prices[0], true
	}
	return productdata.Price{}, false
}

func encodePriceValue(amount productdata.Amount) string {
	if amount.Value.Scale() <= 2 {
		scaled, err := amount.Value.Rescale(2)
		if err == nil {
			return strconv.FormatInt(scaled.Unscaled(), 10)
		}
	}
	return amount.Value.String()
}

func textMarkerFor(article *productdata.Article, textKey string) string {
	if marker := extra(article, "datanorm.text_marker"); marker != "" {
		if TextMarker(marker).Valid() {
			return marker
		}
	}
	hasLong := textKey != ""
	hasDimension := article.Text(productdata.TextDimension) != ""
	switch {
	case hasLong && hasDimension:
		return string(TextMarkerAll)
	case hasLong:
		return string(TextMarkerShortAndLong)
	case hasDimension:
		return string(TextMarkerShortAndDimension)
	default:
		return string(TextMarkerShortOnly)
	}
}

func extra(article *productdata.Article, key string) string {
	if article.Extra == nil {
		return ""
	}
	return article.Extra[key]
}

func uniqueTextKey(articleNumber string, used map[string]string) string {
	base := "T" + sanitizeKey(articleNumber)
	if base == "T" {
		base = "T1"
	}
	if len(base) > 18 {
		base = base[:18]
	}
	key := base
	for i := 2; ; i++ {
		if _, exists := used[key]; !exists {
			used[key] = articleNumber
			return key
		}
		suffix := strconv.Itoa(i)
		candidate := base
		if len(candidate)+len(suffix) > 18 {
			candidate = candidate[:18-len(suffix)]
		}
		key = candidate + suffix
	}
}

func sanitizeKey(value string) string {
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// wrapText breaks text into lines of at most width characters. Existing line
// breaks are preserved. It prefers breaks at spaces. It returns an error if the
// text exceeds 99 lines.
func wrapText(text string, width int) ([]string, error) {
	if text == "" {
		return nil, nil
	}
	var out []string
	for _, paragraph := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		for len(paragraph) > width {
			cut := strings.LastIndexByte(paragraph[:width+1], ' ')
			if cut <= 0 {
				cut = width
			}
			out = append(out, strings.TrimRight(paragraph[:cut], " "))
			paragraph = strings.TrimLeft(paragraph[cut:], " ")
		}
		out = append(out, paragraph)
	}
	for len(out) > 0 && strings.TrimSpace(out[0]) == "" {
		out = out[1:]
	}
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}
	if len(out) > 99 {
		return nil, ErrTooManyTextLines
	}
	return out, nil
}

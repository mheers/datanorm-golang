package datanorm

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/mheers/datanorm-golang/productdata"
)

type textLine struct {
	number int
	text   string
	line   int
}

func assembleCatalog(file *File) (*productdata.Catalog, error) {
	catalog := &productdata.Catalog{
		Diagnostics: append(productdata.Diagnostics(nil), file.Diagnostics...),
	}

	if file.Header == nil {
		return nil, ErrMissingHeader
	}
	catalog.Header = buildHeader(file)

	textBlocks := buildTextBlocks(file, &catalog.Diagnostics)
	dimensionBlocks := buildDimensionBlocks(file, textBlocks, &catalog.Diagnostics)
	mainGroups, productGroups, groupList := buildGroups(file)
	discountGroups := buildDiscountGroups(file, &catalog.Diagnostics)
	mediaByArticle := buildMedia(file)

	articleIndex := make(map[string]int)
	for _, record := range file.Records {
		switch typed := record.(type) {
		case *ArticleRecord:
			article := articleFromRecord(typed, file, catalog.Header.Currency, textBlocks, dimensionBlocks, mainGroups, productGroups, mediaByArticle, &catalog.Diagnostics)
			if index, exists := articleIndex[article.SupplierArticleNumber]; exists {
				catalog.Articles[index] = article
			} else {
				articleIndex[article.SupplierArticleNumber] = len(catalog.Articles)
				catalog.Articles = append(catalog.Articles, article)
			}
		case *PriceRecord:
			applyPrice(typed, catalog, articleIndex, &catalog.Diagnostics)
		}
	}

	// Apply changes after all articles and prices, so that deletions and
	// renumbering see the final article set of this file.
	for _, record := range file.Records {
		if change, ok := record.(*ChangeRecord); ok {
			applyChange(change, catalog, articleIndex, file.Path)
		}
	}

	catalog.Groups = groupList
	catalog.DiscountGroups = discountGroups
	applyGroupNames(catalog)
	return catalog, nil
}

func buildHeader(file *File) productdata.Header {
	header := productdata.Header{
		Format:        "datanorm",
		FormatVersion: "5.0",
		Encoding:      string(file.Charset),
		Description:   file.Header.Description,
		Info:          nonEmptyStrings(file.Header.Info),
		Currency:      productdata.NormalizeCurrency(file.Header.Currency),
		Supplier:      productdata.Supplier{Name: file.Header.SupplierName()},
		RecordCount:   len(file.Records),
		Source:        file.Path,
	}
	if header.Currency == "" {
		header.Currency = productdata.EUR
	}
	if file.Header.DateValid {
		date := file.Header.Date
		header.CreatedAt = &date
		header.PriceDate = &date
	}
	if !header.Currency.Valid() {
		header.Currency = productdata.EUR
	}
	return header
}

func nonEmptyStrings(values []string) []string {
	var out []string
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}

func buildTextBlocks(file *File, diagnostics *productdata.Diagnostics) map[string][]textLine {
	blocks := make(map[string][]textLine)
	for _, record := range file.Texts() {
		if record.Key == "" {
			continue
		}
		if record.ProcessingFlag == FlagDelete {
			delete(blocks, record.Key)
			continue
		}
		blocks[record.Key] = append(blocks[record.Key], textLine{
			number: record.LineNumber,
			text:   record.Text,
			line:   record.Line(),
		})
	}
	for key, lines := range blocks {
		blocks[key] = sortAndDedupeTextLines(lines, "T", key, diagnostics)
	}
	return blocks
}

func sortAndDedupeTextLines(lines []textLine, recordType, key string, diagnostics *productdata.Diagnostics) []textLine {
	sort.SliceStable(lines, func(i, j int) bool {
		if lines[i].number == lines[j].number {
			return lines[i].line < lines[j].line
		}
		return lines[i].number < lines[j].number
	})
	out := lines[:0]
	seen := make(map[int]struct{}, len(lines))
	for _, line := range lines {
		if _, exists := seen[line.number]; exists {
			diagnostics.AddAt(productdata.Diagnostic{
				Severity:   productdata.SeverityWarning,
				Code:       "dn.text-line-duplicate",
				Message:    fmt.Sprintf("duplicate line number %d in text key %q; first line wins", line.number, key),
				Line:       line.line,
				RecordType: recordType,
			})
			continue
		}
		seen[line.number] = struct{}{}
		out = append(out, line)
	}
	return out
}

func joinTextLines(lines []textLine) string {
	if len(lines) == 0 {
		return ""
	}
	texts := make([]string, 0, len(lines))
	for _, line := range lines {
		texts = append(texts, strings.TrimRight(line.text, " \t"))
	}
	for len(texts) > 0 && strings.TrimSpace(texts[0]) == "" {
		texts = texts[1:]
	}
	for len(texts) > 0 && strings.TrimSpace(texts[len(texts)-1]) == "" {
		texts = texts[:len(texts)-1]
	}
	return strings.Join(texts, "\n")
}

func blockLines(blocks map[string][]textLine, key string) []string {
	text := joinTextLines(blocks[key])
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

func buildDimensionBlocks(file *File, textBlocks map[string][]textLine, diagnostics *productdata.Diagnostics) map[string]string {
	grouped := make(map[string][]*DimensionRecord)
	for _, record := range file.Dimensions() {
		if record.ArticleNumber == "" {
			continue
		}
		if record.ProcessingFlag == FlagDelete {
			delete(grouped, record.ArticleNumber)
			continue
		}
		grouped[record.ArticleNumber] = append(grouped[record.ArticleNumber], record)
	}

	out := make(map[string]string, len(grouped))
	for article, records := range grouped {
		sort.SliceStable(records, func(i, j int) bool {
			if records[i].LineNumber == records[j].LineNumber {
				return records[i].Line() < records[j].Line()
			}
			return records[i].LineNumber < records[j].LineNumber
		})
		var texts []string
		seen := make(map[int]struct{}, len(records))
		for _, record := range records {
			if _, exists := seen[record.LineNumber]; exists {
				diagnostics.AddAt(productdata.Diagnostic{
					Severity:   productdata.SeverityWarning,
					Code:       "dn.text-line-duplicate",
					Message:    fmt.Sprintf("duplicate line number %d in dimension text for article %q", record.LineNumber, article),
					Line:       record.Line(),
					RecordType: "D",
				})
				continue
			}
			seen[record.LineNumber] = struct{}{}
			if record.IsInsertion() {
				if record.Key == "" {
					continue
				}
				inserted := blockLines(textBlocks, record.Key)
				if len(inserted) == 0 {
					diagnostics.AddAt(productdata.Diagnostic{
						Severity:   productdata.SeverityWarning,
						Code:       "dn.missing-text",
						Message:    fmt.Sprintf("dimension text inserts missing text key %q", record.Key),
						Line:       record.Line(),
						RecordType: "D",
						Value:      record.Key,
					})
					continue
				}
				texts = append(texts, inserted...)
				continue
			}
			texts = append(texts, strings.TrimRight(record.Text, " \t"))
		}
		joined := strings.Join(texts, "\n")
		joined = strings.Trim(joined, "\n")
		if joined != "" {
			out[article] = joined
		}
	}
	return out
}

func buildGroups(file *File) (map[string]string, map[string]string, []productdata.Group) {
	mainNames := make(map[string]string)
	productNames := make(map[string]string)
	byKey := make(map[string]productdata.Group)
	var order []string

	for _, record := range file.Groups() {
		if record.MainGroup == "" {
			continue
		}
		mainDescription := record.MainGroupDescription
		if mainDescription == "" && record.ProductGroup == "" {
			mainDescription = record.Description
		}
		if mainDescription != "" {
			mainNames[record.MainGroup] = mainDescription
		}
		mainKey := "main\x00" + record.MainGroup
		if _, exists := byKey[mainKey]; !exists {
			name := mainNames[record.MainGroup]
			byKey[mainKey] = productdata.Group{Level: productdata.GroupMain, ID: record.MainGroup, Name: name}
			order = append(order, mainKey)
		}
		if record.ProductGroup == "" {
			continue
		}
		productNames[record.ProductGroup] = record.Description
		key := "product\x00" + record.ProductGroup
		if _, exists := byKey[key]; !exists {
			order = append(order, key)
		}
		byKey[key] = productdata.Group{
			Level:    productdata.GroupProduct,
			ID:       record.ProductGroup,
			Name:     record.Description,
			ParentID: record.MainGroup,
		}
	}

	groups := make([]productdata.Group, 0, len(order))
	for _, key := range order {
		groups = append(groups, byKey[key])
	}
	return mainNames, productNames, groups
}

func buildDiscountGroups(file *File, diagnostics *productdata.Diagnostics) []productdata.DiscountGroup {
	byID := make(map[string]*productdata.DiscountGroup)
	var order []string
	for _, record := range file.Discounts() {
		if record.Group == "" {
			continue
		}
		group, exists := byID[record.Group]
		if !exists {
			group = &productdata.DiscountGroup{ID: record.Group, Name: record.Description}
			byID[record.Group] = group
			order = append(order, record.Group)
		} else if group.Name == "" {
			group.Name = record.Description
		}
		value, err := productdata.ParseDecimal(record.ValueRaw)
		if err != nil {
			diagnostics.AddAt(productdata.Diagnostic{
				Severity:   productdata.SeverityWarning,
				Code:       "dn.discount-value",
				Message:    fmt.Sprintf("discount value %q is not numeric", record.ValueRaw),
				Line:       record.Line(),
				RecordType: "R",
				Field:      4,
				Value:      record.ValueRaw,
			})
			continue
		}
		group.Discounts = append(group.Discounts, productdata.Discount{
			Kind:        discountKind(record.KindRaw),
			Value:       value,
			Description: record.Description,
		})
	}
	out := make([]productdata.DiscountGroup, 0, len(order))
	for _, id := range order {
		out = append(out, *byID[id])
	}
	return out
}

func discountKind(raw string) productdata.DiscountKind {
	switch strings.TrimSpace(raw) {
	case "2":
		return productdata.DiscountFactor
	case "3":
		return productdata.DiscountSurcharge
	default:
		return productdata.DiscountPercent
	}
}

func buildMedia(file *File) map[string][]productdata.Media {
	out := make(map[string][]productdata.Media)
	for _, record := range file.Records {
		mediaRecord, ok := record.(*MediaRecord)
		if !ok || mediaRecord.ArticleNumber == "" {
			continue
		}
		media := productdata.Media{
			Type:        mediaType(mediaRecord.MediaTypeRaw),
			Filename:    mediaRecord.Filename,
			Description: mediaRecord.Description,
		}
		out[mediaRecord.ArticleNumber] = append(out[mediaRecord.ArticleNumber], media)
	}
	return out
}

func mediaType(raw string) productdata.MediaType {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "1", "B", "BILD", "IMG", "IMAGE", "JPG", "JPEG", "PNG", "GIF":
		return productdata.MediaImage
	case "2", "CAD", "ZEICHNUNG", "DRAWING", "DXF", "DWG":
		return productdata.MediaDrawing
	case "3", "VIDEO", "MP4", "AVI":
		return productdata.MediaVideo
	case "4", "AUDIO", "MP3", "WAV":
		return productdata.MediaAudio
	case "5", "DOC", "DOKUMENT", "PDF":
		return productdata.MediaDocument
	default:
		return productdata.MediaOther
	}
}

func articleFromRecord(
	record *ArticleRecord,
	file *File,
	currency productdata.Currency,
	textBlocks map[string][]textLine,
	dimensionBlocks map[string]string,
	mainGroups map[string]string,
	productGroups map[string]string,
	mediaByArticle map[string][]productdata.Media,
	diagnostics *productdata.Diagnostics,
) productdata.Article {
	unit := productdata.NormalizeUnit(record.Unit)
	article := productdata.Article{
		SupplierArticleNumber:     record.Number,
		ManufacturerArticleNumber: record.ManufacturerNumber,
		Type:                      record.ManufacturerType,
		Matchcode:                 record.Matchcode,
		GTIN:                      record.GTIN,
		Unit:                      unit,
		Status:                    articleStatus(record.ProcessingFlag),
		Media:                     mediaByArticle[record.Number],
		Extra:                     make(map[string]string),
		Source: productdata.Source{
			Format:     "datanorm",
			Version:    "5.0",
			File:       file.Path,
			Line:       record.Line(),
			RecordType: "A",
		},
		Raw: record,
	}

	if text := shortText(record); text != "" {
		article.AddText(productdata.NewText(productdata.TextShort, textLines(record.ShortText1, record.ShortText2)...))
	}
	if record.LongTextKey != "" {
		if lines := blockLines(textBlocks, record.LongTextKey); len(lines) > 0 {
			article.AddText(productdata.NewText(productdata.TextLong, lines...))
		} else {
			diagnostics.AddAt(productdata.Diagnostic{
				Severity:   productdata.SeverityWarning,
				Code:       "dn.missing-text",
				Message:    fmt.Sprintf("article %q references missing text key %q", record.Number, record.LongTextKey),
				File:       file.Path,
				Line:       record.Line(),
				RecordType: "A",
				Field:      24,
				Value:      record.LongTextKey,
			})
		}
	}
	if dimension := dimensionBlocks[record.Number]; dimension != "" {
		article.AddText(productdata.NewText(productdata.TextDimension, strings.Split(dimension, "\n")...))
	}

	if record.PriceRaw != "" || strings.TrimSpace(record.PriceTypeRaw) == "9" {
		if record.PriceRaw == "" {
			article.Prices = append(article.Prices, productdata.Price{
				Type: productdata.PriceOnRequest,
			})
		} else {
			amount, err := ParsePrice(record.PriceRaw, currency)
			if err != nil {
				diagnostics.AddAt(productdata.Diagnostic{
					Severity:   productdata.SeverityWarning,
					Code:       "dn.price",
					Message:    fmt.Sprintf("article %q has an invalid price %q", record.Number, record.PriceRaw),
					File:       file.Path,
					Line:       record.Line(),
					RecordType: "A",
					Field:      9,
					Value:      record.PriceRaw,
				})
			} else {
				price := productdata.Price{
					Type:   productdata.PriceTypeFromCode(record.PriceTypeRaw),
					Amount: amount,
				}
				if per, ok := record.PricePer(); ok && per != 1 {
					price.Per = &productdata.Quantity{Value: productdata.NewDecimal(per, 0), Unit: unit}
				} else if _, ok := record.PricePer(); !ok && strings.TrimSpace(record.PriceUnitRaw) != "" {
					diagnostics.AddAt(productdata.Diagnostic{
						Severity:   productdata.SeverityWarning,
						Code:       "dn.price-unit",
						Message:    fmt.Sprintf("article %q has an invalid price unit %q", record.Number, record.PriceUnitRaw),
						File:       file.Path,
						Line:       record.Line(),
						RecordType: "A",
						Field:      8,
						Value:      record.PriceUnitRaw,
					})
				}
				article.Prices = append(article.Prices, price)
			}
		}
	}

	if record.MinOrderQuantityRaw != "" {
		if quantity, err := productdata.ParseDecimal(record.MinOrderQuantityRaw); err == nil {
			article.MinOrderQuantity = &productdata.Quantity{Value: quantity, Unit: unit}
		}
	}
	if record.PackagingQuantity != "" {
		if quantity, err := productdata.ParseDecimal(record.PackagingQuantity); err == nil {
			article.PackagingQuantity = &productdata.Quantity{Value: quantity, Unit: unit}
		}
	}

	if record.DiscountGroup != "" {
		article.Groups = append(article.Groups, productdata.GroupRef{
			Level: productdata.GroupDiscount,
			ID:    record.DiscountGroup,
		})
	}
	if record.MainGroup != "" {
		article.Groups = append(article.Groups, productdata.GroupRef{
			Level: productdata.GroupMain,
			ID:    record.MainGroup,
			Name:  mainGroups[record.MainGroup],
		})
	}
	if record.ProductGroup != "" {
		article.Groups = append(article.Groups, productdata.GroupRef{
			Level: productdata.GroupProduct,
			ID:    record.ProductGroup,
			Name:  productGroups[record.ProductGroup],
		})
	}

	if record.GTIN != "" && !productdata.ValidateGTIN(record.GTIN) {
		diagnostics.AddAt(productdata.Diagnostic{
			Severity:   productdata.SeverityWarning,
			Code:       "dn.ean-checksum",
			Message:    fmt.Sprintf("article %q has an invalid EAN/GTIN %q", record.Number, record.GTIN),
			File:       file.Path,
			Line:       record.Line(),
			RecordType: "A",
			Field:      19,
			Value:      record.GTIN,
		})
	}

	putExtra(article.Extra, "datanorm.alternative_article_number", record.AlternativeArticleNumber)
	putExtra(article.Extra, "datanorm.alternative_article_number_creator", record.AlternativeArticleNumberCreator)
	putExtra(article.Extra, "datanorm.manufacturer_number_creator", record.ManufacturerNumberCreator)
	putExtra(article.Extra, "datanorm.media_reference", record.MediaReference)
	putExtra(article.Extra, "datanorm.catalog_page", record.CatalogPage)
	putExtra(article.Extra, "datanorm.text_marker", string(record.TextMarker))
	putExtra(article.Extra, "datanorm.storage_flag", record.StorageFlag)
	putExtra(article.Extra, "datanorm.reference_number_creator", record.ReferenceNumberCreator)
	putExtra(article.Extra, "datanorm.reference_number", record.ReferenceNumber)
	putExtra(article.Extra, "datanorm.vat_marker", record.VATMarker)
	putExtra(article.Extra, "datanorm.cost_type", record.CostType)
	putExtra(article.Extra, "datanorm.packaging_quantity", record.PackagingQuantity)
	putExtra(article.Extra, "datanorm.price_type", record.PriceTypeRaw)
	putExtra(article.Extra, "datanorm.price_unit", record.PriceUnitRaw)
	if len(article.Extra) == 0 {
		article.Extra = nil
	}
	return article
}

func shortText(record *ArticleRecord) string {
	return strings.TrimSpace(strings.Join([]string{record.ShortText1, record.ShortText2}, " "))
}

func textLines(values ...string) []string {
	var out []string
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}

func articleStatus(flag ProcessingFlag) productdata.ArticleStatus {
	switch flag {
	case FlagDelete:
		return productdata.StatusDeleted
	case FlagChange:
		return productdata.StatusChanged
	case FlagRenumber:
		return productdata.StatusChanged
	default:
		return productdata.StatusActive
	}
}

func putExtra(extra map[string]string, key, value string) {
	if strings.TrimSpace(value) != "" {
		extra[key] = value
	}
}

func applyChange(change *ChangeRecord, catalog *productdata.Catalog, index map[string]int, file string) {
	source := productdata.Source{
		Format:     "datanorm",
		Version:    "5.0",
		File:       file,
		Line:       change.Line(),
		RecordType: "B",
	}
	switch change.ProcessingFlag {
	case FlagDelete:
		if i, ok := index[change.ArticleNumber]; ok {
			catalog.Articles[i].Status = productdata.StatusDeleted
			return
		}
		if change.ArticleNumber != "" {
			index[change.ArticleNumber] = len(catalog.Articles)
			catalog.Articles = append(catalog.Articles, productdata.Article{
				SupplierArticleNumber: change.ArticleNumber,
				Status:                productdata.StatusDeleted,
				Source:                source,
				Raw:                   change,
			})
		}
	case FlagRenumber:
		if change.NewArticleNumber == "" {
			return
		}
		if i, ok := index[change.ArticleNumber]; ok {
			catalog.Articles[i].SupplierArticleNumber = change.NewArticleNumber
			catalog.Articles[i].Status = productdata.StatusChanged
			catalog.Articles[i].Extra = ensureExtra(catalog.Articles[i].Extra)
			catalog.Articles[i].Extra["datanorm.previous_article_number"] = change.ArticleNumber
			delete(index, change.ArticleNumber)
			index[change.NewArticleNumber] = i
			return
		}
		index[change.NewArticleNumber] = len(catalog.Articles)
		catalog.Articles = append(catalog.Articles, productdata.Article{
			SupplierArticleNumber: change.NewArticleNumber,
			Status:                productdata.StatusChanged,
			Extra:                 map[string]string{"datanorm.previous_article_number": change.ArticleNumber},
			Source:                source,
			Raw:                   change,
		})
	case FlagChange:
		if i, ok := index[change.ArticleNumber]; ok {
			if change.Successor != "" {
				catalog.Articles[i].Status = productdata.StatusDiscontinued
				catalog.Articles[i].Replacement = &productdata.ArticleRef{SupplierArticleNumber: change.Successor}
			}
		}
	}
}

func ensureExtra(extra map[string]string) map[string]string {
	if extra == nil {
		return make(map[string]string)
	}
	return extra
}

func applyPrice(record *PriceRecord, catalog *productdata.Catalog, index map[string]int, diagnostics *productdata.Diagnostics) {
	if record.ArticleNumber == "" {
		return
	}
	if record.PriceRaw == "" && strings.TrimSpace(record.PriceTypeRaw) == "9" {
		price := productdata.Price{Type: productdata.PriceOnRequest}
		storePrice(record, catalog, index, price)
		return
	}
	amount, err := ParsePrice(record.PriceRaw, catalog.Header.Currency)
	if err != nil {
		if errors.Is(err, ErrNoPrice) {
			return
		}
		diagnostics.AddAt(productdata.Diagnostic{
			Severity:   productdata.SeverityWarning,
			Code:       "dn.price",
			Message:    fmt.Sprintf("price record for article %q has an invalid price %q", record.ArticleNumber, record.PriceRaw),
			Line:       record.Line(),
			RecordType: "P",
			Field:      5,
			Value:      record.PriceRaw,
		})
		return
	}
	price := productdata.Price{
		Type:   productdata.PriceTypeFromCode(record.PriceTypeRaw),
		Amount: amount,
	}
	if per, err := strconv.ParseInt(strings.TrimSpace(record.PriceUnitRaw), 10, 64); err == nil && per > 1 {
		price.Per = &productdata.Quantity{Value: productdata.NewDecimal(per, 0)}
	}
	storePrice(record, catalog, index, price)
}

func storePrice(record *PriceRecord, catalog *productdata.Catalog, index map[string]int, price productdata.Price) {
	i, exists := index[record.ArticleNumber]
	if !exists {
		article := productdata.Article{
			SupplierArticleNumber: record.ArticleNumber,
			Prices:                []productdata.Price{price},
			Status:                productdata.StatusChanged,
			Source: productdata.Source{
				Format:     "datanorm",
				Version:    "5.0",
				Line:       record.Line(),
				RecordType: "P",
			},
			Raw: record,
		}
		if record.DiscountGroup != "" {
			article.Groups = append(article.Groups, productdata.GroupRef{
				Level: productdata.GroupDiscount,
				ID:    record.DiscountGroup,
			})
		}
		catalog.Articles = append(catalog.Articles, article)
		index[record.ArticleNumber] = len(catalog.Articles) - 1
		return
	}
	article := &catalog.Articles[i]
	if price.Per != nil && price.Per.Unit.Code == "" {
		price.Per.Unit = article.Unit
	}
	if record.DiscountGroup != "" {
		found := false
		for _, group := range article.Groups {
			if group.Level == productdata.GroupDiscount {
				found = true
				break
			}
		}
		if !found {
			article.Groups = append(article.Groups, productdata.GroupRef{
				Level: productdata.GroupDiscount,
				ID:    record.DiscountGroup,
			})
		}
	}
	for j := range article.Prices {
		if article.Prices[j].Type == price.Type {
			article.Prices[j] = price
			return
		}
	}
	article.Prices = append(article.Prices, price)
}

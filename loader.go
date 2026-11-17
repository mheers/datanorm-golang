package datanorm

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mheers/datanorm-golang/charset"
	"github.com/mheers/datanorm-golang/productdata"
)

// DeliveryRole is the role of a file inside a DATANORM delivery.
type DeliveryRole string

// Delivery roles.
const (
	RoleArticles  DeliveryRole = "articles"
	RolePrices    DeliveryRole = "prices"
	RoleGroups    DeliveryRole = "groups"
	RoleDiscounts DeliveryRole = "discounts"
	RoleSets      DeliveryRole = "sets"
	RoleTexts     DeliveryRole = "texts"
	RoleUnknown   DeliveryRole = "unknown"
)

// RoleOf returns the role of a parsed file, using the V data marker first and
// the file name as a fallback.
func RoleOf(file *File) DeliveryRole {
	if file != nil && file.Header != nil {
		switch strings.ToUpper(strings.TrimSpace(file.Header.DataMarker)) {
		case "A":
			return RoleArticles
		case "P":
			return RolePrices
		case "S":
			return RoleGroups
		case "R":
			return RoleDiscounts
		case "J":
			return RoleSets
		case "T":
			return RoleTexts
		}
	}
	if file != nil {
		return RoleFromName(file.Path)
	}
	return RoleUnknown
}

// RoleFromName guesses the role from a file name.
func RoleFromName(path string) DeliveryRole {
	base := strings.ToLower(filepath.Base(path))
	switch {
	case strings.HasSuffix(base, ".rab"):
		return RoleDiscounts
	case strings.HasSuffix(base, ".wrg"):
		return RoleGroups
	case strings.HasPrefix(base, "datpreis"):
		return RolePrices
	case strings.HasPrefix(base, "datasets"):
		return RoleSets
	case strings.HasPrefix(base, "datatext"):
		return RoleTexts
	case strings.HasPrefix(base, "datanorm"):
		return RoleArticles
	default:
		return RoleUnknown
	}
}

// sniffRole reads the first line of a file to determine its role from the V
// data marker. It is used for ordering before the full parse, so that renamed
// files still work.
func sniffRole(path string) DeliveryRole {
	fallback := RoleFromName(path)
	file, err := os.Open(path)
	if err != nil {
		return fallback
	}
	defer func() { _ = file.Close() }()

	reader := bufio.NewReaderSize(file, 4096)
	sample, _ := reader.Peek(4096)
	decoded, err := charset.Decode(sample, charset.Auto)
	if err != nil {
		return fallback
	}
	line := decoded
	if index := strings.IndexAny(line, "\r\n"); index >= 0 {
		line = line[:index]
	}
	line = strings.TrimPrefix(line, "\uFEFF")
	fields := strings.Split(line, ";")
	if len(fields) < 3 || fields[0] != "V" {
		return fallback
	}
	switch strings.ToUpper(strings.TrimSpace(fields[2])) {
	case "A":
		return RoleArticles
	case "P":
		return RolePrices
	case "S":
		return RoleGroups
	case "R":
		return RoleDiscounts
	case "J":
		return RoleSets
	case "T":
		return RoleTexts
	default:
		return fallback
	}
}

// Loader reads a complete delivery.
type Loader struct {
	options []Option
}

// NewLoader returns a loader with the given parser options.
func NewLoader(opts ...Option) *Loader {
	return &Loader{options: opts}
}

// Load reads a complete delivery from paths in the recommended order,
// regardless of the order in which the paths are supplied. Articles,
// groups, discount groups, and price changes are merged into one catalog.
func (l *Loader) Load(ctx context.Context, paths ...string) (*productdata.Catalog, error) {
	return loadDelivery(ctx, paths, l.options)
}

// Load reads a complete delivery from paths.
func Load(ctx context.Context, paths []string, opts ...Option) (*productdata.Catalog, error) {
	return loadDelivery(ctx, paths, opts)
}

func loadDelivery(ctx context.Context, paths []string, opts []Option) (*productdata.Catalog, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, ErrNoFiles
	}
	type entry struct {
		path string
		role DeliveryRole
	}
	entries := make([]entry, len(paths))
	for i, path := range paths {
		entries[i] = entry{path: path, role: sniffRole(path)}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if roleRank(entries[i].role) != roleRank(entries[j].role) {
			return roleRank(entries[i].role) < roleRank(entries[j].role)
		}
		return entries[i].path < entries[j].path
	})

	merged := &productdata.Catalog{}
	articleIndex := make(map[string]int)
	groupIndex := make(map[string]int)
	discountIndex := make(map[string]int)
	totalRecords := 0
	ordered := make([]string, 0, len(entries))
	type pendingChange struct {
		change *ChangeRecord
		file   string
	}
	var changes []pendingChange

	for _, item := range entries {
		path := item.path
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		file, err := Open(path, opts...)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		ordered = append(ordered, path)
		totalRecords += len(file.Records)
		role := RoleOf(file)
		catalog, err := file.catalogWithoutChanges()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		mergeHeader(&merged.Header, catalog.Header, role)
		merged.Diagnostics = append(merged.Diagnostics, prefixDiagnostics(catalog.Diagnostics, path)...)

		if merged.Header.PriceDate == nil && catalog.Header.PriceDate != nil {
			merged.Header.PriceDate = catalog.Header.PriceDate
		}
		if merged.Header.CreatedAt == nil && catalog.Header.CreatedAt != nil {
			merged.Header.CreatedAt = catalog.Header.CreatedAt
		}

		switch role {
		case RoleGroups:
			for _, group := range catalog.Groups {
				key := string(group.Level) + "\x00" + group.ID
				if index, ok := groupIndex[key]; ok {
					merged.Groups[index] = group
				} else {
					groupIndex[key] = len(merged.Groups)
					merged.Groups = append(merged.Groups, group)
				}
			}
		case RoleDiscounts:
			for _, group := range catalog.DiscountGroups {
				if index, ok := discountIndex[group.ID]; ok {
					merged.DiscountGroups[index].Discounts = append(merged.DiscountGroups[index].Discounts, group.Discounts...)
					if merged.DiscountGroups[index].Name == "" {
						merged.DiscountGroups[index].Name = group.Name
					}
				} else {
					discountIndex[group.ID] = len(merged.DiscountGroups)
					merged.DiscountGroups = append(merged.DiscountGroups, group)
				}
			}
		default:
			mergeArticles := role != RoleArticles || len(file.Articles()) > 0
			if mergeArticles {
				for _, article := range catalog.Articles {
					if index, ok := articleIndex[article.SupplierArticleNumber]; ok {
						merged.Articles[index] = mergeArticle(merged.Articles[index], article)
					} else {
						articleIndex[article.SupplierArticleNumber] = len(merged.Articles)
						merged.Articles = append(merged.Articles, article)
					}
				}
			}
			for _, change := range file.Changes() {
				changes = append(changes, pendingChange{change: change, file: path})
			}
			for _, group := range catalog.Groups {
				key := string(group.Level) + "\x00" + group.ID
				if index, ok := groupIndex[key]; ok {
					merged.Groups[index] = group
				} else {
					groupIndex[key] = len(merged.Groups)
					merged.Groups = append(merged.Groups, group)
				}
			}
			merged.Sets = append(merged.Sets, catalog.Sets...)
			merged.Surcharges = append(merged.Surcharges, catalog.Surcharges...)
		}
	}

	// Apply deletions and renumbering after all article and price files, so that
	// file order cannot resurrect a deleted article.
	for _, item := range changes {
		applyChange(item.change, merged, articleIndex, item.file)
	}

	// Re-resolve group names on all articles after groups have been merged.
	applyGroupNames(merged)
	merged.Header.RecordCount = totalRecords
	merged.Header.Source = strings.Join(ordered, ", ")
	return merged, nil
}

func mergeHeader(dst *productdata.Header, src productdata.Header, role DeliveryRole) {
	if dst.Format == "" {
		dst.Format = src.Format
	}
	if dst.FormatVersion == "" {
		dst.FormatVersion = src.FormatVersion
	}
	if dst.Encoding == "" {
		dst.Encoding = src.Encoding
	}
	if dst.Currency == "" && src.Currency != "" {
		dst.Currency = src.Currency
	}
	if role == RoleArticles {
		dst.Description = src.Description
		dst.Info = src.Info
		dst.Supplier = src.Supplier
		dst.PriceDate = src.PriceDate
		dst.CreatedAt = src.CreatedAt
	} else if dst.Description == "" {
		dst.Description = src.Description
		dst.Info = src.Info
		dst.Supplier = src.Supplier
	}
}

func mergeArticle(base, next productdata.Article) productdata.Article {
	if next.SupplierArticleNumber == "" {
		next.SupplierArticleNumber = base.SupplierArticleNumber
	}
	if next.Manufacturer == "" {
		next.Manufacturer = base.Manufacturer
	}
	if next.ManufacturerArticleNumber == "" {
		next.ManufacturerArticleNumber = base.ManufacturerArticleNumber
	}
	if next.Type == "" {
		next.Type = base.Type
	}
	if next.Matchcode == "" {
		next.Matchcode = base.Matchcode
	}
	if next.GTIN == "" {
		next.GTIN = base.GTIN
	}
	if next.Unit.Code == "" {
		next.Unit = base.Unit
	}
	if len(next.Texts) == 0 {
		next.Texts = base.Texts
	} else {
		for _, baseText := range base.Texts {
			found := false
			for _, nextText := range next.Texts {
				if nextText.Kind == baseText.Kind && nextText.Language == baseText.Language {
					found = true
					break
				}
			}
			if !found {
				next.Texts = append(next.Texts, baseText)
			}
		}
	}
	if len(next.Groups) == 0 {
		next.Groups = base.Groups
	} else {
		for _, baseGroup := range base.Groups {
			found := false
			for _, nextGroup := range next.Groups {
				if nextGroup.Level == baseGroup.Level {
					found = true
					break
				}
			}
			if !found {
				next.Groups = append(next.Groups, baseGroup)
			}
		}
	}
	if next.MinOrderQuantity == nil {
		next.MinOrderQuantity = base.MinOrderQuantity
	}
	if next.PackagingQuantity == nil {
		next.PackagingQuantity = base.PackagingQuantity
	}
	if next.Status == "" {
		next.Status = base.Status
	}
	if next.Replacement == nil {
		next.Replacement = base.Replacement
	}
	if len(next.Media) == 0 {
		next.Media = base.Media
	} else {
		for _, baseMedia := range base.Media {
			found := false
			for _, nextMedia := range next.Media {
				if nextMedia.Type == baseMedia.Type && nextMedia.Filename == baseMedia.Filename {
					found = true
					break
				}
			}
			if !found {
				next.Media = append(next.Media, baseMedia)
			}
		}
	}
	if next.Logistics.Weight == nil {
		next.Logistics.Weight = base.Logistics.Weight
	}
	if next.Logistics.Length == nil {
		next.Logistics.Length = base.Logistics.Length
	}
	if next.Logistics.Width == nil {
		next.Logistics.Width = base.Logistics.Width
	}
	if next.Logistics.Height == nil {
		next.Logistics.Height = base.Logistics.Height
	}
	if next.Logistics.PackagingQuantity == nil {
		next.Logistics.PackagingQuantity = base.Logistics.PackagingQuantity
	}
	if next.Logistics.CountryOfOrigin == "" {
		next.Logistics.CountryOfOrigin = base.Logistics.CountryOfOrigin
	}
	if next.Logistics.CustomsTariffNumber == "" {
		next.Logistics.CustomsTariffNumber = base.Logistics.CustomsTariffNumber
	}
	if next.Logistics.StorageFlag == "" {
		next.Logistics.StorageFlag = base.Logistics.StorageFlag
	}
	if len(next.Features) == 0 {
		next.Features = base.Features
	} else {
		for _, baseFeature := range base.Features {
			found := false
			for _, nextFeature := range next.Features {
				if nextFeature.Group == baseFeature.Group && nextFeature.Name == baseFeature.Name {
					found = true
					break
				}
			}
			if !found {
				next.Features = append(next.Features, baseFeature)
			}
		}
	}
	if len(next.Classifications) == 0 {
		next.Classifications = base.Classifications
	} else {
		for _, baseClassification := range base.Classifications {
			found := false
			for _, nextClassification := range next.Classifications {
				if nextClassification.System == baseClassification.System && nextClassification.Code == baseClassification.Code {
					found = true
					break
				}
			}
			if !found {
				next.Classifications = append(next.Classifications, baseClassification)
			}
		}
	}
	if len(next.Associations) == 0 {
		next.Associations = base.Associations
	} else {
		for _, baseAssociation := range base.Associations {
			found := false
			for _, nextAssociation := range next.Associations {
				if nextAssociation.Kind == baseAssociation.Kind &&
					nextAssociation.Article.SupplierArticleNumber == baseAssociation.Article.SupplierArticleNumber {
					found = true
					break
				}
			}
			if !found {
				next.Associations = append(next.Associations, baseAssociation)
			}
		}
	}
	if next.Source.RecordType == "P" && base.Source.RecordType == "A" {
		// Keep the article's provenance when a price file only adds prices.
		next.Source = base.Source
	}
	for _, price := range base.Prices {
		found := false
		for _, candidate := range next.Prices {
			if candidate.Type == price.Type {
				found = true
				break
			}
		}
		if !found {
			next.Prices = append(next.Prices, price)
		}
	}
	// A price file carries no unit; fill it from the article data.
	for i := range next.Prices {
		if next.Prices[i].Per != nil && next.Prices[i].Per.Unit.Code == "" {
			next.Prices[i].Per.Unit = next.Unit
		}
	}
	if next.Extra == nil {
		next.Extra = base.Extra
	} else {
		for key, value := range base.Extra {
			if _, exists := next.Extra[key]; !exists {
				next.Extra[key] = value
			}
		}
	}
	if next.Raw == nil {
		next.Raw = base.Raw
	}
	return next
}

func applyGroupNames(catalog *productdata.Catalog) {
	names := make(map[string]string)
	for _, group := range catalog.Groups {
		names[string(group.Level)+"\x00"+group.ID] = group.Name
	}
	for _, group := range catalog.DiscountGroups {
		names[string(productdata.GroupDiscount)+"\x00"+group.ID] = group.Name
	}
	for i := range catalog.Articles {
		for j := range catalog.Articles[i].Groups {
			key := string(catalog.Articles[i].Groups[j].Level) + "\x00" + catalog.Articles[i].Groups[j].ID
			if name, ok := names[key]; ok && name != "" {
				catalog.Articles[i].Groups[j].Name = name
			}
		}
	}
}

func roleRank(role DeliveryRole) int {
	switch role {
	case RoleDiscounts:
		return 0
	case RoleGroups:
		return 1
	case RoleArticles:
		return 2
	case RolePrices:
		return 3
	case RoleTexts:
		return 4
	case RoleSets:
		return 5
	default:
		return 6
	}
}

func prefixDiagnostics(diagnostics productdata.Diagnostics, path string) productdata.Diagnostics {
	out := make(productdata.Diagnostics, len(diagnostics))
	copy(out, diagnostics)
	for i := range out {
		if out[i].File == "" {
			out[i].File = path
		}
	}
	return out
}

// FindDeliveryFiles returns the DATANORM files in a directory, in processing
// order. It accepts the standard names, renamed article files such as
// DATANORM_Komplettdatei.001, and the .RAB/.WRG group files.
func FindDeliveryFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	type candidate struct {
		path string
		role DeliveryRole
	}
	var candidates []candidate
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		if !isDeliveryFileName(name) {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		candidates = append(candidates, candidate{path: path, role: sniffRole(path)})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if roleRank(candidates[i].role) != roleRank(candidates[j].role) {
			return roleRank(candidates[i].role) < roleRank(candidates[j].role)
		}
		return candidates[i].path < candidates[j].path
	})
	paths := make([]string, len(candidates))
	for i, candidate := range candidates {
		paths[i] = candidate.path
	}
	return paths, nil
}

// isDeliveryFileName reports whether a lower-case base name is a DATANORM
// delivery file. The numeric extension is required for DATANORM/DATPREIS/
// DATASETS/DATATEXT files, so a file such as datanorm-readme.txt is ignored.
func isDeliveryFileName(name string) bool {
	if strings.HasSuffix(name, ".rab") || strings.HasSuffix(name, ".wrg") {
		return true
	}
	ext := filepath.Ext(name)
	if len(ext) != 4 || ext[0] != '.' || !allDigits(ext[1:]) {
		return false
	}
	return strings.HasPrefix(name, "datanorm") || strings.HasPrefix(name, "datpreis") ||
		strings.HasPrefix(name, "datasets") || strings.HasPrefix(name, "datatext")
}

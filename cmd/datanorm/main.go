// Command datanorm inspects, validates, and converts DATANORM 5 files.
package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mheers/datanorm-golang"
	"github.com/mheers/datanorm-golang/productdata"
)

const usageText = `datanorm - inspect, validate, and convert DATANORM 5 files

Usage:
  datanorm <command> [flags] FILE...

Commands:
  info        show header, encoding, record counts, and diagnostics summary
  validate    print diagnostics; exit 1 on errors (or warnings with -strict)
  articles    write articles as CSV
  json        write the canonical catalog as JSON
  encodings   show the detected character encoding and header
  version     print the library version

Flags:
  -charset NAME   override encoding detection (auto, utf-8, cp437, cp850,
                  windows-1252, iso-8859-15, iso-8859-1)
  -strict         exit 1 when warnings are present
  -delivery       treat all FILE arguments as one delivery and merge them

Examples:
  datanorm info DATANORM.001
  datanorm info -delivery DATANORM.RAB DATANORM.WRG DATANORM.001 DATPREIS.001
  datanorm validate -delivery DATANORM.001 DATPREIS.001
  datanorm articles DATANORM.001 > articles.csv
  datanorm json -delivery DATANORM.001 DATANORM.RAB DATANORM.WRG > catalog.json
`

func writeString(w io.Writer, s string) {
	_, _ = io.WriteString(w, s)
}

func writef(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

func writeln(w io.Writer, args ...any) {
	_, _ = fmt.Fprintln(w, args...)
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		writeString(stderr, usageText)
		return 2
	}
	command := args[0]
	if command == "help" || command == "-h" || command == "--help" {
		writeString(stdout, usageText)
		return 0
	}
	if command == "version" {
		writef(stdout, "datanorm %s (DATANORM %s)\n", datanorm.LibraryVersion, datanorm.Version)
		return 0
	}

	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(stderr)
	charsetName := fs.String("charset", "", "override character encoding")
	strict := fs.Bool("strict", false, "exit 1 when warnings are present")
	delivery := fs.Bool("delivery", false, "merge all files as one delivery")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	paths := fs.Args()
	if len(paths) == 0 {
		writeString(stderr, usageText)
		return 2
	}

	var options []datanorm.Option
	if *charsetName != "" {
		options = append(options, datanorm.WithCharset(*charsetName))
	}

	switch command {
	case "info":
		return commandInfo(stdout, stderr, paths, *delivery, options)
	case "validate":
		return commandValidate(stdout, stderr, paths, *delivery, *strict, options)
	case "articles":
		return commandArticles(stdout, stderr, paths, *delivery, options)
	case "json":
		return commandJSON(stdout, stderr, paths, *delivery, options)
	case "encodings":
		return commandEncodings(stdout, stderr, paths, options)
	default:
		writef(stderr, "unknown command %q\n\n%s", command, usageText)
		return 2
	}
}

func commandInfo(stdout, stderr io.Writer, paths []string, delivery bool, options []datanorm.Option) int {
	if delivery || len(paths) > 1 {
		catalog, err := datanorm.Load(context.Background(), paths, options...)
		if err != nil {
			writeln(stderr, "error:", err)
			return 1
		}
		printCatalogInfo(stdout, catalog, len(paths))
		return 0
	}
	file, err := datanorm.Open(paths[0], options...)
	if err != nil {
		writeln(stderr, "error:", err)
		return 1
	}
	printFileInfo(stdout, file)
	return 0
}

func printFileInfo(w io.Writer, file *datanorm.File) {
	writef(w, "File:        %s\n", file.Path)
	if file.Header != nil {
		writef(w, "Format:      DATANORM %s (%s)\n", datanorm.Version, file.Header.DataType())
		writef(w, "Encoding:    %s\n", file.Charset)
		writef(w, "Date:        %s\n", formatDate(file.Header.DateRaw, file.Header.DateValid, file.Header.Date))
		writef(w, "Currency:    %s\n", file.Header.Currency)
		writef(w, "Description: %s\n", file.Header.Description)
	}
	writef(w, "Records:     %d\n", len(file.Records))
	printKindCounts(w, file.KindCount())
	diagnostics := file.Diagnostics
	if catalog, err := file.Catalog(); err == nil {
		writef(w, "Articles:    %d\n", len(catalog.Articles))
		writef(w, "Groups:      %d\n", len(catalog.Groups))
		writef(w, "Discounts:   %d\n", len(catalog.DiscountGroups))
		diagnostics = catalog.Diagnostics
	}
	printDiagnosticSummary(w, diagnostics)
}

func printCatalogInfo(w io.Writer, catalog *productdata.Catalog, fileCount int) {
	writef(w, "Delivery:    %d file(s)\n", fileCount)
	writef(w, "Format:      %s %s\n", catalog.Header.Format, catalog.Header.FormatVersion)
	if catalog.Header.PriceDate != nil {
		writef(w, "Price date:  %s\n", catalog.Header.PriceDate.Format("2006-01-02"))
	}
	writef(w, "Currency:    %s\n", catalog.Header.Currency)
	writef(w, "Description: %s\n", catalog.Header.Description)
	if catalog.Header.Supplier.Name != "" {
		writef(w, "Supplier:    %s\n", catalog.Header.Supplier.Name)
	}
	writef(w, "Articles:    %d\n", len(catalog.Articles))
	writef(w, "Groups:      %d\n", len(catalog.Groups))
	writef(w, "Discounts:   %d\n", len(catalog.DiscountGroups))
	printDiagnosticSummary(w, catalog.Diagnostics)
}

func printKindCounts(w io.Writer, counts map[datanorm.Kind]int) {
	var kinds []int
	for kind := range counts {
		kinds = append(kinds, int(kind))
	}
	sort.Ints(kinds)
	parts := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		parts = append(parts, fmt.Sprintf("%c=%d", byte(kind), counts[datanorm.Kind(byte(kind))]))
	}
	writef(w, "By type:     %s\n", strings.Join(parts, " "))
}

func printDiagnosticSummary(w io.Writer, diagnostics productdata.Diagnostics) {
	writef(w, "Diagnostics: %d error(s), %d warning(s), %d info\n",
		diagnostics.Count(productdata.SeverityError),
		diagnostics.Count(productdata.SeverityWarning),
		diagnostics.Count(productdata.SeverityInfo))
	codes := make(map[string]int)
	for _, diagnostic := range diagnostics {
		codes[diagnostic.Code]++
	}
	var keys []string
	for key := range codes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		writef(w, "  %-28s %d\n", key+":", codes[key])
	}
}

func commandValidate(stdout, stderr io.Writer, paths []string, delivery, strict bool, options []datanorm.Option) int {
	var diagnostics productdata.Diagnostics
	if delivery || len(paths) > 1 {
		catalog, err := datanorm.Load(context.Background(), paths, options...)
		if err != nil {
			writeln(stderr, "error:", err)
			return 1
		}
		diagnostics = catalog.Diagnostics
	} else {
		file, err := datanorm.Open(paths[0], options...)
		if err != nil {
			writeln(stderr, "error:", err)
			return 1
		}
		catalog, err := file.Catalog()
		if err != nil {
			writeln(stderr, "error:", err)
			return 1
		}
		diagnostics = catalog.Diagnostics
	}
	for _, diagnostic := range diagnostics {
		location := diagnostic.File
		if location == "" {
			location = "-"
		}
		if diagnostic.Line > 0 {
			location += ":" + strconv.Itoa(diagnostic.Line)
		}
		writef(stdout, "%s: %s %s: %s\n", location, diagnostic.Severity, diagnostic.Code, diagnostic.Message)
	}
	if diagnostics.HasErrors() {
		return 1
	}
	if strict && diagnostics.Count(productdata.SeverityWarning) > 0 {
		return 1
	}
	return 0
}

func commandArticles(stdout, stderr io.Writer, paths []string, delivery bool, options []datanorm.Option) int {
	catalog, err := loadCatalog(paths, delivery, options)
	if err != nil {
		writeln(stderr, "error:", err)
		return 1
	}
	writer := csv.NewWriter(stdout)
	_ = writer.Write([]string{
		"article_number", "short_text", "long_text", "unit",
		"list_price", "list_currency", "net_price", "net_currency",
		"price_unit", "discount_group", "main_group", "product_group",
		"gtin", "matchcode", "manufacturer_number", "manufacturer_type", "status",
	})
	for _, article := range catalog.Articles {
		listPrice, listCurrency := priceFields(article, productdata.PriceList)
		netPrice, netCurrency := priceFields(article, productdata.PriceNet)
		priceUnit := ""
		if price, ok := article.Price(productdata.PriceList); ok && price.Per != nil {
			priceUnit = price.Per.Value.String()
		}
		discountGroup, mainGroup, productGroup := groupFields(article)
		_ = writer.Write([]string{
			article.SupplierArticleNumber,
			article.ShortText(),
			article.LongText(),
			article.Unit.Code,
			listPrice, listCurrency,
			netPrice, netCurrency,
			priceUnit,
			discountGroup, mainGroup, productGroup,
			article.GTIN,
			article.Matchcode,
			article.ManufacturerArticleNumber,
			article.Type,
			string(article.Status),
		})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		writeln(stderr, "error:", err)
		return 1
	}
	return 0
}

func loadCatalog(paths []string, delivery bool, options []datanorm.Option) (*productdata.Catalog, error) {
	if delivery || len(paths) > 1 {
		return datanorm.Load(context.Background(), paths, options...)
	}
	file, err := datanorm.Open(paths[0], options...)
	if err != nil {
		return nil, err
	}
	return file.Catalog()
}

func commandJSON(stdout, stderr io.Writer, paths []string, delivery bool, options []datanorm.Option) int {
	catalog, err := loadCatalog(paths, delivery, options)
	if err != nil {
		writeln(stderr, "error:", err)
		return 1
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(catalog); err != nil {
		writeln(stderr, "error:", err)
		return 1
	}
	return 0
}

func commandEncodings(stdout, stderr io.Writer, paths []string, options []datanorm.Option) int {
	for _, path := range paths {
		file, err := datanorm.Open(path, options...)
		if err != nil {
			writef(stdout, "%s: error: %v\n", path, err)
			continue
		}
		header := ""
		if file.Header != nil {
			header = fmt.Sprintf("version=%s marker=%s date=%s currency=%s",
				file.Header.Version, file.Header.DataMarker, file.Header.DateRaw, file.Header.Currency)
		}
		writef(stdout, "%s: %s %s\n", path, file.Charset, header)
	}
	return 0
}

func priceFields(article productdata.Article, kind productdata.PriceType) (string, string) {
	price, ok := article.Price(kind)
	if !ok {
		return "", ""
	}
	return price.Amount.Value.String(), string(price.Amount.Currency)
}

func groupFields(article productdata.Article) (discount, main, product string) {
	if group, ok := article.Group(productdata.GroupDiscount); ok {
		discount = group.ID
	}
	if group, ok := article.Group(productdata.GroupMain); ok {
		main = group.ID
	}
	if group, ok := article.Group(productdata.GroupProduct); ok {
		product = group.ID
	}
	return discount, main, product
}

func formatDate(raw string, valid bool, date time.Time) string {
	if valid {
		return date.Format("2006-01-02")
	}
	if raw != "" {
		return raw + " (invalid)"
	}
	return ""
}

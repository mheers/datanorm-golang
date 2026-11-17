// Command inspect prints the header and first articles of a DATANORM file.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/mheers/datanorm-golang"
	"github.com/mheers/datanorm-golang/productdata"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		writef(stderr, "usage: inspect FILE\n")
		return 2
	}
	file, err := datanorm.Open(args[0])
	if err != nil {
		writef(stderr, "error: %v\n", err)
		return 1
	}
	return reportFile(file, stdout, stderr)
}

func reportFile(file *datanorm.File, stdout, stderr io.Writer) int {
	catalog, err := file.Catalog()
	if err != nil {
		writef(stderr, "error: %v\n", err)
		return 1
	}

	writef(stdout, "encoding: %s\n", file.Charset)
	writef(stdout, "version:  %s (%s)\n", file.Header.Version, file.Header.DataType())
	writef(stdout, "date:     %s\n", file.Header.DateRaw)
	writef(stdout, "currency: %s\n", file.Header.Currency)
	writef(stdout, "records:  %d\n", len(file.Records))
	writef(stdout, "articles: %d\n", len(catalog.Articles))
	writef(stdout, "errors:   %d, warnings: %d\n",
		catalog.Diagnostics.Count(productdata.SeverityError),
		catalog.Diagnostics.Count(productdata.SeverityWarning))

	for i, article := range catalog.Articles {
		if i >= 10 {
			break
		}
		price := ""
		if list, ok := article.Price(productdata.PriceList); ok {
			price = list.Amount.String()
		}
		writef(stdout, "%-15s %-60s %s\n", article.SupplierArticleNumber, article.ShortText(), price)
	}
	return 0
}

func writef(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

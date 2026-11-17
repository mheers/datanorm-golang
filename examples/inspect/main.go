// Command inspect prints the header and first articles of a DATANORM file.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/mheers/datanorm-golang"
	"github.com/mheers/datanorm-golang/productdata"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: %s FILE", os.Args[0])
	}
	file, err := datanorm.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	catalog, err := file.Catalog()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("encoding: %s\n", file.Charset)
	fmt.Printf("version:  %s (%s)\n", file.Header.Version, file.Header.DataType())
	fmt.Printf("date:     %s\n", file.Header.DateRaw)
	fmt.Printf("currency: %s\n", file.Header.Currency)
	fmt.Printf("records:  %d\n", len(file.Records))
	fmt.Printf("articles: %d\n", len(catalog.Articles))
	fmt.Printf("errors:   %d, warnings: %d\n",
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
		fmt.Printf("%-15s %-60s %s\n", article.SupplierArticleNumber, article.ShortText(), price)
	}
}

// Command load loads a complete DATANORM delivery and prints a summary.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mheers/datanorm-golang"
)

func main() {
	paths := os.Args[1:]
	if len(paths) == 0 {
		var err error
		paths, err = datanorm.FindDeliveryFiles(".")
		if err != nil {
			log.Fatal(err)
		}
	}
	if len(paths) == 0 {
		log.Fatal("no DATANORM files found")
	}

	catalog, err := datanorm.Load(context.Background(), paths)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("files:     %d\n", len(paths))
	fmt.Printf("articles:  %d\n", len(catalog.Articles))
	fmt.Printf("groups:    %d\n", len(catalog.Groups))
	fmt.Printf("discounts: %d\n", len(catalog.DiscountGroups))
	fmt.Printf("currency:  %s\n", catalog.Header.Currency)
	if catalog.Header.PriceDate != nil {
		fmt.Printf("price date: %s\n", catalog.Header.PriceDate.Format("2006-01-02"))
	}
}

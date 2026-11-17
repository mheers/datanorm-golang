// Command load loads a complete DATANORM delivery and prints a summary.
package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/mheers/datanorm-golang"
)

func main() {
	os.Exit(run(os.Args[1:], ".", os.Stdout, os.Stderr))
}

func run(args []string, dir string, stdout, stderr io.Writer) int {
	paths := args
	if len(paths) == 0 {
		var err error
		paths, err = datanorm.FindDeliveryFiles(dir)
		if err != nil {
			writef(stderr, "error: %v\n", err)
			return 1
		}
	}
	if len(paths) == 0 {
		writef(stderr, "no DATANORM files found\n")
		return 1
	}

	catalog, err := datanorm.Load(context.Background(), paths)
	if err != nil {
		writef(stderr, "error: %v\n", err)
		return 1
	}

	writef(stdout, "files:     %d\n", len(paths))
	writef(stdout, "articles:  %d\n", len(catalog.Articles))
	writef(stdout, "groups:    %d\n", len(catalog.Groups))
	writef(stdout, "discounts: %d\n", len(catalog.DiscountGroups))
	writef(stdout, "currency:  %s\n", catalog.Header.Currency)
	if catalog.Header.PriceDate != nil {
		writef(stdout, "price date: %s\n", catalog.Header.PriceDate.Format("2006-01-02"))
	}
	return 0
}

func writef(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

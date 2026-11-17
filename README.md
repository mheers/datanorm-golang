# datanorm-golang

A Go SDK for **DATANORM 5** — the German B2B exchange format for product master
data, prices, texts, product groups, and discount groups.

The library reads DATANORM deliveries, resolves cross-references, exposes a
canonical product model, validates supplier data, and writes DATANORM files. The
canonical model lives in `productdata` and is designed to host other formats
(IDS Connect 2, Open Masterdata/OMD) later without changing application code.

- **Module:** `github.com/mheers/datanorm-golang`
- **License:** MIT (see [LICENSE](LICENSE))
- **Format reference:** [SPEC.md](SPEC.md) — a detailed, source-backed DATANORM 5 specification
- **Architecture:** [docs/architecture.md](docs/architecture.md)

## Status

| Area | State |
| --- | --- |
| DATANORM 5 article files | Read + write |
| `DATPREIS`, `DATANORM.RAB`, `DATANORM.WRG` | Read + write |
| V, A, B, T, D, E, P, S, R, G records | Typed parsing where the field order is established |
| Z, J, C records | Preserved losslessly; typed extraction is partial |
| DATANORM 4 | Detected and rejected with a clear error |
| DATANORM 3 | Not supported |
| IDS Connect 2, OMD | Planned on the same canonical model |

DATANORM is not an open standard: the authoritative specification is a
commercial book. [SPEC.md](SPEC.md) documents what is publicly known, what was
verified against independent supplier catalogues, and what remains uncertain.
Every record keeps its raw fields, so an undocumented value is never lost.

## Install

```bash
go get github.com/mheers/datanorm-golang
```

Go 1.23 or newer.

## Quick start

### Read one file

```go
package main

import (
	"fmt"
	"log"

	"github.com/mheers/datanorm-golang"
)

func main() {
	file, err := datanorm.Open("DATANORM.001")
	if err != nil {
		log.Fatal(err)
	}
	catalog, err := file.Catalog()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(catalog.Header.Supplier.Name, catalog.Header.PriceDate)
	for _, article := range catalog.Articles {
		fmt.Printf("%-15s %s\n", article.SupplierArticleNumber, article.ShortText())
	}
}
```

### Load a complete delivery

A delivery normally consists of several files. `Load` sorts them into the
recommended processing order, merges groups and discount groups, and applies
price changes.

```go
catalog, err := datanorm.Load(context.Background(), []string{
	"DATANORM.RAB",
	"DATANORM.WRG",
	"DATANORM.001",
	"DATPREIS.001",
})
if err != nil {
	log.Fatal(err)
}
```

Or let the library find the files:

```go
paths, err := datanorm.FindDeliveryFiles("./delivery")
if err != nil {
	log.Fatal(err)
}
catalog, err := datanorm.Load(context.Background(), paths)
```

### Work with articles and prices

```go
for _, article := range catalog.Articles {
	price, ok := article.Price(productdata.PriceList)
	if !ok {
		continue
	}
	per := price.PerQuantity()
	fmt.Printf("%s: %s per %s\n",
		article.SupplierArticleNumber,
		price.Amount,
		per,
	)
}
```

`Amount` is an exact decimal with a currency; the library never uses `float64`
for money. The price unit is preserved: a price of `89.00 EUR` per `100 MTR` is
not silently turned into a price per metre.

### Read a specific character encoding

```go
file, err := datanorm.Open("DATANORM.001", datanorm.WithCharset("cp437"))
```

Without an override, the library scores CP437, CP850, Windows-1252,
ISO-8859-15, ISO-8859-1, and UTF-8 against the byte sample. DOS code pages are
the historical default; CP850 wins ties.

### Validate supplier data

```go
file, err := datanorm.Open("DATANORM.001")
if err != nil {
	log.Fatal(err)
}
for _, diagnostic := range file.Diagnostics {
	fmt.Println(diagnostic)
}
if file.Diagnostics.HasErrors() {
	// reject the import
}
```

Diagnostics carry a severity, a stable code (`dn.ean-checksum`, `dn.price`,
`dn.missing-text`, …), a file, a line, and a field. Real supplier data is messy:
a bad EAN or a missing group is a warning, not a failed import.

### Write a file

```go
encoder := datanorm.NewEncoder(datanorm.WithCharset("cp850"))
var buf bytes.Buffer
err := encoder.Encode(context.Background(), &buf, catalog, productdata.EncodeOptions{})
```

`Encode` writes an article file. Use `EncodeGroups`, `EncodeDiscounts`, and
`EncodePrices` for `DATANORM.WRG`, `DATANORM.RAB`, and `DATPREIS` files.

### Stream records

For very large files, `Scan` yields records without building the catalog:

```go
err := datanorm.Scan(reader, func(record datanorm.Record) error {
	if record.Kind() == datanorm.KindArticle {
		fmt.Println(record.Field(3)) // article number
	}
	return nil
})
```

### Use through the format registry

The root package registers itself with `productdata`:

```go
format, ok := productdata.Get("datanorm")
if !ok {
	log.Fatal("datanorm format not registered")
}
decoder, _ := format.NewDecoder()
catalog, err := decoder.Decode(context.Background(), reader, productdata.DecodeOptions{})
```

## CLI

```bash
go run ./cmd/datanorm info DATANORM.001
go run ./cmd/datanorm info -delivery DATANORM.RAB DATANORM.WRG DATANORM.001 DATPREIS.001
go run ./cmd/datanorm validate -delivery DATANORM.001 DATPREIS.001
go run ./cmd/datanorm articles DATANORM.001 > articles.csv
go run ./cmd/datanorm json -delivery DATANORM.001 DATANORM.RAB DATANORM.WRG > catalog.json
go run ./cmd/datanorm encodings DATANORM.001
```

Build a binary with:

```bash
go build -o bin/datanorm ./cmd/datanorm
```

Flags are per command and come after the command, for example
`datanorm info -delivery …`, not `datanorm -delivery info …`.

## Package layout

```
productdata/   canonical, format-neutral model (decimal money, prices, texts,
               groups, media, logistics, format registry, diagnostics)
charset/       legacy DOS/Windows encoding detection and conversion
datanorm/      root package: DATANORM 5 parser, raw records, catalog assembly,
               delivery loader, encoder, validation
cmd/datanorm/  command-line tool
```

## Supported DATANORM records

| Record | Meaning | Typed fields |
| --- | --- | --- |
| `V` | header | version, data marker, date, currency, info, creator, address |
| `A` | article | all 30 positions documented in [SPEC.md §6.2](SPEC.md#62-a--article-record) |
| `B` | deletion / renumbering | flag, article number, new number, successor |
| `T` | long text | key, type, line number, text |
| `D` | dimension / insertion text | article, line, type, text/key |
| `E` | end record | count, creator |
| `P` | price change | article, price type, unit, price, discount group |
| `S` | product group | main group, product group, descriptions |
| `R` | discount group | group, kind, value, description |
| `G` | media reference | article, reference, type, description, file name (heuristic) |
| `K` | customer control | candidate customer number |
| `Z`, `J`, `C` | surcharges/tiers, sets, services | raw fields preserved; partial typed access |

Unknown record types are preserved as `UnknownRecord` and reported as a
warning.

## Scope and limitations

- **DATANORM 5 only.** Version 4 uses a fixed-width header and a different
  article record. It is detected and rejected with `ErrVersion4`.
- The exact layouts of `Z`, `J`, `C`, and `G` are not publicly specified. The
  library preserves every field and exposes only what is unambiguous.
- The encoder writes article (`A`), text (`T`, `D`), group (`S`), discount
  (`R`), and price (`P`) records plus the `V` and `E` frames. It does not
  synthesize `G`, `Z`, `J`, `C`, or `K` records; those are read as raw data.
- The canonical catalog is loaded into memory. For multi-gigabyte files, use
  `Scan` for record-level streaming.
- The library does not apply discounts, VAT, or customer-specific conditions.
  It exposes list price, net price, price unit, tiers, and discount rates; the
  final price is an application decision.
- The library never downloads media referenced by `G` records.

## Tests

```bash
go test ./...
go test -race ./...
go test -coverpkg=./... -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | tail -1
```

The test suite keeps every package above 90% statement coverage; the aggregate
across all packages is above 94%.

All fixtures are synthetic. No supplier or customer data is included in this
repository; public catalogues were used only to verify field positions and are
not redistributed.

## Roadmap

The `productdata` model and the format registry are the extension points:

1. **IDS Connect 2** — XML decoder/encoder for cart send/receive, article search,
   and product responses.
2. **Open Masterdata (OMD)** — tolerant JSON decoder for the OMD data packages,
   plus an optional HTTP client in a separate package.
3. **DATANORM 4** — optional compatibility decoder if the demand justifies the
   extra field maps.
4. Typed `Z`, `J`, and `C` extraction once reliable public layouts are available.

## License

MIT. Copyright (c) 2026 Marcel Heers, marcel@heers.it.

DATANORM is a trademark of its respective owners. This project is not
affiliated with the DATANORM working group or the Krammer Verlag.

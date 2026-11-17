# Architecture and the format abstraction

This library has two layers:

1. **`productdata`** — a canonical, format-neutral model for product master
   data, prices, groups, media, logistics, and interchanges.
2. **`datanorm`** (module root) — a DATANORM 5 reader/writer that maps the
   legacy record format onto that model, plus a raw record API for lossless
   access.

The split is deliberate. DATANORM is one of several formats in this market:
IDS Connect 2 exchanges carts and article data as XML with wholesaler shops;
Open Masterdata (OMD) exchanges article data as JSON on demand; BMEcat/ETIM
xChange and Open Datacheck cover manufacturer product data. Applications
should not have to rewrite their domain model for each one.

The authoritative DATANORM field reference is [`SPEC.md`](../SPEC.md). This
document describes how the canonical model and the format packages fit
together.

## Goals

- A **single domain model** that can hold the data of DATANORM 5, IDS Connect 2,
  OMD, and future formats without flattening away what matters.
- **Lossless access** to the source: every DATANORM record keeps its raw
  fields, even fields the library does not interpret.
- **Explicit uncertainty**: fields that are not publicly specified are not
  given invented semantics.
- **Streaming where it matters**: record-level streaming for multi-gigabyte
  files; an in-memory catalog when the caller needs cross-references.
- **Diagnostics instead of exceptions**: supplier data is messy; a bad EAN or
  a missing group must not destroy an import.
- **No hidden I/O**: the core works on `io.Reader`/`io.Writer`. Network
  clients, database storage, and ZIP handling belong to the caller or to
  separate packages.

## Non-goals

- Re-implementing every historical DATANORM 3/4 quirk.
- A full ERP price engine. The library exposes list price, net price, price
  unit, tiers, and discount rates. It does not decide a customer's final price.
- Rendering long texts, resolving media URLs, or downloading files.
- Guessing vendor-specific field semantics.

## The canonical model

The central types live in `productdata`:

```
Catalog
├── Header
│   ├── Format, FormatVersion, Encoding
│   ├── CreatedAt, PriceDate
│   ├── Currency
│   └── Supplier
├── Articles []Article
│   ├── identity: supplier number, manufacturer, manufacturer number, GTIN, type
│   ├── descriptions: short, long, dimension, marketing, additional
│   ├── prices: list, net, recommended retail, on request, tiers
│   ├── groups: main product group, product group, discount group
│   ├── logistics: weight, dimensions, packaging, customs tariff, origin
│   ├── media and documents
│   ├── features and classifications (ETIM, DIN 4000, …)
│   ├── associations: alternative, successor, accessory, spare part
│   └── provenance: source format, file, line, raw record
├── Groups []Group
├── DiscountGroups []DiscountGroup
├── Sets []Set
├── Surcharges []Surcharge
└── Diagnostics
```

Important choices:

- **Money is exact.** `Decimal` stores an `int64` unscaled value and a scale.
  `Amount` combines it with a currency. No `float64` anywhere in the money
  path.
- **Quantities are not converted.** A unit keeps its original code and a
  normalised kind. `1000 MMT` is not silently turned into `1 MTR`.
- **Prices are plural.** An article can carry a list price, a net price, a
  recommended retail price, and tiers at the same time. The old "one price per
  article" constraint of DATANORM is a limitation of the file, not of the
  model.
- **Texts carry a kind and a language.** DATANORM has short, long, and
  dimension texts; OMD has marketing texts; IDS has position texts.
- **Provenance is first-class.** `Article.Source` and `Header.Source` say where
  the data came from. Diagnostics can point at a file and line.
- **`Extra` maps** are the escape hatch for format- or vendor-specific values.
  They are never used to hide a field that has a proper home.

### Message kinds

A catalog is only one kind of interchange. The model is designed so the others
fit without a parallel universe of types:

| Message | Used by | Canonical shape |
| --- | --- | --- |
| Catalog / price list | DATANORM, BMEcat, OMD bulk | `Catalog` with `Header` and `Articles` |
| Single product response | OMD | `Catalog` with one article, or a dedicated `ProductResponse` wrapper |
| Availability | OMD, IDS | `Availability` attached to an article or position |
| Cart / basket | IDS Connect 2 | `Basket` with `BasketPosition`s referencing articles |
| Order / confirmation | IDS Connect 2 | future; same position model with quantities and references |

The first release implements `Catalog` and the supporting types. The
interfaces and model are shaped so `Basket` and `Availability` can be added
without breaking DATANORM.

## Mapping the formats

### DATANORM 5 → canonical

| DATANORM | Canonical |
| --- | --- |
| `V` header | `Header` (format, version, date, currency, supplier, info lines) |
| `A` article | `Article` |
| A field 3 | `Article.SupplierArticleNumber` |
| A fields 4–5 | `Article.Texts` of kind `short` |
| A field 6 | `Article.Unit` |
| A fields 7–9 | `Article.Prices` (`PriceType`, `Per`, `Amount`) |
| A field 10 | `GroupRef` level `discount` |
| A field 11 | `GroupRef` level `main` |
| A field 12 | `GroupRef` level `product` |
| A field 13 | `Article.Matchcode` |
| A field 17 | `Article.ManufacturerArticleNumber` (observed) |
| A field 19 | `Article.GTIN` |
| A field 21 | `Article.MinOrderQuantity` |
| A field 22 | catalog page (stored as extra) |
| A field 23 | text marker (stored as extra / used to select texts) |
| A field 24 | `Article.LongTextKey`; resolved to a `long` text |
| A field 29 | cost type (stored as extra) |
| A field 30 | packaging quantity |
| `T` records | `Text` kind `long`, keyed by `T` key |
| `D` records | `Text` kind `dimension`, keyed by article number |
| `B` records | `Article.Status` / replacement association |
| `S` records | `Group` (`main`/`product`) |
| `R` records | `DiscountGroup` with `Discount` entries |
| `Z` records | raw record preserved; typed surcharge/tier mapping is planned |
| `J` records | raw record preserved; typed set mapping is planned |
| `G` records | `Media` on the article |
| `C` records | raw; trade-specific service data |
| `K` records | customer-number candidate on the raw record |
| `E` record | `Header.RecordCount`, completeness diagnostic |

### IDS Connect 2 → canonical (planned)

IDS Connect 2 is an XML interchange between craft-business software and
wholesaler shops. It is message-oriented: cart send, cart receive, article
search, deep link, heating label, login information, interface version.

| IDS Connect 2 | Canonical |
| --- | --- |
| Article number / manufacturer number / GTIN | `Article` identity |
| Description / long text | `Text` kinds `short`, `long` |
| Quantity + unit | `Quantity` |
| List price / net price / recommended price | `Price` types |
| Discounts | `Discount` / `DiscountGroup` |
| Availability / delivery time | `Availability` |
| Position references | `BasketPosition` references |
| Feature list | `Feature` / `Classification` |
| Documents and images | `Media` |
| Cart | `Basket` |
| Heating label | a specialized message; mapped later |

IDS is not a batch catalogue format, so its decoder should produce `Basket`
and `ProductResponse` values, not pretend to be a `Catalog`. The shared
`Article`, `Price`, `Quantity`, and `Text` types are what make the formats
interoperable.

### Open Masterdata → canonical (planned)

OMD is a REST/JSON standard: one article lookup returns a product with data
packages such as `basic`, `additional`, `prices`, `descriptions`, `logistics`,
`pictures`, and `documents`.

| OMD | Canonical |
| --- | --- |
| `basic` identity fields | `Article` identity |
| `descriptions` | `Text` |
| `prices` (list, net, RRP, scales, promotional) | `Price`, `PriceTier` |
| `prices.rawMaterial` | `Surcharge` |
| `logistics` | `Logistics`, `PackagingQuantity` |
| `pictures` / `documents` | `Media` |
| `additional.attributes` | `Feature` |
| ETIM classification | `Classification` |
| availability fields | `Availability` |
| successor / alternative / accessory | `Association` |
| spare part lists | `Set` |

OMD is tolerant by necessity: the same JSON field can be a string in one
supplier response and a number in another; dates appear as ISO or `YYYYMMDD`;
some fields are misspelled. A decoder must normalise at the boundary and keep
the raw JSON for audit.

## Format registry

`productdata.Format` describes a format and creates decoders/encoders:

```go
type Format interface {
    Name() string
    Description() string
    Extensions() []string
    Detect(filename string, head []byte) bool
    Capabilities() Capabilities
    NewDecoder() (Decoder, error)
    NewEncoder() (Encoder, error)
}
```

`Capabilities` answers questions such as "can this format represent media?"
or "is it a batch catalogue or a request/response interface?" before a caller
tries to use it.

`productdata.Register` keeps a registry. `productdata.Detect` chooses a format
by filename and content. This is how an application can accept a ZIP of
supplier files without hard-coding DATANORM.

The DATANORM package registers itself on import:

```go
import _ "github.com/mheers/datanorm-golang"
```

or, more explicitly:

```go
format, err := productdata.Get("datanorm")
```

## Streaming versus in-memory

DATANORM files can be several gigabytes. Two APIs are provided:

- **Record streaming** — `datanorm.Scan` yields `Record` values with a scanner
  buffer, no full-file allocation. Ideal for indexing, filtering, or copying.
- **Catalog loading** — `datanorm.Load` parses one or more delivery files and
  assembles referenced texts, groups, and discounts. This is the API most
  applications want, and it holds the catalog in memory.

The raw `File` returned by `datanorm.Open`/`Parse` keeps every record and is
the lossless representation. `File.Catalog()` assembles the canonical model.
This separation means an application can inspect, validate, and re-emit a file
without ever building the canonical catalog.

A future streaming article API can be built on the same record scanner; it is
not required for correctness.

## Error and diagnostic model

- **Fatal errors** are returned as `error` values: unreadable input,
  unsupported version, an unparseable header, an invalid line in strict mode.
- **Recoverable problems** become `productdata.Diagnostic` entries with a
  severity, a stable code, a message, and a location (file, line, record type,
  field). A bad EAN is a warning; an oversized article number is a warning; a
  missing long text is a warning.
- **Strict mode** rejects unknown record types as a parse error. Structural
  warnings remain diagnostics; a caller can reject them by checking
  `Diagnostics.Count(SeverityWarning)`. Strict mode is off by default because
  real supplier data would otherwise never import.

Stable diagnostic codes (examples): `dn.header-date`, `dn.article-number`,
`dn.short-text`, `dn.unit`, `dn.text-marker`, `dn.field-count`,
`dn.processing-flag`, `dn.price-fields`, `dn.price`, `dn.price-unit`,
`dn.ean-checksum`, `dn.duplicate-article`, `dn.missing-text`,
`dn.text-line-duplicate`, `dn.end-count`, `dn.discount-value`,
`dn.unknown-record`, `dn.renumber-collision`.

## Adding a new format

1. Implement `productdata.Format` in a new package.
2. Map the format's data into `productdata.Catalog`, `Article`, `Price`,
   `Text`, and the other canonical types.
3. Preserve format-specific raw data in `Article.Extra`/`Article.Raw` or a
   parallel raw model.
4. Emit diagnostics with stable codes.
5. Register the format in an `init` function or let the caller register it
   explicitly.
6. Add round-trip and golden tests with synthetic fixtures. Never commit
   supplier data.

Nothing in `productdata` imports `datanorm`, and nothing in `datanorm` imports
a future `idsconnect` or `openmasterdata` package. The dependency direction is
always: format package → canonical model.

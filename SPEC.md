# DATANORM 5.0 — Specification and Field Reference

**Document status:** community reconstruction, revision 2026-11-17.
**Scope:** DATANORM version 5.0, file marker `V;050;`.
**Normative keywords:** MUST, MUST NOT, SHOULD, SHOULD NOT, MAY are used as in RFC 2119.

This document describes the format as precisely as the public record permits. The
authoritative DATANORM 5 book (Krammer Verlag, 1999, ISBN 3-88382-071-7) and the
commercial DATANORM Developer Guide are not freely available. This document therefore
separates:

- **Specified** — stated by the official DATANORM change notes, published field-length
  tables, or consistent vendor documentation.
- **Observed** — verified against independent public DATANORM 5 catalogues. The
  catalogues used are listed in §13; none of their data is included in this repository.
- **Inferred** — follows from the version 4 field set, the version 5 merge rules, and
  the observed position, but has not been seen filled in a public file.
- **Unknown** — no public description; the field is preserved verbatim.

A reader MUST NOT treat "inferred" as normative. A reader SHOULD preserve every raw
field so that a correction never requires re-reading a supplier file.

---

## 1. Conformance

A file conforms to this specification if:

1. it is a text file using one of the encodings in §3.2;
2. its first record is a `V` record with version `050`;
3. every record starts with a one-character record type;
4. fields are separated by `;` and contain no `;` themselves;
5. referenced data (texts, groups) is internally consistent, or the inconsistency is
   reported as a diagnostic;
6. a reader MAY accept files without an `E` record, but SHOULD warn.

A reader MUST NOT silently interpret a version 4 file as version 5. Version 4 has a
fixed-width 128-character header without semicolons and a different article record.

---

## 2. Terminology

| Term | Meaning |
| --- | --- |
| Record / Satz | One line of a DATANORM file |
| Record type / Satzart | First character of a record |
| Field | One semicolon-separated value inside a record |
| Processing flag / Verarbeitungskennzeichen | `N`, `A`, `L`, or `X` |
| Price type / Preiskennzeichen | 1, 2, 3, 4, 5, or 9 |
| Price unit / Preiseinheit | Number of quantity units a price refers to |
| Text marker / Textkennzeichen | 0–6, selects which text kinds are printed |
| Main group / Hauptwarengruppe | Coarse product group, max 3 characters |
| Product group / Warengruppe | Finer product group, max 10 characters |
| Discount group / Rabattgruppe | Commercial key into the `R` records |
| Long text / Langtext | Text block referenced by a key, stored in `T` records |
| Dimension text / Dimensionstext | Article-specific text lines, stored in `D` records |
| Delta delivery / Änderungslieferung | Delivery containing only changes |

---

## 3. Physical layer

### 3.1 File structure

A DATANORM file is a sequence of records, one per line. There is no file header
other than the `V` record and no trailer other than the `E` record. Records are
positional within the file only in the sense that the `V` record MUST come first and
the `E` record, if present, SHOULD come last. All other records MAY appear in any
order; cross-references are by key, not by position.

### 3.2 Character encoding

There is no encoding declaration. The standard names code page 850; public files also
use:

| Encoding | Typical marker |
| --- | --- |
| CP850 | German umlauts as DOS bytes; default for older exports |
| CP437 | Also common; differs from CP850 on several accented letters |
| Windows-1252 | Newer Windows exports |
| ISO-8859-15 | Occasional modern exports (€ sign) |
| ISO-8859-1 | Occasional |
| UTF-8 | Modern exports, sometimes with BOM |

A reader MUST decode with the selected encoding and MUST report the selection. A reader
SHOULD auto-detect by scoring candidate decodings (see §11) and SHOULD allow an explicit
override.

### 3.3 Field separation

- Fields are separated by a single semicolon `;`.
- There is no quoting, escaping, or doubling.
- A semicolon inside a value is not representable. If a record contains more fields
  than its layout allows, the extra fields MUST be preserved and a diagnostic emitted;
  a reader MUST NOT silently shift them into named fields.
- Trailing empty fields MAY be omitted. A record with fewer fields than the maximum is
  valid; missing fields are empty.
- A record MAY contain more fields than the layout defines. Extra trailing fields
  MUST be preserved.

### 3.4 Line endings and end of file

- CRLF is the norm. LF-only files occur and MUST be accepted.
- A final line without a line ending MUST be accepted.
- A DOS EOF byte `0x1A` MAY appear after the last record and MUST be ignored.
- Empty lines SHOULD be skipped with a warning; they are not records.
- A UTF-8 BOM at the start of the file MUST be skipped.

### 3.5 Length limits

Lengths are characters after decoding, not bytes. The maxima below are specified by
the published DATANORM 5 field-length table:

| Field | Max |
| --- | --- |
| Processing flag | 1 |
| Article number | 15 |
| Short text 1 / 2 | 40 each |
| Quantity unit | 4 |
| Price type | 1 |
| Price unit | 6 |
| Price | 8 integer digits + 2 decimals |
| Discount group | 4 |
| Main product group | 3 |
| Product group | 10 |
| Matchcode | 15 |
| Creator code, alternative article number | 13 |
| Alternative article number | 15 |
| Creator code, manufacturer number | 13 |
| Manufacturer number | 35 |
| Manufacturer type | 35 |
| EAN / GTIN | 18 |
| Media reference number | 15 |
| Minimum order quantity | 5 |
| Catalog page | 15 |
| Text marker | 1 |
| Long-text key | determined by `T` key; typical 15–18 |
| Storage flag | 1 |
| Creator code, reference number | 13 |
| Reference number | 17 |
| VAT marker | 1 |
| Cost type | 2 |
| Packaging quantity | 15 (inferred) |
| Long text line | 40 |
| Long text block | 99 lines |

Over-length values are common. A reader SHOULD warn and MUST NOT truncate silently.

---

## 4. File types and naming

| File pattern | Content | Record types | `V` data marker |
| --- | --- | --- | --- |
| `DATANORM.001` … `DATANORM.999` | Article master data | `V`, `A`, `B`, `T`, `D`, `G`, `Z`, `C`, `J`, `E` | `A` |
| `DATPREIS.001` … `DATPREIS.999` | Price changes | `V`, `P`, `E` | `P` (inferred) |
| `DATASETS.001` … `DATASETS.999` | Sets / bills of materials | `V`, `J`, `E` | `J` (inferred) |
| `DATANORM.RAB` | Discount groups | `V`, `R`, `E` | `R` |
| `DATANORM.WRG` | Product groups | `V`, `S`, `E` | `S` |
| `DATATEXT.001` … `DATATEXT.999` | Stand-alone long texts | `V`, `T`, `E` | `T` (inferred) |

Observed `V` data markers: `A` (article file), `S` (WRG), `R` (RAB). The markers for
price, set, and text files are inferred from the record type.

The numeric extension is a sequence number, not a version. Files of one delivery MUST
be processed in ascending order. A delivery commonly also includes a `DATANORM.RAB`
and a `DATANORM.WRG`; those SHOULD be processed before the article files.

The `V` marker is the authoritative file-type hint. A reader SHOULD use it before
guessing from the file name, because suppliers rename files.

---

## 5. Record types

| Type | German name | Purpose | Section |
| --- | --- | --- | --- |
| `V` | Vorlaufsatz | Header | §6.1 |
| `A` | Artikelhauptsatz | Article master record | §6.2 |
| `B` | Artikelneben-/Änderungssatz | Deletion, discontinuation, renumbering | §6.3 |
| `T` | Langtextsatz | Shared/referenced long text | §6.4 |
| `D` | Dimensionstextsatz | Article-specific text / text insertion | §6.5 |
| `E` | Endesatz | End of file and record count | §6.6 |
| `P` | Preisänderungssatz | Price change | §6.7 |
| `S` | Warengruppensatz | Product group | §6.8 |
| `R` | Rabattsatz | Discount group | §6.9 |
| `Z` | Zu-/Abschlagssatz | Tier prices, raw-material surcharges, special costs | §6.10 |
| `J` | Artikel-Set-Satz | Set / bill of materials | §6.11 |
| `C` | Leistungssatz | Service, time, tender text, material data | §6.12 |
| `G` | Grafik-Anbindungssatz | Media reference | §6.13 |
| `K` | Kontrollsatz | Customer number control | §6.14 |

`N`, `A`, `L`, `X` are not record types. They are processing flags inside records.
`E` means an insertion-text record in versions 3 and 4; in version 5 it is the end
record.

A reader MUST preserve unknown record types and SHOULD emit a diagnostic. A reader
MUST NOT guess an unknown type from its content.

---

## 6. Record layouts

Fields are numbered from 1. The first field is the record type. "M" means mandatory,
"O" optional, "C" conditional. Length is the specified maximum.

### 6.1 `V` — header

Observed physical shape (16 fields in the public files; trailing fields omitted):

```
V;050;A;20260115;EUR;Example GmbH Preisliste 2026;freibleibend;Example GmbH;www.example.com;info@example.com;Musterstraße 1;DE;12345;Musterstadt;;
```

| # | Field | M/O | Length | Confidence | Notes |
| --- | --- | --- | --- | --- | --- |
| 1 | Record type | M | 1 | specified | `V` |
| 2 | Version | M | 3 | specified | `050` |
| 3 | Data marker | M | 1 | specified | `A` articles, `S` groups, `R` discounts, inferred `P` prices, `J` sets, `T` texts |
| 4 | Date | M | 8 | specified | `YYYYMMDD` (price date / creation date) |
| 5 | Currency | M | 3 | specified | ISO 4217, e.g. `EUR` |
| 6 | Info line 1 | O | 40 | specified | File description or supplier name |
| 7 | Info line 2 | O | 40 | specified | Free text |
| 8 | Info line 3 | O | 40 | specified | Free text |
| 9 | Data content / copyright | O | 40 | specified | Free text |
| 10 | Data creator / copyright | O | 40 | specified | Free text |
| 11 | Creator code | O | 13 | specified | Short code of the data creator |
| 12–14 | Address lines 1–3 | O | 30 each | specified | Company/address; usage varies |
| 15 | Street | O | 30 | specified | |
| 16 | Country | O | 3 | specified | ISO 3166-1 alpha-2 |
| 17 | Postal code | O | 9 | specified | May exceed 9 in practice |
| 18 | City | O | 30 | specified | |

**Important:** the field-length table gives the logical order above, but public files
do not use the address block consistently. In particular, some exporters put website,
email, and address lines into fields 9–15. A reader MUST expose fields 6–18 verbatim
and MUST NOT rely on the address interpretation. For the canonical supplier, use field
6 as the description, fields 6–8 as info lines, field 9 as copyright, and fields 10+
as raw sender data; a caller MAY override the supplier.

The date is `YYYYMMDD` in version 5. A reader MAY accept `DDMMYYYY` if the first four
digits are not a plausible year, and SHOULD warn.

### 6.2 `A` — article record

The version 5 A record merges the version 4 A and B records. It has 30 fields. The
layout below was verified against public catalogues from three independent suppliers
and cross-checked with the published version 4 field order and the version 5
field-length table.

| # | German name | English name | M/O | Max | Confidence | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | Satzart | Record type | M | 1 | specified | `A` |
| 2 | Verarbeitungskennzeichen | Processing flag | M | 1 | specified | `N`, `A`, `L` (and `X` in some files) |
| 3 | Artikelnummer | Article number | M | 15 | specified | Supplier article number |
| 4 | Kurztext 1 | Short text 1 | M | 40 | specified | |
| 5 | Kurztext 2 | Short text 2 | O | 40 | specified | |
| 6 | Mengeneinheit | Quantity unit | O | 4 | specified | Version 5 uses ISO-style codes (`PCE`, `MTR`, …); German abbreviations still occur |
| 7 | Preiskennzeichen | Price type | C | 1 | specified | 1 list, 2 net, 3 route, 4 recommended retail, 5 guide, 9 on request |
| 8 | Preiseinheit | Price unit | C | 6 | specified | Direct number (`1`, `10`, `100`, …), not the version 4 code |
| 9 | Preis | Price | C | 8+2 | specified | Minor units (cents) or explicit decimal; see §7.4 |
| 10 | Rabattgruppe | Discount group | O | 4 | specified | References `R` field 2 |
| 11 | Hauptwarengruppe | Main product group | O | 3 | specified | References `S` field 2 |
| 12 | Warengruppe | Product group | O | 10 | specified | References `S` field 3 |
| 13 | Matchcode | Matchcode | O | 15 | specified | Search term |
| 14 | Kürzel Ersteller Altern.Artikelnr. | Alt. number creator code | O | 13 | specified | |
| 15 | Alternativartikelnummer | Alternative article number | O | 15 | specified | Observed in a public valve catalogue |
| 16 | Kürzel Ersteller Herstellernr. | Manufacturer number creator code | O | 13 | specified | |
| 17 | Herstellernummer | Manufacturer article number | O | 35 | specified | Observed in a public sanitary catalogue |
| 18 | Herstellertype | Manufacturer type designation | O | 35 | specified | |
| 19 | EAN / GTIN | GTIN | O | 18 | specified | String; leading zeros significant |
| 20 | Grafikanbindungsnummer | Media reference number | O | 15 | specified | Links to `G` records |
| 21 | Mindestverpackungsmenge | Minimum order/packaging quantity | O | 5 | specified | |
| 22 | Katalogseite | Catalog page | O | 15 | specified | Observed |
| 23 | Textkennzeichen | Text marker | O | 1 | specified | 0–6; see §7.2. Verified by exact correlation with `T`/`D` presence |
| 24 | Langtextschlüssel | Long-text key | O | — | specified | References `T` field 3 |
| 25 | Lagermerker | Storage flag | O | 1 | specified | Often empty or a single space |
| 26 | Kürzel Ersteller Referenznr. | Reference number creator code | O | 13 | specified | |
| 27 | Referenznummer | Reference number | O | 17 | specified | |
| 28 | MwSt-Merker | VAT marker | O | 1 | specified | Meaning not publicly documented |
| 29 | Kostenart | Cost type | O | 2 | inferred | Version 4 B field 11; one public file has value `2` |
| 30 | Verpackungsmenge | Packaging quantity | O | 15 | inferred | Version 4 B field 12 |

**Why 29 and 30 are inferred.** The A record is exactly the version 4 A and B field set
merged and reordered, plus the new version 5 fields. Every other position is confirmed
by observation. Fields 29 and 30 are the only version 4 B fields left, and their
positions follow the merge order. Field 29 was observed once with value `2`; field 30
was empty in every public file examined. A reader MUST preserve both.

**Vendor deviations.** Positions 10–12 are the most important to treat as strings, not
numbers: `Bade` is a valid product group, `280` is a valid discount group, and a group
may be empty. Do not assume fixed width or numeric content. Field 20 may contain a
plain number or a compound reference. Field 25 is often a single space.

The order of positions 10–12 follows the version 4 A record (`Rabattgruppe`,
`Hauptwarengruppe`, `Warengruppe`) and the published field-length table. One public
catalogue puts a 4-character value (`Bade`) in the main-group position; vendors exceed
the 3-character main-group maximum in practice. A reader MUST NOT truncate it. The
library reports no error for an over-length group, because doing so would reject
otherwise valid catalogues.

### 6.3 `B` — change, deletion, renumbering

Version 5 uses `B` for deletions, discontinuations, and article-number changes:

```
B;L;12345;;;
B;X;12345;54321;;
```

| # | Field | M/O | Confidence | Notes |
| --- | --- | --- | --- | --- |
| 1 | Record type | M | specified | `B` |
| 2 | Processing flag | M | specified | `L` delete/discontinue, `X` renumber, `A` change |
| 3 | Article number | M | specified | Old/affected number |
| 4 | New article number | C | observed | For `X` |
| 5 | Successor / replacement | C | observed | Public deletion files leave it empty |
| 6+ | Trailing fields | O | unknown | Preserve |

Deletion files are often named `DATANORM.099` or `DATANORM_Löschdatei.001` and contain
only `B;L;…` records plus header and end record. A `B;L` record has the same article
number as the article to remove. A `B;X` record renames an existing article; it does
not create a new one.

### 6.4 `T` — long text

```
T;N;T12345;1;01;First line of the long text;
T;N;T12345;1;02;Second line;
```

| # | Field | M/O | Max | Confidence | Notes |
| --- | --- | --- | --- | --- | --- |
| 1 | Record type | M | 1 | specified | `T` |
| 2 | Processing flag | M | 1 | specified | `N`, `A`, `L` |
| 3 | Text key | M | — | specified | Referenced by A field 24 or D field 7 |
| 4 | Text type | O | — | specified | `1` = normal long text; other values occur |
| 5 | Line number | M | — | specified | Decimal; compare numerically |
| 6 | Text line | O | 40 | specified | |
| 7 | Reserved | O | — | unknown | Usually empty |

Lines of one key are sorted by numeric line number. Duplicate line numbers: the first
wins and a warning is emitted. Empty lines inside a text are meaningful; leading and
trailing empty lines are trimmed when assembling the canonical text.

### 6.5 `D` — dimension / article-specific text

```
D;N;12345;01;F;A free article-specific line;;
D;N;12345;02;T;;T67890;
```

| # | Field | M/O | Max | Confidence | Notes |
| --- | --- | --- | --- | --- | --- |
| 1 | Record type | M | 1 | specified | `D` |
| 2 | Processing flag | M | 1 | specified | `N`, `A`, `L` |
| 3 | Article number | M | 15 | specified | |
| 4 | Line number | M | — | specified | `01` … `99` |
| 5 | Kind | M | 1 | specified | `F` free text, `T` insert text block |
| 6 | Free text | C | 40 | observed | Used for `F` by at least one exporter |
| 7 | Text key / line text | C | 40 | observed | Used for `T` (key); some exporters put `F` text here |
| 8 | Reserved | O | — | unknown | Usually empty |

**Vendor deviation:** public files disagree on whether the free text of an `F` record
lives in field 6 or field 7. A reader MUST accept both: for `F`, take field 7 if
non-empty, otherwise field 6; for `T`, take field 7 if non-empty, otherwise field 6.
This is the only safe interpretation.

`D` texts are keyed by article number, not by a shared text key. A `T` insertion inside
a `D` block expands the referenced `T` block at that position.

### 6.6 `E` — end of file

```
E;387911;Created by example exporter;
```

| # | Field | M/O | Confidence | Notes |
| --- | --- | --- | --- | --- |
| 1 | Record type | M | specified | `E` |
| 2 | Record count | M | specified | Count convention varies; see below |
| 3 | Creator / copyright | O | specified | Free text |
| 4+ | Reserved | O | unknown | Preserve |

The count excludes a trailing `0x1A`. Three conventions occur in public files:

1. all records including `V` and `E` (the article files examined),
2. all records except `V`,
3. only the data records, excluding `V` and `E` (converted `WRG`/`RAB` files examined).

A reader MUST accept all three and SHOULD report any other mismatch as a warning,
not a fatal error. Some exporters get the count wrong entirely.

### 6.7 `P` — price change

Version 5 uses one article per line. Version 4 packed up to three article blocks per
line. The exact version 5 layout is not publicly documented; two independent parser
projects use the shape below, and it is consistent with the version 5 merge rules:

```
P;12345;1;1;129900;06;;;;
P;12345;2;1;99900;;;;;
```

| # | Field | M/O | Confidence | Notes |
| --- | --- | --- | --- | --- |
| 1 | Record type | M | specified | `P` |
| 2 | Article number | M | observed | No processing flag in most version 5 files |
| 3 | Price type | M | observed | 1, 2, 3, 4, 5, 9 |
| 4 | Price unit | M | observed | Direct number |
| 5 | Price | M | observed | Minor units or explicit decimal |
| 6 | Discount group | O | observed | May be a group code or a trailing condition |
| 7+ | Discounts / validity / vendor fields | O | unknown | Preserve |

Some files still put a processing flag in field 2 (version 4 style: `P;A;12345;2;…`).
A reader MUST detect a one-character flag `N`/`A`/`L`/`P` in field 2 and shift the
interpretation by one field. A reader MUST NOT guess whether field 6 is a discount
group or a discount rate when it is numeric; expose it raw and let the caller decide.

The authoritative rule from the DATANORM FAQ: if any of price type, price unit, or
price changes, all three MUST be present. A reader SHOULD warn if only one or two are
present.

### 6.8 `S` — product group (`DATANORM.WRG`)

Version 5 (observed):

```
S;110;;VELUX-Flachdach-Fenster;
S;210;210;VELUX-Dachfenster-Zubehör;
```

| # | Field | M/O | Max | Confidence | Notes |
| --- | --- | --- | --- | --- | --- |
| 1 | Record type | M | 1 | specified | `S` |
| 2 | Main product group | M | 3 | specified | |
| 3 | Product group | O | 10 | specified | May be empty |
| 4 | Description | O | 40 | specified | |
| 5 | Description of the main group | O | 40 | observed | Occurs in some files |
| 6+ | Reserved | O | — | unknown | Preserve |

There is **no processing flag** in version 5 `S` records. Version 4 used `S;;main;sub;…`
with an empty second field. A version 5 reader MUST treat field 2 as the main group,
not as a flag. A `S` record from a version 4 file would only be seen after a version 4
header, which is rejected.

### 6.9 `R` — discount group (`DATANORM.RAB`)

Version 5 (observed):

```
R;110;1;0;VELUX-Flachdach-Fenster;
R;210;2;0.85;VELUX-Zubehör;
```

| # | Field | M/O | Max | Confidence | Notes |
| --- | --- | --- | --- | --- | --- |
| 1 | Record type | M | 1 | specified | `R` |
| 2 | Discount group | M | 4 | specified | Referenced by A field 10 |
| 3 | Kind | M | 1 | specified | 1 percent, 2 multiplier, 3 surcharge/increase |
| 4 | Value | M | 15 | specified | Percent, factor, or surcharge |
| 5 | Description | O | 40 | specified | |
| 6+ | Further rates | O | — | specified | Applied multiplicatively in order |

There is **no processing flag** in version 5 `R` records. A rate of `0` is common and
means that the actual conditions are agreed outside the file. DATANORM chains multiple
rates multiplicatively: `100 × 0.80 × 0.90 = 72`, not `100 × 0.70 = 70`.

### 6.10 `Z` — tiers, surcharges, special costs

`Z` is a family of records, not one layout. It covers:

- tier prices by quantity, date, or distance,
- raw-material surcharges (copper, silver, aluminium, brass),
- special costs and discounts per text.

The processing/sub-type marker and the article/group reference come first; the rest
depends on the sub-type. The version 4 book describes `Z` as a staffel price and
surcharge record; version 5 merged the surcharge and its processing rule. Public
implementations support at least:

| Sub-type | Meaning |
| --- | --- |
| Tier by quantity | `from` quantity, `to` quantity, price/percent |
| Raw-material surcharge | material, weight basis, weight, price basis, quotation |
| Special cost | text key, amount/percent |

This document does not invent a universal layout. A reader MUST preserve every field.
A reader MAY expose the tier shape when the sub-type marker and field count are
unambiguous. A writer MUST NOT emit `Z` unless the caller supplies the raw fields.

### 6.11 `J` — sets / bills of materials

`J` records describe a set article and its components. Multiple `J` records form one
set; a set is always delivered complete, never incrementally. Fields include the set
article number, component article number, quantity, unit, call-off/optional flags, and
text. The layout depends on the sub-type. A reader MUST preserve the raw records; a
reader MAY expose the set and component numbers and quantities when they are
unambiguous. Application-specific composition belongs to the caller.

### 6.12 `C` — services, tender texts, times, material data

`C` records carry trade-specific service data: tender text blocks, assembly and
disassembly times, workshop times, material data (weight, surface, cross-section), and
HVAC data. Sub-types are encoded inside the record. This document does not define a
universal layout. A reader MUST preserve the raw records and SHOULD expose the
sub-type marker and article/set reference.

### 6.13 `G` — media reference

Version 5 only. A `G` record links an article to an external file. The official change
notes state:

- up to 99 references per article,
- every reference carries a media type (image, CAD drawing, video, audio),
- every reference carries a short description,
- the file itself is not embedded.

The field order is not publicly documented. Observed vendor documentation and the
field-length table indicate fields for:

| Meaning | Max |
| --- | --- |
| Record type | 1 |
| Processing flag | 1 |
| Article number | 15 |
| Media reference number (matches A field 20) | 15 |
| Media type | 2 |
| Description | 40 |
| File name | 8.3 (traditionally), longer in modern exports |

A reader MUST preserve all fields. A reader SHOULD expose the article number, the media
reference number, the file name, and the media type using positional heuristics, and
SHOULD mark those values as heuristic. Files live next to the article file, traditionally
in a `D5BILD` subdirectory; strict implementations use DOS 8.3 names.

### 6.14 `K` — customer-number control

`K` carries the customer number for which the delivery was produced. It is a control
record and does not alter article data. The exact layout is not publicly documented.
A reader MUST preserve the record and SHOULD expose the first non-empty value as the
customer number candidate.

---

## 7. Semantics

### 7.1 Processing flags

| Flag | Meaning | Correct handling |
| --- | --- | --- |
| `N` | New | Create or replace the article |
| `A` | Change | Update existing; missing fields keep prior values in a delta |
| `L` | Delete / discontinue | Mark as deleted; do not blindly hard-delete history |
| `X` | Renumber | Version 5 uses a `B` record; old number → new number |

A full delivery and a delta delivery use the same layout. The receiver MUST know which
one it has. Replacing a catalogue with a delta is data loss. A full delivery SHOULD be
imported into an empty catalogue; a delta SHOULD be applied to an existing one.

### 7.2 Text markers

The text marker (A field 23) selects which text kinds an application should print. The
practical mapping published by a DATANORM importer is:

| Marker | Short 1 | Short 2 | Long (`T`) | Dimension (`D`) |
| --- | --- | --- | --- | --- |
| 0 | yes | yes | no | no |
| 1 | yes | no | yes | no |
| 2 | yes | no | no | yes |
| 3 | no | no | yes | yes |
| 4 | yes | yes | yes | no |
| 5 | yes | yes | no | yes |
| 6 | yes | yes | yes | yes |

**Verification:** in a 25,648-article public catalogue, 18,186 articles had marker 0 and
neither `T` nor `D`; 6,608 had marker 4 and a `T` key but no `D`; 854 had marker 6, a
`T` key, and exactly 854 `D` articles. This is a perfect correlation and makes the
marker semantics effectively certain. Another public catalogue showed the same pattern
for markers 0, 4, and 5.

Some sources describe marker 1/2 slightly differently. The library exposes the marker
verbatim; applications that print texts SHOULD use the table above and MAY allow an
override.

### 7.3 Price types

| Code | German | English | Notes |
| --- | --- | --- | --- |
| 1 | Listenpreis | List price | Gross list price before customer discounts |
| 2 | Nettopreis | Net price | Customer's buying price; discounts already applied |
| 3 | Streckenpreis | Route/distance price | Version 5; only in `P` records; no discounts |
| 4 | empfohlener Verkaufspreis | Recommended retail price | Version 5; only in `P` records; no discounts |
| 5 | Richtpreis | Guide/quotation price | Version 5; sources are inconsistent with the optional guide price under code 9 |
| 9 | Preis auf Anfrage | Price on request | A supplied amount, if any, is a non-binding guide |

`Brutto`/`Netto` in DATANORM does **not** mean VAT-inclusive/VAT-exclusive. Prices are
always net of VAT. `Brutto` means list price before discounts; `Netto` means buying
price after discounts.

The official FAQ: if price type, price unit, or price changes, all three MUST be
present.

### 7.4 Price representation

Two representations occur in version 5:

1. **Minor units** — `129900` means `1299.00` in a two-decimal currency. This is the
   traditional and most common form.
2. **Explicit decimal** — `1299,00` or `1299.00` means `1299.00`. The decimal
   separator makes the major-unit interpretation explicit.

A reader MUST use this rule:

- if the value contains `,` or `.`, parse it as a decimal number in major units;
- otherwise parse it as an integer in minor units (scale 2 for EUR).

A reader MUST NOT apply both interpretations. A writer SHOULD write minor units unless
the caller explicitly requests decimals.

The price unit (A field 8, P field 4) is a direct number: `1`, `10`, `100`, `1000`, or
any positive integer in version 5. Version 4 encoded it as 0–3 for 1/10/100/1000. A
reader MUST NOT map the version 5 value through the version 4 table.

### 7.5 Discounts

- A discount group is referenced by A field 10 and defined by `R`.
- `R` field 3 distinguishes percent (1), multiplier (2), and surcharge/increase (3).
- Multiple rates are applied in order, multiplicatively.
- A missing `R` file means only the group code is known; a zero rate is valid.
- A discount group is a commercial key, not a product category.

### 7.6 Product groups

- Main product group (A field 11, `S` field 2) max 3 characters.
- Product group (A field 12, `S` field 3) max 10 characters.
- A `S` record may describe a main group with an empty sub-group, or a sub-group.
- Group codes are opaque strings. A missing `S` file means only codes are known.
- A group code may be numeric or alphabetic (`110`, `Bade`, `EPL`).

### 7.7 Units

Version 5 restricts the free-form units of version 4, but both ISO-style and German
abbreviations occur. A reader SHOULD preserve the original code and SHOULD provide a
normalized kind without converting quantities.

| Meaning | Common codes |
| --- | --- |
| Piece | `PCE`, `C62`, `H87`, `STK`, `ST`, `Stck`, `Stück` |
| Pair | `PR`, `PRS`, `Paar`, `PA` |
| Metre | `MTR`, `m`, `lfm` |
| Square metre | `MTK`, `m2`, `qm`, `QM` |
| Cubic metre | `MTQ`, `m3`, `cbm` |
| Kilogram | `KGM`, `kg` |
| Gram | `GRM`, `g` |
| Litre | `LTR`, `l`, `Liter` |
| Millilitre | `MLT`, `ml` |
| Hour | `HUR`, `Std`, `h` |
| Minute | `MIN`, `Min` |
| Set | `SET`, `Set` |
| Package | `PK`, `PAK`, `Pck`, `Pkg` |
| Roll | `ROL`, `Rolle` |
| Tonne | `TNE`, `t` |

### 7.8 EAN / GTIN

The EAN field is a GTIN-8, GTIN-12, GTIN-13, or GTIN-14. It is a string, not a number:
leading zeros are significant. A reader SHOULD validate the GS1 check digit and emit a
warning on mismatch. A bad EAN MUST NOT reject the article; supplier data frequently
contains one.

### 7.9 Media

`G` records reference files, not embedded data. A reader SHOULD resolve the file name
against the delivery root and, for strict legacy files, against a `D5BILD`
subdirectory. A reader MUST NOT download or open referenced files implicitly.

### 7.10 Sets and bills of materials

A set (`J`) is delivered complete. Optional positions are marked by call-off flags.
A receiver SHOULD replace an existing set with the new complete set, never append
positions. A set member may itself be a set; recursion must be bounded.

### 7.11 Surcharges and tiers

`Z` records carry tier prices and raw-material surcharges. A receiver SHOULD apply the
tier with the highest minimum quantity not exceeding the ordered quantity. Raw-material
surcharges depend on a daily quotation and SHOULD be exposed as data, not silently
added to a price.

---

## 8. Processing model

Recommended order for a delivery:

1. `DATANORM.RAB` — discount groups.
2. `DATANORM.WRG` — product groups.
3. `DATANORM.001` … `DATANORM.999` — articles, texts, media (ascending).
4. `DATPREIS.001` … — price changes (ascending).
5. `DATASETS.001` … — sets (ascending).

Rationale:

- Articles reference discount and product groups.
- Price files update articles that must already exist.
- Sequence numbers order split catalogues.
- Sets may reference articles from all article files.

A full delivery SHOULD replace the article base. A delta SHOULD be applied to the
existing base. Deletions (`B;L`) SHOULD mark rather than purge. Renumbering (`B;X`)
SHOULD preserve the article's history under the new number.

---

## 9. Validation rules

A validating reader SHOULD check:

| Rule | Severity |
| --- | --- |
| First record is `V` | error |
| Version is `050` | error |
| Currency is a 3-letter code | warning |
| Date parses | warning |
| `E` present | warning |
| `E` count matches actual record count | warning |
| Article number present, ≤ 15 | error / warning |
| Short text 1 present | warning |
| Short text ≤ 40 | warning |
| Long text line ≤ 40 | warning |
| Processing flag known | warning |
| Price type known | warning |
| Price parseable | error if a price is required |
| Price unit positive | warning |
| Discount group referenced by an `R` record | warning |
| Main/product group referenced by an `S` record | warning |
| Long-text key referenced by a `T` record | warning |
| `D` insertion target exists | warning |
| EAN check digit | warning |
| Duplicate article number in one file | warning |
| Unknown record type | warning |
| Record with more fields than the layout | warning |
| Semicolon inside a value (field count explodes) | error for that record |

A strict mode MAY turn selected warnings into errors. Strict mode SHOULD NOT turn a bad
EAN, a missing group, or a zero discount rate into an error.

---

## 10. Encoding detection

Recommended algorithm:

1. If the sample starts with a UTF-8 BOM, choose UTF-8.
2. If the sample is valid UTF-8 and contains at least one multi-byte sequence, choose
   UTF-8.
3. Otherwise score CP437, CP850, Windows-1252, ISO-8859-15, and ISO-8859-1:
   - decode the sample;
   - award points for German letters and common German words;
   - penalize control characters, box-drawing characters, and the replacement
     character;
   - prefer CP850 on a tie (the standard's stated code page);
4. If no candidate is plausible, choose CP850 and warn.

CP437 and CP850 differ exactly on several accented letters (`ä`, `ö`, `ü`, `é`, …).
A wrong choice corrupts product names. The caller MUST be able to override the result.

---

## 11. Vendor deviations observed

| Area | Deviation | Handling |
| --- | --- | --- |
| Character set | CP437, CP850, CP1252, ISO-8859-15, UTF-8 | detect + override |
| `V` address block | Website/email/address in different fields | expose raw; do not trust |
| A field 10/11/12 | Numeric or alphabetic groups; empty | treat as strings |
| A field 20 | Plain number or compound reference | preserve |
| A field 25 | Empty or a single space | trim for semantics |
| A field 29/30 | Rarely used | preserve |
| `D` free text | Field 6 or field 7 | accept both |
| `T` text type | Always `1` in the observed files; other values possible | preserve |
| `P` layout | Version 5 shape vs version 4 flag and multi-article blocks | detect flag; preserve tail |
| `S`/`R` | Version 5 has no processing flag; version 4 has an empty field 2 | version 5 only |
| `E` count | Includes `E` in the observed files; some exporters differ | warn, continue |
| `0x1A` | Present in most DOS files | ignore |
| Line endings | CRLF and LF | accept both |
| Trailing fields | Omitted or extra | preserve |

---

## 12. Open questions

The following are not publicly specified and are treated conservatively:

1. Exact `P` layout in version 5 (flag vs no flag, discount group position).
2. Exact `G` layout and media-type codes.
3. Exact `Z` sub-type layouts and tier/raw-material field order.
4. Exact `J` sub-type layouts, call-off flags, and nested-set rules.
5. Exact `C` sub-types (tender texts, times, material data, HVAC).
6. Meaning of A field 28 (VAT marker) and field 25 (storage flag).
7. Meaning of A field 29/30 beyond the version 4 merge inference.
8. `K` layout.
9. `V` address block beyond the field-length table.
10. Whether text marker 5/6 always imply a `D` block, or whether the block may be
    absent while the marker still requests it.

The library preserves the raw fields for all of these. Contributions with public,
non-customer sample files are welcome.

---

## 13. Evidence and sources

### 13.1 Public DATANORM 5 catalogues used for observation

Only structural observations were used; no catalogue data is included in this
repository or its tests.

| Source | What it confirmed |
| --- | --- |
| VELUX DATANORM 5 complete catalogue (25,648 articles, 361,407 text records, 854 dimension articles), plus `DATANORM.WRG` and `DATANORM.RAB` | 30-field A layout, fields 10/11/12, field 23 text marker (perfect correlation), field 24 text key, `E` count, `S`/`R` no-flag layout, deletion `B;L` |
| Nordholm DATANORM 5 sanitary catalogue (456 articles, 136 dimension records, 17 text records) | A fields 10–12, 13 matchcode, 17 manufacturer number, 19 EAN, 21 minimum quantity, 23 text marker 0/4/5, 24 text key, 25 space, 29 value `2` |
| KS Licht DATANORM 5 catalogue (4,708 articles, 210,636 text records, 126 dimension records) | A fields 10/11, 21, 22 catalog page, 23 text marker 3, 24 text key, `T`/`D` layout |
| Public valve manufacturer A-record snippet | A fields 13 matchcode, 15 alternative article number, 19 EAN, 23 text marker 0 |
| Solvis DATANORM 4 catalogue and RAB/WRG files | Version 4 A/B merge order; `R`/`S` field order; DN4 header and processing flags |
| KSB and ABN catalogues | Version 4 counter-examples (fixed-width header), used to reject version 4 |

### 13.2 Published documentation

- datanorm.de, *DATANORM 5* — official change notes: A/B merge, B only for deletions
  and renumbering, direct price unit, media reference block, VAT marker, new price
  types, standardised units, customs/dangerous-goods/tile/DIN 4000 additions.
- datanorm.de, *DATANORM Versionen* — file names, record types, version matrix,
  character-set restrictions, 2026 status note.
- datanorm.de, *Anfragen* — price type/unit/price must be sent together.
- datanorm.de, *DATANORM Bücher* — book and developer-guide metadata (ISBN
  3-88382-071-7; 1999; not freely available).
- Mohr IT, *Maximale Feldlängen* — DATANORM 4/5 maximum field lengths and logical field
  order.
- DMS Systemhaus, *Datanorm Schnittstelle für Microsoft Dynamics 365 Business Central*
  — text markers 0–6, import/export behaviour.
- comtech, *Datanorm-Datei* — DATANORM 4 A/B/P field descriptions and R/S semantics.
- 4master, *Datanorm* — processing flags, file order, P records.
- EULANDA, *Datanorm-Schnittstelle* — file names, import order, price/rounding
  behaviour.
- Solutioo, *Datanorm Integration* — version detection, record types, price types,
  multiplicative discounts, staging model.
- nextPIM, *DATANORM-Import* — practical field mapping, group/media handling.
- ITEK/DGH, *IDS Connect 2.5* and Open Masterdata documentation — used for the
  format-neutral architecture, not for DATANORM field semantics.
- Independent open-source parsers (Ruby `halo/datanorm`, Python `FahrJo/datanorm`,
  C `yNiklas/datanorm`, PowerShell EulandaConnect) — cross-checks for record shapes and
  version 4 behaviour.

### 13.3 Source URLs

- datanorm.de — version 5 change notes: https://www.datanorm.de/DATANORM/DATANORM5
- datanorm.de — version matrix and file types: https://www.datanorm.de/DATANORM/DATANORM-Versionen
- datanorm.de — price type/unit/price rule: https://www.datanorm.de/Hilfe/Anfragen
- datanorm.de — books and developer guide: https://www.datanorm.de/DATANORM/DATANORM-Buecher
- Mohr IT — DATANORM 4/5 maximum field lengths: https://www.mohr-it.de/Hilfe/DatanormStudio/maximalefeldlangen.html
- DMS Systemhaus — Business Central DATANORM documentation, text markers 0–6:
  https://dms-sys.de/wp-content/uploads/2026/04/DATANORM-DOKU_BC_17.0.5.1.pdf
- comtech — DATANORM 4 A/B/P record descriptions: https://hilfe.comtech.at/ce/773/html/datanorm_datei.htm
- 4master — processing flags and file order:
  https://www.4master.de/software/handwerk/OnlineHilfe/datanorm.html
- EULANDA — DATANORM interface, order, price handling:
  https://doc.eulanda.eu/de/docs/Zubeh%C3%B6r/Datanorm-Schnittstelle
- Solutioo — version detection and record type reference:
  https://docs.solutioo.de/datanorm-item-import/einleitung/unterstuetzte-datanorm-versionen
- nextPIM — practical DATANORM import mapping:
  https://knowledge.nextpim.de/datanorm-import
- Qucosa dissertation (2007) — price type codes 1/2/3/4/9 and text marker notes:
  https://ul.qucosa.de/api/qucosa%3A16780/attachment/ATT-0
- VELUX DATANORM 5 catalogue (observation only):
  https://www.velux.de/fachkunden/tools-technik/artikelstammdaten
- Nordholm DATANORM 5 catalogue (observation only):
  https://nordholm.de/de/datanorm
- KS Licht DATANORM 5 catalogue (observation only):
  https://www.ks-licht.ch/media/static/datanorm/SLV/v5/DATANORM.001
- xt:Commerce forum A-record snippet (observation only):
  https://forums.xt-commerce.com/topic/81260-import-datanorm/
- ITEK/DGH — IDS Connect 2.5:
  https://itek.de/wissen/verzeichnis-branchenstandards/ids-connect
- Open Masterdata API:
  https://omd.obeta.de/
- Open Masterdata integration notes (field variance):
  https://github.com/LandrixSoftware/OpenMasterdata-for-Delphi

Public catalogues were fetched during research and were not committed. Only
their structural properties (record shapes, field positions, correlations) are
described here.

### 13.4 Confidence of the A-record table

| Fields | Confidence | Basis |
| --- | --- | --- |
| 1–9 | specified | Official change notes, all public files |
| 10–12 | specified | Version 4 merge order, all public files, RAB/WRG cross-reference |
| 13–22 | specified | Version 4 B fields, field-length table, public files |
| 23 | specified | Perfect correlation with `T`/`D` in two public catalogues; published text-marker table |
| 24 | specified | Public files and `T` key correlation |
| 25–28 | specified | Field-length table and version 4 B merge |
| 29–30 | inferred | Version 4 B fields 11/12; field 29 observed once |

---

## 14. Document history

| Date | Change |
| --- | --- |
| 2026-11-17 | Initial reconstruction. A-record layout corrected after cross-checking four public DATANORM 5 catalogues: field 10 is the discount group, 11 the main group, 12 the product group, 23 the text marker, 24 the long-text key, 29 the cost type, 30 the packaging quantity. |

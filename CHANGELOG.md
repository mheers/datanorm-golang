# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
semantic versioning.

## [0.1.0] — 2026-11-17

### Added

- Canonical `productdata` model: exact decimals, currencies, prices and tiers,
  texts, groups, discount groups, articles, media, features, logistics,
  associations, sets, surcharges, diagnostics, and a format registry.
- `charset` package: detection and conversion for CP437, CP850, Windows-1252,
  ISO-8859-15, ISO-8859-1, and UTF-8.
- DATANORM 5 parser for `V`, `A`, `B`, `T`, `D`, `E`, `P`, `S`, `R`, `G`, `K`,
  and unknown records.
- Symbolic text keys, dimension texts with `T`-block insertion, product groups,
  discount groups, media references, deletions, and renumbering.
- Delivery loader with role ordering, merging, price updates, and group-name
  resolution.
- DATANORM encoder for article, product-group, discount-group, and price files.
- Validation with stable diagnostic codes and a source-backed `SPEC.md`.
- `cmd/datanorm` CLI: `info`, `validate`, `articles`, `json`, `encodings`.
- `examples/inspect` and `examples/load`.
- Synthetic test fixtures only; no supplier or customer data is included.

### Notes

- DATANORM 5 only. DATANORM 4 is detected and rejected with a clear error.
- `Z`, `J`, and `C` records are preserved; typed extraction is partial because
  their layouts are not publicly specified.

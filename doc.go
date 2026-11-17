// Package datanorm reads and writes DATANORM 5.0 product and price data.
//
// DATANORM is a German B2B exchange format for product master data, prices,
// texts, product groups, and discount groups. This package parses the
// semicolon-separated record files (DATANORM.001, DATPREIS.001, DATANORM.RAB,
// DATANORM.WRG, DATASETS.001), resolves cross-references, and exposes a
// canonical model from the sibling productdata package.
//
// The format is not fully publicly specified. The authoritative field reference
// for this implementation is SPEC.md in the repository root. Fields whose
// meaning is not publicly documented are preserved verbatim on the raw records.
//
// Version 5 only. Version 4 files are detected and rejected with ErrVersion4.
package datanorm

// Version is the DATANORM version implemented by this package.
const Version = "5.0"

// LibraryVersion is the semantic version of this library.
const LibraryVersion = "0.1.0"

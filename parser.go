package datanorm

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mheers/datanorm-golang/charset"
	"github.com/mheers/datanorm-golang/productdata"
)

const detectionSampleSize = 32 * 1024

// File is a parsed DATANORM file. It keeps every record in order and exposes
// typed accessors and the canonical catalog.
type File struct {
	Path        string
	Charset     charset.Charset
	Records     []Record
	Header      *HeaderRecord
	End         *EndRecord
	Diagnostics productdata.Diagnostics
}

// KindCount returns the number of records per type.
func (f *File) KindCount() map[Kind]int {
	counts := make(map[Kind]int)
	for _, record := range f.Records {
		counts[record.Kind()]++
	}
	return counts
}

// RecordsOfKind returns all records of the given type.
func (f *File) RecordsOfKind(kind Kind) []Record {
	var out []Record
	for _, record := range f.Records {
		if record.Kind() == kind {
			out = append(out, record)
		}
	}
	return out
}

// Articles returns all article records.
func (f *File) Articles() []*ArticleRecord {
	var out []*ArticleRecord
	for _, record := range f.Records {
		if article, ok := record.(*ArticleRecord); ok {
			out = append(out, article)
		}
	}
	return out
}

// Texts returns all long-text records.
func (f *File) Texts() []*TextRecord {
	var out []*TextRecord
	for _, record := range f.Records {
		if text, ok := record.(*TextRecord); ok {
			out = append(out, text)
		}
	}
	return out
}

// Dimensions returns all dimension-text records.
func (f *File) Dimensions() []*DimensionRecord {
	var out []*DimensionRecord
	for _, record := range f.Records {
		if dimension, ok := record.(*DimensionRecord); ok {
			out = append(out, dimension)
		}
	}
	return out
}

// Groups returns all product-group records.
func (f *File) Groups() []*GroupRecord {
	var out []*GroupRecord
	for _, record := range f.Records {
		if group, ok := record.(*GroupRecord); ok {
			out = append(out, group)
		}
	}
	return out
}

// Discounts returns all discount records.
func (f *File) Discounts() []*DiscountRecord {
	var out []*DiscountRecord
	for _, record := range f.Records {
		if discount, ok := record.(*DiscountRecord); ok {
			out = append(out, discount)
		}
	}
	return out
}

// Changes returns all change records.
func (f *File) Changes() []*ChangeRecord {
	var out []*ChangeRecord
	for _, record := range f.Records {
		if change, ok := record.(*ChangeRecord); ok {
			out = append(out, change)
		}
	}
	return out
}

// Catalog assembles the canonical productdata catalog, resolving text keys,
// groups, and discount groups. B records are applied to the result.
func (f *File) Catalog() (*productdata.Catalog, error) {
	return assembleCatalog(f, true)
}

// catalogWithoutChanges assembles the catalog without applying B records. The
// delivery loader needs this so that deletions and renumbering are applied once,
// after all article and price files have been merged.
func (f *File) catalogWithoutChanges() (*productdata.Catalog, error) {
	return assembleCatalog(f, false)
}

// Parse reads a DATANORM 5 file from r.
func Parse(r io.Reader, opts ...Option) (*File, error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}
	cs, err := o.resolveCharset()
	if err != nil {
		return nil, err
	}
	buffered := bufio.NewReaderSize(r, detectionSampleSize)
	if cs == charset.Auto {
		sample, _ := buffered.Peek(detectionSampleSize)
		cs = charset.Detect(sample)
	}
	decoded, err := charset.NewReader(buffered, cs)
	if err != nil {
		return nil, err
	}
	return parseReader(decoded, cs, o)
}

// Open reads a DATANORM 5 file from disk.
func Open(path string, opts ...Option) (*File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	parsed, err := Parse(file, opts...)
	if err != nil {
		return nil, err
	}
	parsed.Path = path
	for i := range parsed.Diagnostics {
		if parsed.Diagnostics[i].File == "" {
			parsed.Diagnostics[i].File = path
		}
	}
	return parsed, nil
}

// Scan streams records without keeping the whole file. It still parses each
// record, but it does not assemble the catalog. The header is validated before
// the callback is called for any record.
func Scan(r io.Reader, fn func(Record) error, opts ...Option) error {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}
	cs, err := o.resolveCharset()
	if err != nil {
		return err
	}
	buffered := bufio.NewReaderSize(r, detectionSampleSize)
	if cs == charset.Auto {
		sample, _ := buffered.Peek(detectionSampleSize)
		cs = charset.Detect(sample)
	}
	decoded, err := charset.NewReader(buffered, cs)
	if err != nil {
		return err
	}
	return scanReader(decoded, o, fn)
}

func parseReader(r io.Reader, cs charset.Charset, o options) (*File, error) {
	file := &File{Charset: cs}
	var headerSeen bool
	emptyLines := 0

	err := scanReaderInternal(r, o, func(record Record, raw string) error {
		if !headerSeen {
			if raw == "" {
				return nil
			}
			if strings.HasPrefix(raw, "V ") && !strings.Contains(raw, ";") {
				return &ParseError{Line: record.Line(), RecordType: 'V', Err: ErrVersion4}
			}
			header, ok := record.(*HeaderRecord)
			if !ok {
				if raw != "" && raw[0] != 'V' {
					return &ParseError{Line: record.Line(), Err: ErrNotDatanorm}
				}
				return &ParseError{Line: record.Line(), Err: ErrMissingHeader}
			}
			if err := checkVersion(header); err != nil {
				return &ParseError{Line: record.Line(), RecordType: 'V', Value: header.Version, Err: err}
			}
			file.Header = header
			headerSeen = true
		} else if header, ok := record.(*HeaderRecord); ok {
			file.Diagnostics.Add(productdata.SeverityWarning, "dn.duplicate-header",
				"additional V record; the first header wins")
			_ = header
		}

		file.Records = append(file.Records, record)
		switch typed := record.(type) {
		case *EndRecord:
			file.End = typed
		case *UnknownRecord:
			file.Diagnostics.AddAt(productdata.Diagnostic{
				Severity:   productdata.SeverityWarning,
				Code:       "dn.unknown-record",
				Message:    fmt.Sprintf("unknown record type %q", typed.RawKind),
				Line:       typed.Line(),
				RecordType: string(typed.RawKind),
			})
		}
		return nil
	}, func(line int, raw string) {
		if strings.TrimSpace(raw) == "" {
			emptyLines++
		}
	})

	if err != nil {
		return nil, err
	}
	if !headerSeen {
		return nil, &ParseError{Err: ErrMissingHeader}
	}
	if emptyLines > 0 {
		file.Diagnostics.Add(productdata.SeverityInfo, "dn.empty-line",
			fmt.Sprintf("skipped %d empty line(s)", emptyLines))
	}
	if file.End == nil {
		file.Diagnostics.Add(productdata.SeverityWarning, "dn.missing-end",
			"file has no E record; completeness cannot be checked")
	} else if file.End.CountValid && !countMatches(file.End.Count, len(file.Records)) {
		file.Diagnostics.AddAt(productdata.Diagnostic{
			Severity:   productdata.SeverityWarning,
			Code:       "dn.end-count",
			Message:    fmt.Sprintf("E record says %d records, found %d", file.End.Count, len(file.Records)),
			Line:       file.End.Line(),
			RecordType: "E",
		})
	}
	if o.validate {
		file.Diagnostics = append(file.Diagnostics, validateFile(file)...)
	}
	return file, nil
}

func scanReader(r io.Reader, o options, fn func(Record) error) error {
	headerSeen := false
	err := scanReaderInternal(r, o, func(record Record, raw string) error {
		if !headerSeen {
			if strings.HasPrefix(raw, "V ") && !strings.Contains(raw, ";") {
				return &ParseError{Line: record.Line(), RecordType: 'V', Err: ErrVersion4}
			}
			header, ok := record.(*HeaderRecord)
			if !ok {
				if raw != "" && raw[0] != 'V' {
					return &ParseError{Line: record.Line(), Err: ErrNotDatanorm}
				}
				return &ParseError{Line: record.Line(), Err: ErrMissingHeader}
			}
			if err := checkVersion(header); err != nil {
				return &ParseError{Line: record.Line(), RecordType: 'V', Value: header.Version, Err: err}
			}
			headerSeen = true
		}
		return fn(record)
	}, nil)
	if err != nil {
		return err
	}
	if !headerSeen {
		return &ParseError{Err: ErrMissingHeader}
	}
	return nil
}

func scanReaderInternal(
	r io.Reader,
	o options,
	fn func(record Record, raw string) error,
	onLine func(line int, raw string),
) error {
	reader := bufio.NewReaderSize(r, 64*1024)
	lineNumber := 0
	for {
		raw, err := readLine(reader, o.maxLineBytes)
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		if raw == "" && errors.Is(err, io.EOF) {
			return nil
		}
		lineNumber++
		raw = strings.TrimSuffix(raw, "\r")
		if lineNumber == 1 {
			raw = strings.TrimPrefix(raw, "\uFEFF")
		}
		if strings.TrimSpace(raw) == "" {
			if onLine != nil {
				onLine(lineNumber, raw)
			}
			if errors.Is(err, io.EOF) {
				return nil
			}
			continue
		}
		if raw[0] == 0x1A {
			return nil
		}

		fields := strings.Split(raw, ";")
		kind := Kind(raw[0])
		record := parseRecord(kind, lineNumber, raw, fields)
		if !kind.Known() && o.strict {
			return &ParseError{Line: lineNumber, RecordType: raw[0], Err: fmt.Errorf("unknown record type %q", raw[0])}
		}
		if callErr := fn(record, raw); callErr != nil {
			return callErr
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
	}
}

// readLine reads one physical line including its terminator. It strips the
// terminator and returns the line. A final line without a terminator is
// returned together with io.EOF.
func readLine(reader *bufio.Reader, max int) (string, error) {
	var buf []byte
	for {
		chunk, err := reader.ReadSlice('\n')
		if len(buf)+len(chunk) > max {
			return "", ErrLineTooLong
		}
		buf = append(buf, chunk...)
		switch {
		case err == nil:
			return trimLineEnding(string(buf)), nil
		case errors.Is(err, bufio.ErrBufferFull):
			continue
		case errors.Is(err, io.EOF):
			if len(buf) == 0 {
				return "", io.EOF
			}
			return trimLineEnding(string(buf)), io.EOF
		default:
			return "", err
		}
	}
}

func trimLineEnding(line string) string {
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	return line
}

// countMatches accepts the three record-count conventions observed in public
// files: all records including V and E, all records except V, or only the data
// records (excluding V and E).
func countMatches(count, total int) bool {
	return count == total || count == total-1 || count == total-2
}

func checkVersion(header *HeaderRecord) error {
	version := strings.TrimSpace(header.Version)
	switch version {
	case "050", "50", "5", "5.0", "05":
		if version != "050" {
			header.Version = "050"
		}
		return nil
	case "04", "4", "4.0", "41":
		return ErrVersion4
	default:
		return fmt.Errorf("%w: %q", ErrUnsupportedVersion, version)
	}
}

func parseRecord(kind Kind, line int, raw string, fields []string) Record {
	b := newBase(kind, line, raw, fields)
	f := func(position int) string {
		if position < 1 || position > len(fields) {
			return ""
		}
		return strings.TrimSpace(fields[position-1])
	}
	rawField := func(position int) string {
		if position < 1 || position > len(fields) {
			return ""
		}
		return strings.TrimRight(fields[position-1], " \t")
	}

	switch kind {
	case KindHeader:
		header := &HeaderRecord{
			base:        b,
			Version:     f(2),
			DataMarker:  f(3),
			DateRaw:     f(4),
			Currency:    f(5),
			Description: f(6),
			DataContent: f(9),
			Copyright:   f(10),
			Creator:     f(11),
		}
		header.Info = []string{f(6), f(7), f(8)}
		for position := 12; position <= 18; position++ {
			if value := f(position); value != "" {
				header.AddressLines = append(header.AddressLines, value)
			}
		}
		if date, err := parseDatanormDate(header.DateRaw); err == nil {
			header.Date = date
			header.DateValid = true
		}
		return header

	case KindArticle:
		return &ArticleRecord{
			base:                            b,
			ProcessingFlag:                  ParseProcessingFlag(f(2)),
			Number:                          f(3),
			ShortText1:                      f(4),
			ShortText2:                      f(5),
			Unit:                            f(6),
			PriceTypeRaw:                    f(7),
			PriceUnitRaw:                    f(8),
			PriceRaw:                        f(9),
			DiscountGroup:                   f(10),
			MainGroup:                       f(11),
			ProductGroup:                    f(12),
			Matchcode:                       f(13),
			AlternativeArticleNumberCreator: f(14),
			AlternativeArticleNumber:        f(15),
			ManufacturerNumberCreator:       f(16),
			ManufacturerNumber:              f(17),
			ManufacturerType:                f(18),
			GTIN:                            f(19),
			MediaReference:                  f(20),
			MinOrderQuantityRaw:             f(21),
			CatalogPage:                     f(22),
			TextMarker:                      TextMarker(f(23)),
			LongTextKey:                     f(24),
			StorageFlag:                     f(25),
			ReferenceNumberCreator:          f(26),
			ReferenceNumber:                 f(27),
			VATMarker:                       f(28),
			CostType:                        f(29),
			PackagingQuantity:               f(30),
		}

	case KindChange:
		flag := ParseProcessingFlag(f(2))
		change := &ChangeRecord{
			base:           b,
			ProcessingFlag: flag,
			ArticleNumber:  f(3),
			DateRaw:        f(6),
		}
		if flag == FlagRenumber {
			change.NewArticleNumber = f(4)
			change.Successor = f(5)
		} else {
			change.Successor = f(4)
			change.DateRaw = firstNonEmpty(f(5), f(6))
		}
		if len(fields) > 6 {
			change.Extra = fields[6:]
		}
		return change

	case KindText:
		lineNumber, _ := strconv.Atoi(f(5))
		return &TextRecord{
			base:           b,
			ProcessingFlag: ParseProcessingFlag(f(2)),
			Key:            f(3),
			TextType:       f(4),
			LineNumberRaw:  f(5),
			LineNumber:     lineNumber,
			Text:           rawField(6),
		}

	case KindDimension:
		lineNumber, _ := strconv.Atoi(f(4))
		dimension := &DimensionRecord{
			base:           b,
			ProcessingFlag: ParseProcessingFlag(f(2)),
			ArticleNumber:  f(3),
			LineNumberRaw:  f(4),
			LineNumber:     lineNumber,
			Type:           strings.ToUpper(f(5)),
		}
		if dimension.IsInsertion() {
			dimension.Key = firstNonEmpty(f(7), f(6))
		} else {
			dimension.Text = firstNonEmpty(rawField(7), rawField(6))
		}
		return dimension

	case KindEnd:
		count, err := strconv.Atoi(f(2))
		return &EndRecord{
			base:       b,
			CountRaw:   f(2),
			Count:      count,
			CountValid: err == nil,
			Creator:    f(3),
		}

	case KindPrice:
		price := &PriceRecord{base: b}
		flag := ParseProcessingFlag(f(2))
		useFlag := false
		if flag.Valid() {
			noFlagValid := looksLikePriceType(f(3)) && looksLikePriceUnit(f(4)) && f(5) != ""
			flagValid := looksLikePriceType(f(4)) && looksLikePriceUnit(f(5)) && f(6) != ""
			useFlag = flagValid && !noFlagValid
		}
		offset := 0
		if useFlag {
			price.ProcessingFlag = flag
			offset = 1
		}
		price.ArticleNumber = f(2 + offset)
		price.PriceTypeRaw = f(3 + offset)
		price.PriceUnitRaw = f(4 + offset)
		price.PriceRaw = f(5 + offset)
		price.DiscountGroup = f(6 + offset)
		if len(fields) > 6+offset {
			price.Extra = fields[6+offset:]
		}
		return price

	case KindGroup:
		return &GroupRecord{
			base:                 b,
			MainGroup:            f(2),
			ProductGroup:         f(3),
			Description:          f(4),
			MainGroupDescription: f(5),
		}

	case KindDiscount:
		return &DiscountRecord{
			base:        b,
			Group:       f(2),
			KindRaw:     f(3),
			ValueRaw:    f(4),
			Description: f(5),
		}

	case KindMedia:
		offset := 0
		if ParseProcessingFlag(f(2)).Valid() {
			offset = 1
		}
		return &MediaRecord{
			base:            b,
			Heuristic:       true,
			ArticleNumber:   f(2 + offset),
			ReferenceNumber: f(3 + offset),
			MediaTypeRaw:    f(4 + offset),
			Description:     f(5 + offset),
			Filename:        f(6 + offset),
		}

	case KindSurcharge:
		surcharge := &SurchargeRecord{
			base:             b,
			ProcessingFlag:   ParseProcessingFlag(f(2)),
			ArticleNumber:    f(3),
			SetNumberRaw:     f(4),
			ProcessingMarker: f(5),
			BasisMarker:      f(6),
			BasisDescription: f(7),
			PriceTypeRaw:     f(8),
			PriceRaw:         f(9),
			FromRaw:          f(10),
			ToRaw:            f(11),
		}
		if len(fields) > 11 {
			surcharge.Extra = fields[11:]
		}
		return surcharge

	case KindSet:
		return &SetRecord{
			base:                   b,
			SetArticleNumber:       f(3),
			ComponentArticleNumber: f(4),
			SubTypeRaw:             f(2),
		}

	case KindService:
		return &ServiceRecord{
			base:          b,
			ArticleNumber: f(3),
			SubTypeRaw:    f(2),
		}

	case KindControl:
		return &ControlRecord{
			base:           b,
			CustomerNumber: firstNonEmpty(f(3), f(2)),
		}

	default:
		rawKind := byte(0)
		if raw != "" {
			rawKind = raw[0]
		}
		return &UnknownRecord{base: b, RawKind: rawKind}
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func looksLikePriceType(value string) bool {
	switch strings.TrimSpace(value) {
	case "1", "2", "3", "4", "5", "9":
		return true
	default:
		return false
	}
}

func looksLikePriceUnit(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	n, err := strconv.ParseInt(value, 10, 64)
	return err == nil && n > 0
}

func parseDatanormDate(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if len(value) != 8 || !allDigits(value) {
		return time.Time{}, fmt.Errorf("invalid date %q", raw)
	}
	if date, err := time.Parse("20060102", value); err == nil && date.Year() >= 1900 {
		return date, nil
	}
	if date, err := time.Parse("02012006", value); err == nil && date.Year() >= 1900 {
		return date, nil
	}
	return time.Time{}, fmt.Errorf("invalid date %q", raw)
}

func allDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return value != ""
}

// utf8RuneCount is a small helper for field-length validation.
func utf8RuneCount(value string) int {
	return utf8.RuneCountInString(value)
}

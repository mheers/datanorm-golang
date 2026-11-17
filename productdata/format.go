package productdata

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"sync"
)

// Errors shared by format implementations and the registry.
var (
	ErrEncodingUnsupported = errors.New("productdata: format does not support encoding")
	ErrDuplicateFormat     = errors.New("productdata: duplicate format registration")
	ErrInvalidFormat       = errors.New("productdata: invalid format")
	ErrNotDetected         = errors.New("productdata: no format detected")
)

// Capabilities describes what a format can represent and do. It lets a caller
// choose a format before attempting an operation.
type Capabilities struct {
	Read          bool `json:"read"`
	Write         bool `json:"write"`
	Streaming     bool `json:"streaming"`
	Batch         bool `json:"batch"`
	SingleProduct bool `json:"singleProduct"`
	Prices        bool `json:"prices"`
	PriceTiers    bool `json:"priceTiers"`
	Texts         bool `json:"texts"`
	Media         bool `json:"media"`
	Features      bool `json:"features"`
	Availability  bool `json:"availability"`
	Basket        bool `json:"basket"`
	Associations  bool `json:"associations"`
	CustomsTariff bool `json:"customsTariff"`
}

// DecodeOptions are common options for decoding. Format-specific decoders may
// define additional options of their own.
type DecodeOptions struct {
	// Strict turns selected warnings into errors.
	Strict bool
	// Charset overrides character-set detection, for example "cp850".
	Charset string
	// CollectRaw keeps format-specific raw data on articles and records.
	CollectRaw bool
}

// EncodeOptions are common options for encoding.
type EncodeOptions struct {
	// Charset selects the output character set, for example "cp850".
	Charset string
	// LineEnding is "\r\n" or "\n".
	LineEnding string
	// Version selects a format version.
	Version string
}

// Decoder reads a format and returns the canonical model.
type Decoder interface {
	Decode(ctx context.Context, r io.Reader, opts DecodeOptions) (*Catalog, error)
}

// Encoder writes the canonical model in a format.
type Encoder interface {
	Encode(ctx context.Context, w io.Writer, catalog *Catalog, opts EncodeOptions) error
}

// Format describes a product-data format and creates decoders and encoders.
type Format interface {
	// Name is the stable registry key, for example "datanorm".
	Name() string
	// Description is a short human-readable description.
	Description() string
	// Extensions are lower-case file extensions, including the dot.
	Extensions() []string
	// Detect reports whether filename and/or the first bytes match the format.
	Detect(filename string, head []byte) bool
	// Capabilities describes what the format can represent.
	Capabilities() Capabilities
	// NewDecoder creates a decoder.
	NewDecoder() (Decoder, error)
	// NewEncoder creates an encoder. Formats that cannot write return
	// ErrEncodingUnsupported.
	NewEncoder() (Encoder, error)
}

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Format)
	ordered    []Format
)

// Register adds a format to the process-wide registry. Registering the same
// name twice returns ErrDuplicateFormat.
func Register(format Format) error {
	if format == nil || format.Name() == "" {
		return ErrInvalidFormat
	}
	name := strings.ToLower(format.Name())

	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[name]; exists {
		return ErrDuplicateFormat
	}
	registry[name] = format
	ordered = append(ordered, format)
	return nil
}

// Get returns a registered format by name.
func Get(name string) (Format, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	format, ok := registry[strings.ToLower(name)]
	return format, ok
}

// Formats returns the registered formats in registration order.
func Formats() []Format {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]Format, len(ordered))
	copy(out, ordered)
	return out
}

// Detect returns the first registered format that matches the filename or the
// first bytes.
func Detect(filename string, head []byte) (Format, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	for _, format := range ordered {
		if format.Detect(filename, head) {
			return format, true
		}
	}
	return nil, false
}

// DetectExtension is a helper for Format implementations. It reports whether
// the lower-case filename has one of the given extensions.
func DetectExtension(filename string, extensions ...string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, candidate := range extensions {
		if ext == strings.ToLower(candidate) {
			return true
		}
	}
	return false
}

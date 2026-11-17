package datanorm

import (
	"context"
	"io"
	"path/filepath"
	"strings"

	"github.com/mheers/datanorm-golang/productdata"
)

func init() {
	_ = productdata.Register(datanormFormat{})
}

type datanormFormat struct{}

func (datanormFormat) Name() string        { return "datanorm" }
func (datanormFormat) Description() string { return "DATANORM 5.0 product and price data" }

func (datanormFormat) Extensions() []string {
	return []string{".001", ".002", ".003", ".004", ".005", ".006", ".007", ".008", ".009", ".rab", ".wrg"}
}

func (datanormFormat) Detect(filename string, head []byte) bool {
	text := strings.TrimPrefix(string(head), "\uFEFF")
	if strings.HasPrefix(text, "V;050;") || strings.HasPrefix(text, "V;0") {
		return true
	}
	if strings.HasPrefix(text, "V ") {
		// DATANORM 4 fixed-width header. Detect it so the decoder can return a
		// clear ErrVersion4 instead of a generic parse failure.
		return true
	}
	base := strings.ToLower(filepath.Base(filename))
	if strings.HasPrefix(base, "datanorm") || strings.HasPrefix(base, "datpreis") ||
		strings.HasPrefix(base, "datasets") || strings.HasPrefix(base, "datatext") {
		return true
	}
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".rab", ".wrg":
		return true
	}
	return false
}

func (datanormFormat) Capabilities() productdata.Capabilities {
	return productdata.Capabilities{
		Read:          true,
		Write:         true,
		Streaming:     true,
		Batch:         true,
		SingleProduct: false,
		Prices:        true,
		PriceTiers:    false,
		Texts:         true,
		Media:         true,
		Features:      false,
		Availability:  false,
		Basket:        false,
		Associations:  true,
		CustomsTariff: false,
	}
}

func (datanormFormat) NewDecoder() (productdata.Decoder, error) {
	return decoder{}, nil
}

func (datanormFormat) NewEncoder() (productdata.Encoder, error) {
	return NewEncoder(), nil
}

type decoder struct{}

func (decoder) Decode(ctx context.Context, r io.Reader, opts productdata.DecodeOptions) (*productdata.Catalog, error) {
	return Decode(ctx, r, opts)
}

// Decode reads a DATANORM 5 stream and returns the canonical catalog. It is the
// generic-format entry point; format-specific options are available through
// Parse and Open.
func Decode(ctx context.Context, r io.Reader, opts productdata.DecodeOptions) (*productdata.Catalog, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var parseOptions []Option
	if opts.Strict {
		parseOptions = append(parseOptions, WithStrict(true))
	}
	if opts.Charset != "" {
		parseOptions = append(parseOptions, WithCharset(opts.Charset))
	}
	file, err := Parse(r, parseOptions...)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return file.Catalog()
}

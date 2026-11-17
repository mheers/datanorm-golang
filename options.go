package datanorm

import "github.com/mheers/datanorm-golang/charset"

// options holds parser and encoder options.
type options struct {
	charsetName  string
	charset      charset.Charset
	strict       bool
	maxLineBytes int
	validate     bool
	lineEnding   string
}

// Option configures a parser or encoder.
type Option func(*options)

func defaultOptions() options {
	return options{
		charset:      charset.Auto,
		maxLineBytes: 1 << 20,
		validate:     true,
		lineEnding:   "\r\n",
	}
}

// WithCharset selects the input or output character encoding. Accepted values
// include "auto", "utf-8", "cp437", "cp850", "windows-1252", "iso-8859-15",
// and "iso-8859-1".
func WithCharset(name string) Option {
	return func(o *options) {
		o.charsetName = name
	}
}

// WithStrict turns selected warnings into errors. It is off by default because
// real supplier files are frequently imperfect.
func WithStrict(strict bool) Option {
	return func(o *options) {
		o.strict = strict
	}
}

// WithMaxLineBytes sets the maximum accepted physical line length.
func WithMaxLineBytes(n int) Option {
	return func(o *options) {
		if n > 0 {
			o.maxLineBytes = n
		}
	}
}

// WithoutValidation disables structural validation.
func WithoutValidation() Option {
	return func(o *options) {
		o.validate = false
	}
}

// WithLineEnding selects "\r\n" or "\n" for the encoder.
func WithLineEnding(lineEnding string) Option {
	return func(o *options) {
		if lineEnding == "\n" || lineEnding == "\r\n" {
			o.lineEnding = lineEnding
		}
	}
}

func (o options) resolveCharset() (charset.Charset, error) {
	if o.charsetName == "" {
		return o.charset, nil
	}
	return charset.Parse(o.charsetName)
}

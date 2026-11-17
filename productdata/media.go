package productdata

// MediaType describes the kind of a referenced file.
type MediaType string

// Media types.
const (
	MediaImage    MediaType = "image"
	MediaDocument MediaType = "document"
	MediaDrawing  MediaType = "drawing"
	MediaVideo    MediaType = "video"
	MediaAudio    MediaType = "audio"
	MediaOther    MediaType = "other"
)

// Media is a reference to an external file. The file is never embedded in the
// product data itself.
type Media struct {
	Type        MediaType `json:"type"`
	URL         string    `json:"url,omitempty"`
	Filename    string    `json:"filename,omitempty"`
	MIME        string    `json:"mime,omitempty"`
	Description string    `json:"description,omitempty"`
	Language    string    `json:"language,omitempty"`
}

// Feature is a technical attribute, for example an ETIM feature.
type Feature struct {
	Group string `json:"group,omitempty"`
	Name  string `json:"name"`
	Value string `json:"value"`
	Unit  string `json:"unit,omitempty"`
}

// Classification is a standardized classification of an article, for example
// an ETIM class or a DIN 4000 characteristic list.
type Classification struct {
	System string `json:"system"`
	Code   string `json:"code"`
	Name   string `json:"name,omitempty"`
}

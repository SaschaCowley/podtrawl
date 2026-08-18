package feed

import "encoding/xml"

type Rss struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}

// Channel may also have the following optional children,
// which are intentionally excluded as I don't see the use of them in this application:
// generator, docs, cloud, image, rating, and textInput.
type Channel struct {
	// Required elements
	Title       string `xml:"title"`
	Link        string `xml:"link"` // URL
	Description string `xml:"description"`

	// Optional elements
	Language       *string          `xml:"language"`
	Copyright      *string          `xml:"copyright"`
	ManagingEditor *string          `xml:"managingEditor"` // Email address
	WebMaster      *string          `xml:"webMaster"`      // Email address
	PubDate        *Rfc2822Datetime `xml:"pubDate"`
	LastBuildDate  *Rfc2822Datetime `xml:"lastBuildDate"`
	Categories     []Category       `xml:"category"`
	Ttl            *int             `xml:"ttl"`
	SkipHours      []int            `xml:"skipHours>hour"`
	SkipDays       []string         `xml:"skipDays>day"`
	Items          []Item           `xml:"item"`
}

// The spec says that all of these children are optional,
// but that either title or description must be provided.
// Currently the unmarshaler does not validate this.
type Item struct {
	// Optional children
	Title       *string          `xml:"title"`
	Link        *string          `xml:"link"`
	Description *string          `xml:"description"`
	Author      *string          `xml:"author"` // email address
	Categories  []Category       `xml:"category"`
	Comments    *string          `xml:"comments"` // Url
	Enclosures  []Enclosure      `xml:"enclosure"`
	Guid        *Guid            `xml:"guid"`
	PubDate     *Rfc2822Datetime `xml:"pubDate"`
	Source      *Source          `xml:"source"`
}

type Category struct {
	Domain *string `xml:"domain,attr"`
	Value  string  `xml:",chardata"`
}

type Enclosure struct {
	Url    string `xml:"url,attr"`
	Length int64  `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

type Guid struct {
	IsPermalink bool   `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

// UnmarshalXML decodes a guid, defaulting isPermaLink to true as RSS 2.0 requires.
// The type alias keeps encoding/xml from recursing back into this method.
func (g *Guid) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type guid Guid
	aux := guid{
		IsPermalink: true,
	}
	if err := d.DecodeElement(&aux, &start); err != nil {
		return err
	}
	*g = (Guid)(aux)
	return nil
}

type Source struct {
	Url   string `xml:"url,attr"`
	Title string `xml:",chardata"`
}

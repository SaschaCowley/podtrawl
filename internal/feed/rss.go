package feed

import "encoding/xml"

// nsUnnamespaced is the name space Load assigns to a feed's unadorned elements,
// so we can distinguish between <tag> and <ns:tag>.
const nsUnnamespaced = "_"

type Rss struct {
	XMLName xml.Name `xml:"_ rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"_ channel"`
}

// Channel may also have the following optional children,
// which are intentionally excluded as I don't see the use of them in this application:
// generator, docs, cloud, image, rating, and textInput.
type Channel struct {
	// Required elements
	Title       string `xml:"_ title"`
	Link        string `xml:"_ link"` // URL
	Description string `xml:"_ description"`

	// Optional elements
	Language       *string          `xml:"_ language"`
	Copyright      *string          `xml:"_ copyright"`
	ManagingEditor *string          `xml:"_ managingEditor"` // Email address
	WebMaster      *string          `xml:"_ webMaster"`      // Email address
	PubDate        *Rfc2822Datetime `xml:"_ pubDate"`
	LastBuildDate  *Rfc2822Datetime `xml:"_ lastBuildDate"`
	Categories     []Category       `xml:"_ category"`
	Ttl            *int             `xml:"_ ttl"`
	SkipHours      []int            `xml:"_ skipHours>hour"`
	SkipDays       []string         `xml:"_ skipDays>day"`
	Items          []Item           `xml:"_ item"`
}

// The spec says that all of these children are optional,
// but that either title or description must be provided.
// Currently the unmarshaler does not validate this.
type Item struct {
	// Optional children
	Title       *string          `xml:"_ title"`
	Link        *string          `xml:"_ link"`
	Description *string          `xml:"_ description"`
	Author      *string          `xml:"_ author"` // email address
	Categories  []Category       `xml:"_ category"`
	Comments    *string          `xml:"_ comments"` // Url
	Enclosures  []Enclosure      `xml:"_ enclosure"`
	Guid        *Guid            `xml:"_ guid"`
	PubDate     *Rfc2822Datetime `xml:"_ pubDate"`
	Source      *Source          `xml:"_ source"`
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

package feed

import (
	"encoding/xml"
	"io"
)

func Load(body io.Reader) (*Rss, error) {
	var feed Rss
	dec := xml.NewDecoder(body)
	dec.DefaultSpace = nsUnnamespaced
	if err := dec.Decode(&feed); err != nil {
		return nil, err
	}
	return &feed, nil
}

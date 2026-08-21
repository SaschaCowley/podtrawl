package feed

import (
	"fmt"
	"strings"
	"time"
)

type Rfc2822Datetime struct {
	time.Time
}

// Equal reports whether two datetimes denote the same instant. Parsing a named
// zone that is not the local zone fabricates a location, so comparing the
// Rfc2822Datetime structs directly would distinguish times that are identical.
func (d Rfc2822Datetime) Equal(other Rfc2822Datetime) bool {
	return d.Time.Equal(other.Time)
}

var rfc2822Formats = [...]string{
	"Mon, 2 Jan 2006 15:04:05 -0700",
	"Mon, 2 Jan 2006 15:04:05 MST",
	"2 Jan 2006 15:04:05 -0700",
	"2 Jan 2006 15:04:05 MST",
}

func (d *Rfc2822Datetime) UnmarshalText(text []byte) error {
	s := strings.TrimSpace(string(text))
	for _, format := range rfc2822Formats {
		t, err := time.Parse(format, s)
		if err == nil {
			d.Time = t
			return nil
		}
	}
	return fmt.Errorf("unable to parse RFC 2822 datetime %q", s)
}

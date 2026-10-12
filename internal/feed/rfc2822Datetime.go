package feed

import (
	"fmt"
	"strings"
	"time"
)

// Rfc2822Datetime is a datetime in the RFC 2822 format that RSS 2.0 requires.
// Feeds in the wild also use RFC 3339, so that is accepted as a fallback.
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
	// Not RFC 2822, but common enough that rejecting it would lose whole feeds.
	// Parsing accepts fractional seconds even though the layout omits them.
	time.RFC3339,
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
	return fmt.Errorf("unable to parse RFC 2822 or RFC 3339 datetime %q", s)
}

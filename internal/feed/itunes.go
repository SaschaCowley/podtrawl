package feed

import (
	"math"
	"strconv"
	"strings"
	"time"
)

// nsItunes is the name space of Apple's podcast extensions to RSS.
// Struct tags cannot refer to constants, so the tags below repeat it literally;
// TestItunesTypesAreNamespaced catches any that drift.
const nsItunes = "http://www.itunes.com/dtds/podcast-1.0.dtd"

// ItunesChannel holds the itunes: children of a channel.
// The following children are intentionally excluded, being either deprecated or of no use in this application:
// summary, subtitle, keywords, block, image and complete.
type ItunesChannel struct {
	Title      *string          `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd title"`
	Author     *string          `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd author"`
	Type       *string          `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd type"` // episodic or serial
	NewFeedUrl *string          `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd new-feed-url"`
	Explicit   ItunesExplicit   `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd explicit"`
	Categories []ItunesCategory `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd category"`
	Owner      *ItunesOwner     `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd owner"`
}

// ItunesItem holds the itunes: children of an item.
// The following children are intentionally excluded, being either deprecated or of no use in this application:
// summary, subtitle, keywords, image and block.
type ItunesItem struct {
	Title       *string        `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd title"`
	Author      *string        `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd author"`
	EpisodeType *string        `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd episodeType"` // full, trailer, or bonus
	Episode     ItunesNumber   `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd episode"`
	Season      ItunesNumber   `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd season"`
	Duration    ItunesDuration `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd duration"`
	Explicit    ItunesExplicit `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd explicit"`
}

type ItunesCategory struct {
	Text          string           `xml:"text,attr"`
	Subcategories []ItunesCategory `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd category"`
}

type ItunesOwner struct {
	Name  *string `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd name"`
	Email *string `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd email"`
}

// The types below are lenient: their UnmarshalText never fails,
// so a malformed value costs only that value rather than the whole feed.
// Their zero value means absent or unparseable, and is never a legal value in itself.
// They are values rather than pointers because encoding/xml allocates a pointer before unmarshaling into it,
// so a pointer could not be left nil for a malformed value.

// ItunesExplicit reports whether an itunes:explicit element marks content as explicit.
type ItunesExplicit int

const (
	ExplicitUnknown ItunesExplicit = iota
	ExplicitNo
	ExplicitYes
)

func (e *ItunesExplicit) UnmarshalText(text []byte) error {
	switch strings.ToLower(strings.TrimSpace(string(text))) {
	case "true", "yes", "explicit":
		*e = ExplicitYes
	case "false", "no", "clean":
		*e = ExplicitNo
	default:
		*e = ExplicitUnknown
	}
	return nil
}

// ItunesNumber is a positive integer, such as an episode or season number.
type ItunesNumber int

func (n *ItunesNumber) UnmarshalText(text []byte) error {
	*n = 0
	s := strings.TrimSpace(string(text))
	v, err := strconv.Atoi(s)
	if err != nil {
		// Some feeds write integers as "1.0".
		f, err := strconv.ParseFloat(s, 64)
		if err != nil || f != math.Trunc(f) || f > math.MaxInt32 {
			return nil
		}
		v = int(f)
	}
	if v > 0 {
		*n = ItunesNumber(v)
	}
	return nil
}

// ItunesDuration is the length of an episode,
// given in the feed as seconds, MM:SS, or HH:MM:SS, the last of which may be fractional.
type ItunesDuration struct {
	time.Duration
}

func (d *ItunesDuration) UnmarshalText(text []byte) error {
	d.Duration = 0
	parts := strings.Split(strings.TrimSpace(string(text)), ":")
	if len(parts) > 3 {
		return nil
	}
	var seconds float64
	for i, part := range parts {
		v, err := strconv.ParseFloat(part, 64)
		// Only the last part may be fractional, and none may be negative, NaN, or infinite.
		if err != nil || !(v >= 0) || math.IsInf(v, 1) || (i < len(parts)-1 && v != math.Trunc(v)) {
			return nil
		}
		seconds = seconds*60 + v
	}
	if seconds*float64(time.Second) > math.MaxInt64 {
		return nil
	}
	d.Duration = time.Duration(seconds * float64(time.Second))
	return nil
}

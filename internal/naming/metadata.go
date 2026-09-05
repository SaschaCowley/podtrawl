// Package naming derives the names under which podtrawl writes episodes.
//
// Metadata flattens channel, item and enclosure attributes into plain values.
// Most elements in RSS are optional, so feed uses  pointers  to distinguish omissions from zero values.
// They are converted to plane values so that templates can't dereference nil pointers.
package naming

import (
	"mime"
	"net/url"
	"path"
	"strings"
	"time"

	"ssch.cc/podtrawl/internal/feed"
)

// FeedMeta describes a show.
type FeedMeta struct {
	// Title, Link and Description are required by RSS.
	Title       string
	Link        string
	Description string

	// Url is the feed's url, as configured.
	// It is guaranteed to be non-zero.
	Url string
}

// EpisodeMeta describes one enclosure of one item.
//
// All members that are optional in RSS are set to their type's zero value if omitted in the feed.
type EpisodeMeta struct {
	// From the item
	Title       string
	Link        string
	Description string
	Guid        string
	PubDate     time.Time

	// From the enclosure
	Url    string
	Type   string
	Length int64 // Frequently absent or wrong

	// Derived
	// Filename is the episode's name as extracted from the enclosure's URL.
	Filename string
	// Stem is Filename less the file extension.
	Stem string
	// Ext is the file extension.
	// If Filename had an extension, it is used,
	// so Filename == Stem + Ext.
	// Otherwise, an extension is guessed from the enclosure's Type.
	// If not empty, Ext will always start with a '.'.
	Ext string
}

// Metadata is everything a name template can see.
type Metadata struct {
	Feed    FeedMeta
	Episode EpisodeMeta
}

// NewFeedMeta describes the channel of the feed fetched from url.
func NewFeedMeta(c feed.Channel, url string) FeedMeta {
	return FeedMeta{
		Title:       c.Title,
		Link:        c.Link,
		Description: c.Description,
		Url:         url,
	}
}

// NewEpisodeMeta describes one enclosure of one item.
// nil pointers become zero values .
func NewEpisodeMeta(item feed.Item, enclosure feed.Enclosure) EpisodeMeta {
	m := EpisodeMeta{
		Url:    enclosure.Url,
		Type:   enclosure.Type,
		Length: enclosure.Length,
	}
	if item.Title != nil {
		m.Title = *item.Title
	}
	if item.Link != nil {
		m.Link = *item.Link
	}
	if item.Description != nil {
		m.Description = *item.Description
	}
	if item.Guid != nil {
		m.Guid = item.Guid.Value
	}
	if item.PubDate != nil {
		m.PubDate = item.PubDate.Time
	}
	m.Filename, m.Stem, m.Ext = nameParts(enclosure.Url, enclosure.Type)
	return m
}

func nameParts(rawUrl, mediaType string) (name, stem, ext string) {
	name = baseName(rawUrl)
	if len(name) < 3 {
		stem = name
	} else {
		stem, ext = splitName(name)
	}
	if ext == "" {
		ext = extFromType(mediaType)
	}
	return
}

// baseName is the last element of a url's path,
// or "" when the url offers nothing usable.
func baseName(rawUrl string) string {
	u, err := url.Parse(rawUrl)
	if err != nil {
		return ""
	}
	// Path is already percent-decoded.
	// Base reports an empty path as "."
	// and a path of nothing but slashes as "/",
	// neither of which is a name.
	if base := path.Base(u.Path); base != "." && base != "/" {
		return base
	}
	return ""
}

// splitName splits a file name into its stem and extension.
// A dot only separates an extension when there is something on either side of it,
// so that ".hidden" and "name." are both stems without extensions.
func splitName(name string) (stem, ext string) {
	if i := strings.LastIndexByte(name, '.'); i > 0 && i < len(name)-1 {
		return name[:i], name[i:]
	}
	return name, ""
}

// extFromType maps an enclosure's media type to a file extension.
//
// The table is consulted first because mime.ExtensionsByType reports every
// registered extension in sorted order, which for audio/mpeg is not .mp3,
// and which varies with whatever the host system has registered.
// Types the table doesn't cover fall through to the system's answer,
// and a type that is unknown or unparseable yields "".
func extFromType(mediaType string) string {
	t, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		return ""
	}
	if ext, ok := mimeToExt[t]; ok {
		return ext
	}
	if exts, err := mime.ExtensionsByType(t); err == nil && len(exts) > 0 {
		return exts[0]
	}
	return ""
}

// mimeToExt gives the preferred extension for the media types podcasts ship.
// Keys must be canonical, as they are looked up after mime.ParseMediaType,
// which lower-cases the type and drops its parameters.
var mimeToExt = map[string]string{
	"audio/aac":      ".aac",
	"audio/flac":     ".flac",
	"audio/mp4":      ".m4a",
	"audio/x-m4a":    ".m4a",
	"audio/mpeg":     ".mp3",
	"audio/ogg":      ".ogg",
	"audio/opus":     ".opus",
	"audio/vnd.wave": ".wav",
	"audio/wav":      ".wav",
	"audio/x-wav":    ".wav",
	"audio/webm":     ".weba",
}

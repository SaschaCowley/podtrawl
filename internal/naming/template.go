package naming

import (
	"fmt"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"ssch.cc/podtrawl/internal/fsname"
)

// Template renders one kind of name.
// T is the metadata that kind of name can see:
// FeedOnly for a show directory;
// or Metadata for an episode of a show.
type Template[T any] struct {
	template *template.Template
	sanitize bool
}

// ParseFeed compiles a template for an entire feed.
// It sees .Feed, but no episode.
func ParseFeed(name, text string, sanitize bool) (*Template[FeedOnly], error) {
	return parse(name, text, sanitize, FeedOnly{Feed: probe.Feed})
}

// ParseEpisode compiles a template for an episode of a feed.
// It sees .Feed and .Episode.
func ParseEpisode(name, text string, sanitize bool) (*Template[Metadata], error) {
	return parse(name, text, sanitize, probe)
}

// parse compiles a name template, validating it against probe.
// Templates that name something that T does not carry,
// or that cannot produce a usable relative path, are rejected.
//
// When sanitize is true, each segment of the rendered path is made safe for the file system.
// To sanitize individual values, they should be piped to sanitize in the template.
func parse[T any](name, text string, sanitize bool, probe T) (*Template[T], error) {
	tmpl, err := template.New(name).Funcs(funcs).Parse(text)
	if err != nil {
		return nil, err
	}
	t := &Template[T]{tmpl, sanitize}
	if _, err := t.Render(probe); err != nil {
		return nil, err
	}
	return t, nil
}

// probe stands in for a feed while a template is checked.
var probe = Metadata{
	Feed: FeedMeta{
		Title:       "feed title",
		Link:        "https://example.com",
		Description: "feed description",
		Url:         "https://example.com/feed.xml",
	},
	Episode: EpisodeMeta{
		Title:       "episode title",
		Link:        "https://example.com/1",
		Description: "episode description",
		Guid:        "episode guid",
		PubDate:     time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC),
		Url:         "https://example.com/episode.mp3",
		Type:        "audio/mpeg",
		Length:      1,
		Filename:    "episode.mp3",
		Stem:        "episode",
		Ext:         ".mp3",
	},
}

// Render produces a relative path from the metadata of one show or episode.
func (t *Template[T]) Render(v T) (string, error) {
	var wr strings.Builder
	if err := t.template.Execute(&wr, v); err != nil {
		return "", err
	}
	return clean(wr.String(), t.sanitize)
}

// toSlash converts path separators to slashes.
//
// filepath.ToSlash only converts backslashes to slashes if backslash is the system path separator.
// This does so regardless.
func toSlash(path string) string {
	n := []byte(path)
	for i := range n {
		if n[i] == '\\' {
			n[i] = '/'
		}
	}
	return string(n)
}

// isAbs determines whether the given path is absolute.
//
// The given path should only use slash as the path separator.
// This does recognise Windows-style paths.
func isAbs(path string) bool {
	switch {
	case len(path) == 0:
		return false
	case path[0] == '/':
		return true
	case len(path) >= 3 && path[1] == ':' && path[2] == '/' && ((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z')):
		return true
	default:
		return false
	}
}

// clean turns a rendered template into a relative path, one segment per slash.
// Empty and "." segments are dropped;
// paths that are empty, absolute, or traverse upward are rejected.
// When sanitize is set, every segment is made safe for the file system,
// and a segment that sanitizes away to nothing is an error.
func clean(path string, sanitize bool) (string, error) {
	path = toSlash(path)
	if isAbs(path) {
		return "", fmt.Errorf("path must not be absolute: %q", path)
	}
	segments := make([]string, 0, strings.Count(path, "/")+1)
	for segment := range strings.SplitSeq(path, "/") {
		if segment == "" || segment == "." {
			continue
		}
		if segment == ".." {
			return "", fmt.Errorf("path must not contain traversal: %q", path)
		}
		if sanitize {
			if sanitized, ok := fsname.Component(segment); ok {
				segment = sanitized
			} else {
				return "", fmt.Errorf("path contains unusable segment: %q", segment)
			}
		}
		segments = append(segments, segment)
	}
	if len(segments) == 0 {
		return "", fmt.Errorf("path must not be empty: %q", path)
	}
	return filepath.Join(segments...), nil
}

var funcs template.FuncMap = template.FuncMap{
	"shorthash":  ShortHash,
	"sanitize":   func(s string) string { return fsname.ComponentOr(s, "_") },
	"sanitizeOr": fsname.ComponentOr,
	"date":       func(layout string, t time.Time) string { return t.Format(layout) },
	"lower":      strings.ToLower,
	"upper":      strings.ToUpper,
	"trim":       strings.TrimSpace,
	"replace":    func(old, new, s string) string { return strings.ReplaceAll(s, old, new) },
}

// Default name templates.
const (
	DefaultDirname  = `{{ sanitizeOr .Feed.Title (printf "show-%s" (shorthash .Feed.Url)) }}`
	DefaultFilename = `{{ sanitizeOr .Episode.Filename (printf "episode-%s" (shorthash .Episode.Url)) }}`
)

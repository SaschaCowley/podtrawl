package naming

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"ssch.cc/podtrawl/internal/fsname"
)

type Template struct {
	template *template.Template
	sanitize bool
}

func Parse(name, text string, sanitize bool) (*Template, error) {
	tmpl, err := template.New(name).Funcs(funcs).Parse(text)
	if err != nil {
		return nil, err
	}
	if err := tmpl.Execute(io.Discard, Metadata{FeedMeta{}, EpisodeMeta{}}); err != nil {
		return nil, err
	}
	return &Template{tmpl, sanitize}, nil
}

func (t *Template) Render(m Metadata) (string, error) {
	var wr strings.Builder
	if err := t.template.Execute(&wr, m); err != nil {
		return "", err
	}
	return clean(wr.String(), t.sanitize)
}

// toSlash converts path separators to slashes.
//
// filepath.toSlash only converts backslashes to slashes if backslash is the system path separator.
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
// This does check Windows-style paths.
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

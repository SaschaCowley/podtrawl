package naming

import (
	"io"
	"strings"
	"text/template"
)

type Template struct {
	template *template.Template
	sanitize bool
}

func Parse(name, text string, sanitize bool) (*Template, error) {
	tmpl, err := template.New(name).Parse(text)
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
	return wr.String(), nil
}

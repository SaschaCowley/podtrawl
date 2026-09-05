package naming

import "testing"

func TestParseRejectsUnknownFields(t *testing.T) {
	tmpl, err := Parse("name", "{{.Foobar}}", false)
	if err == nil {
		t.Fatal("Parse() on template text with unknown fields returned _, nil, want nil, error")
	}
	if tmpl != nil {
		t.Errorf("Parse() on template text with unknown fields gave %v, %v, want nil, error", tmpl, err)
	}
}

func TestBasicRender(t *testing.T) {
	tmpl, err := Parse("test", "{{.Feed.Title}} - {{.Episode.Title}}", false)
	if err != nil {
		t.Fatalf("Parse(): %v", err)
	}
	got, err := tmpl.Render(Metadata{FeedMeta{Title: "Show"}, EpisodeMeta{Title: "Episode"}})
	if err != nil {
		t.Fatalf("tmpl.Render(): %e", err)
	}
	want := "Show - Episode"
	if got != want {
		t.Errorf("tmpl.Render() = %q, want %q", got, want)
	}
}

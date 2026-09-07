package naming

import (
	"path/filepath"
	"testing"
	"time"
)

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

func TestToSlash(t *testing.T) {
	tests := []struct{ name, path, want string }{
		{"unix-like", "/home/user/Downloads/file", "/home/user/Downloads/file"},
		{"windows-like", `C:\Users\user\Downloads\file`, "C:/Users/user/Downloads/file"},
		{"no separators", "foobar", "foobar"},
		{"Mixed separators", `/\`, "//"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toSlash(tt.path); got != tt.want {
				t.Errorf("toSlash(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsAbs(t *testing.T) {
	tests := []struct {
		name, path string
		want       bool
	}{
		{"absolute unix-style", "/home/user/Downloads/file", true},
		{"absolute windows-style", "C:/Users/user/Downloads/file", true},
		{"absolute UNC", "//server/share/dir/file", true},
		{"raw namespace", "//?/test", true},
		{"device namespace", "//./COM56", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAbs(tt.path); got != tt.want {
				t.Errorf("isAbs(%q) = %t, want %t", tt.path, got, tt.want)
			}
		})
	}
}

func TestFuncs(t *testing.T) {
	metadata := Metadata{
		FeedMeta{
			Title:       "Show",
			Description: "	Show description ",
			Url:         "https://example.com/show/feed.xml",
		},
		EpisodeMeta{
			Title:       "Episode: Subtitle",
			Description: "",
			PubDate:     time.Date(2026, time.August, 16, 0, 0, 0, 0, time.UTC),
		},
	}
	tests := []struct {
		name, format, want string
	}{
		{"shorthash", "{{.Feed.Url | shorthash}}", "a87c65f2cf3aa679"},
		{"sanitize clean", "{{.Feed.Title | sanitize}}", "Show"},
		{"sanitizeOr clean", `{{sanitizeOr .Feed.Title "Alternative"}}`, "Show"},
		{"sanitize illegal characters", "{{.Episode.Title | sanitize}}", "Episode- Subtitle"},
		{"sanitizeOr illegal characters", `{{sanitizeOr .Episode.Title "Alternative"}}`, "Episode- Subtitle"},
		{"sanitize empty", "{{.Episode.Description | sanitize}}", "_"},
		{"sanitizeOr empty", `{{sanitizeOr .Episode.Description "Alternative"}}`, "Alternative"},
		{"date", `{{.Episode.PubDate | date "2006-01-02"}}`, "2026-08-16"},
		{"lower", "{{.Feed.Title | lower}}", "show"},
		{"upper", "{{.Feed.Title | upper}}", "SHOW"},
		{"trim", "{{.Feed.Description | trim}}", "Show description"},
		{"replace", `{{.Feed.Url | replace "/" "-"}}`, "https:--example.com-show-feed.xml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := Parse("test", tt.format, false)
			if err != nil {
				t.Fatalf("Parse(\"test\", %q, false) = _, %v, want string, nil", tt.format, err)
			}
			got, err := tmpl.Render(metadata)
			if err != nil {
				t.Fatalf("tmpl.Render() = _, %v, want string, nil", err)
			}
			if got != tt.want {
				t.Errorf("tmpl.Render() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClean(t *testing.T) {
	tests := []struct {
		name, path string
		sanitize   bool
		wantPath   string
		wantErr    bool
	}{
		{"clean single segment sanitized", "test", true, "test", false},
		{"clean single segment unsanitized", "test", false, "test", false},
		{"clean multiple unix-like segments sanitized", "test/path", true, filepath.Join("test", "path"), false},
		{"clean multiple unix-like segments unsanitized", "test/path", false, filepath.Join("test", "path"), false},
		{"clean multiple windows-like segments sanitized", "test\\path", true, filepath.Join("test", "path"), false},
		{"clean multiple windows-like segments unsanitized", "test\\path", false, filepath.Join("test", "path"), false},
		{"absolute unix-like sanitized", "/home/user/Downloads/file", true, "", true},
		{"absolute unix-like unsanitized", "/home/user/Downloads/file", false, "", true},
		{"absolute windows-like sanitized", "C:\\Users\\user\\Downloads\\file", true, "", true},
		{"absolute windows-like unsanitized", "C:\\Users\\user\\Downloads\\file", false, "", true},
		{"absolute unc sanitized", "\\\\server\\share\\dir\\file", true, "", true},
		{"absolute unc unsanitized", "\\\\server\\share\\dir\\file", false, "", true},
		{"drop empty sanitized", "test//path", true, filepath.Join("test", "path"), false},
		{"drop empty unsanitized", "test//path", false, filepath.Join("test", "path"), false},
		{"drop dot sanitized", "test/./path", true, filepath.Join("test", "path"), false},
		{"drop dot unsanitized", "test/./path", false, filepath.Join("test", "path"), false},
		{"traversal fails sanitized", "test/../path", true, "", true},
		{"traversal fails unsanitized", "test/../path", false, "", true},
		{"leading dotdot sanitized", "test/..path", true, filepath.Join("test", "..path"), false},
		{"leading dotdot unsanitized", "test/..path", false, filepath.Join("test", "..path"), false},
		{"unusable chars sanitized", "*:?", true, "---", false},
		{"unusable chars unsanitized", "*:?", false, "*:?", false},
		{"space segment sanitized", "test/ /path", true, "", true},
		{"space segment unsanitized", "test/ /path", false, filepath.Join("test", " ", "path"), false},
		{"reserved segment sanitized", "test/nul", true, filepath.Join("test", "nul_"), false},
		{"reserved segment unsanitized", "test/nul", false, filepath.Join("test", "nul"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := clean(tt.path, tt.sanitize)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("clean(%q, %t) = _, nil, want _, error", tt.path, tt.sanitize)
				}
				if got != "" {
					t.Errorf("clean(%q, %t) = %q, error, want \"\", error", tt.path, tt.sanitize, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("clean(%q, %t) = _, %v, want _, nill", tt.path, tt.sanitize, err)
			}
			if got != tt.wantPath {
				t.Errorf("clean(%q, %t) = %q, nil, want %q, nil", tt.path, tt.sanitize, got, tt.wantPath)
			}
		})
	}
}

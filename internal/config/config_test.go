package config

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
)

const fperm = 0644

// discard keeps Get's debug logging out of the test output.
func discard() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// writeConfig creates a config file in dir and returns its path.
func writeConfig(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, configFileName)
	if err := os.WriteFile(path, []byte(content), fperm); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
	return path
}

// userConfigDir points os.UserConfigDir at a temporary directory
// and returns the directory it will now report,
// so that Get's search path can be exercised
// without reading or writing the real one.
func userConfigDir(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	want := base
	switch runtime.GOOS {
	case "windows":
		t.Setenv("AppData", base)
	case "darwin":
		t.Setenv("HOME", base)
		want = filepath.Join(base, "Library", "Application Support")
	default:
		t.Setenv("XDG_CONFIG_HOME", base)
	}
	got, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("os.UserConfigDir() = _, %v, want nil", err)
	}
	// Without this the test would silently read the real user config directory,
	// and pass or fail for reasons that have nothing to do with the code.
	if got != want {
		t.Fatalf("os.UserConfigDir() = %q, want %q: the environment override did not take effect", got, want)
	}
	return got
}

func TestLoadRoot(t *testing.T) {
	// Root is a pointer so that a root left out can be told from one set to "".
	// Only the first falls back to the platform download directory;
	// the second is rejected as not absolute.
	tests := []struct {
		name    string
		content string
		want    *string
	}{
		{"unset", "", nil},
		{"set", `root = "D:/Podcasts"`, new("D:/Podcasts")},
		{"empty", `root = ""`, new("")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := load(writeConfig(t, t.TempDir(), tt.content))
			if err != nil {
				t.Fatalf("load(%q) = _, %v, want nil", tt.content, err)
			}
			switch {
			case c.Root == nil && tt.want != nil:
				t.Errorf("load(%q) gave Root = nil, want %q", tt.content, *tt.want)
			case c.Root != nil && tt.want == nil:
				t.Errorf("load(%q) gave Root = %q, want nil", tt.content, *c.Root)
			case c.Root != nil && tt.want != nil && *c.Root != *tt.want:
				t.Errorf("load(%q) gave Root = %q, want %q", tt.content, *c.Root, *tt.want)
			}
		})
	}
}

func TestLoadFeeds(t *testing.T) {
	const content = `
root = "D:/Podcasts"

[[feed]]
url = "https://example.com/podcast.rss"

[[feed]]
url = "https://example.org/another/feed.xml"
`
	c, err := load(writeConfig(t, t.TempDir(), content))
	if err != nil {
		t.Fatalf("load() = _, %v, want nil", err)
	}
	want := []string{
		"https://example.com/podcast.rss",
		"https://example.org/another/feed.xml",
	}
	if len(c.Feeds) != len(want) {
		t.Fatalf("load() gave %d feeds, want %d", len(c.Feeds), len(want))
	}
	for i, url := range want {
		if c.Feeds[i].Url != url {
			t.Errorf("Feeds[%d].Url = %q, want %q", i, c.Feeds[i].Url, url)
		}
	}
}

func TestLoadRejects(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"malformed toml", "root ="},
		{"root of the wrong type", "root = 42"},
		{"feed of the wrong shape", `feed = "https://example.com/podcast.rss"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if c, err := load(writeConfig(t, t.TempDir(), tt.content)); err == nil {
				t.Errorf("load(%q) = %+v, nil, want an error", tt.content, c)
			}
		})
	}
}

func TestLoadMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), configFileName)
	if c, err := load(path); err == nil {
		t.Errorf("load(%q) = %+v, nil, want an error", path, c)
	}
}

func TestGetExplicitPath(t *testing.T) {
	path := writeConfig(t, t.TempDir(), `root = "D:/Podcasts"`)
	c, err := Get(&path, discard())
	if err != nil {
		t.Fatalf("Get(%q) = _, %v, want nil", path, err)
	}
	if c.Root == nil {
		t.Fatalf("Get(%q) gave Root = nil, want a root", path)
	}
	if *c.Root != "D:/Podcasts" {
		t.Errorf("Get(%q) gave Root = %q, want %q", path, *c.Root, "D:/Podcasts")
	}
}

func TestGetExplicitPathIsNotSearched(t *testing.T) {
	// A config named on the command line is the only one considered,
	// so a usable config in the search path must not rescue an unusable -c.
	dir := filepath.Join(userConfigDir(t), appDirName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("creating %s: %v", dir, err)
	}
	writeConfig(t, dir, `root = "D:/FromTheSearchPath"`)

	missing := filepath.Join(t.TempDir(), "absent.toml")
	c, err := Get(&missing, discard())
	if err == nil {
		t.Fatalf("Get(%q) = %+v, nil, want an error", missing, c)
	}
	if c != nil {
		t.Errorf("Get(%q) = %+v alongside an error, want nil", missing, c)
	}
	if !errors.Is(err, ErrLoad) {
		t.Errorf("Get(%q) = _, %v, want an error wrapping ErrLoad", missing, err)
	}
}

func TestGetSearchesUserConfigDir(t *testing.T) {
	dir := filepath.Join(userConfigDir(t), appDirName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("creating %s: %v", dir, err)
	}
	writeConfig(t, dir, `root = "D:/Podcasts"`)

	c, err := Get(nil, discard())
	if err != nil {
		t.Fatalf("Get(nil) = _, %v, want nil", err)
	}
	if c.Root == nil {
		t.Fatalf("Get(nil) gave Root = nil, want a root")
	}
	if *c.Root != "D:/Podcasts" {
		t.Errorf("Get(nil) gave Root = %q, want %q", *c.Root, "D:/Podcasts")
	}
}

func TestGetReportsEveryFailure(t *testing.T) {
	// Nothing to find in the user config directory,
	// and nothing next to the test binary either.
	userConfigDir(t)
	c, err := Get(nil, discard())
	if err == nil {
		t.Fatalf("Get(nil) with no config anywhere = %+v, nil, want an error", c)
	}
	if !errors.Is(err, ErrLoad) {
		t.Errorf("Get(nil) = _, %v, want an error wrapping ErrLoad", err)
	}
}

// str returns a string representation of a string pointer.
func str(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return strconv.Quote(*s)
}

func TestLoadEmbeddedFeedSettings(t *testing.T) {
	const content = `
[[feed]]
url = "https://example.com/podcast.rss"
dirname = "{{.Feed.Title}}"
filename = "{{.Episode.Filename}}"
sanitize = false
`
	c, err := load(writeConfig(t, t.TempDir(), content))
	if err != nil {
		t.Fatalf("load() = _, %v, want nil", err)
	}
	if len(c.Feeds) != 1 {
		t.Fatalf("load() gave %d feeds, want 1", len(c.Feeds))
	}
	f := c.Feeds[0]
	if f.Dirname == nil || *f.Dirname != "{{.Feed.Title}}" {
		t.Errorf("Feeds[0].Dirname = %s, want %q", str(f.Dirname), "{{.Feed.Title}}")
	}
	if f.Filename == nil || *f.Filename != "{{.Episode.Filename}}" {
		t.Errorf("Feeds[0].Filename = %s, want %q", str(f.Filename), "{{.Episode.Filename}}")
	}
	if f.Sanitize == nil || *f.Sanitize {
		t.Errorf("Feeds[0].Sanitize = %v, want false", f.Sanitize)
	}
}

func TestLoadDefaults(t *testing.T) {
	const content = `
[defaults]
dirname = "{{.Feed.Title}}"
sanitize = true

[[feed]]
url = "https://example.com/podcast.rss"
`
	c, err := load(writeConfig(t, t.TempDir(), content))
	if err != nil {
		t.Fatalf("load() = _, %v, want nil", err)
	}
	if c.Defaults.Dirname == nil || *c.Defaults.Dirname != "{{.Feed.Title}}" {
		t.Errorf("Defaults.Dirname = %s, want %q", str(c.Defaults.Dirname), "{{.Feed.Title}}")
	}
	if c.Defaults.Sanitize == nil || !*c.Defaults.Sanitize {
		t.Errorf("Defaults.Sanitize = %v, want true", c.Defaults.Sanitize)
	}
	if c.Defaults.Filename != nil {
		t.Errorf("Defaults.Filename = %s, want nil", str(c.Defaults.Filename))
	}
	if f := c.Feeds[0]; f.Dirname != nil || f.Filename != nil || f.Sanitize != nil {
		t.Errorf("a feed with no overrides gave dirname %s, filename %s, sanitize %v, want all nil",
			str(f.Dirname), str(f.Filename), f.Sanitize)
	}
}

func TestLoadEmptyTemplateIsNotUnset(t *testing.T) {
	const content = `
[defaults]
dirname = ""

[[feed]]
url = "https://example.com/podcast.rss"
filename = ""
`
	c, err := load(writeConfig(t, t.TempDir(), content))
	if err != nil {
		t.Fatalf("load() = _, %v, want nil", err)
	}
	if c.Defaults.Dirname == nil || *c.Defaults.Dirname != "" {
		t.Errorf("Defaults.Dirname = %s, want the empty string", str(c.Defaults.Dirname))
	}
	if f := c.Feeds[0]; f.Filename == nil || *f.Filename != "" {
		t.Errorf("Feeds[0].Filename = %s, want the empty string", str(f.Filename))
	}
}

func TestLoadRejectsDuplicateFeeds(t *testing.T) {
	const content = `
[[feed]]
url = "https://example.com/podcast.rss"

[[feed]]
url = "https://example.com/podcast.rss"
dirname = "{{.Feed.Title}}"
`
	if c, err := load(writeConfig(t, t.TempDir(), content)); err == nil {
		t.Errorf("load() with a duplicate feed URL = %+v, nil, want an error", c)
	}
}

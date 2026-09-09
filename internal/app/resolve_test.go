package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ssch.cc/podtrawl/internal/config"
	"ssch.cc/podtrawl/internal/feed"
	"ssch.cc/podtrawl/internal/fsname"
	"ssch.cc/podtrawl/internal/naming"
)

const feedUrl = "https://example.com/podcast.rss"

// meta is what a run would hand a template for one episode of one show.
func meta(title, episodeUrl string) naming.Metadata {
	return naming.Metadata{
		Feed:    naming.NewFeedMeta(feed.Channel{Title: title}, feedUrl),
		Episode: naming.NewEpisodeMeta(feed.Item{}, feed.Enclosure{Url: episodeUrl}),
	}
}

// show is what a run would hand a template naming a directory for a whole feed.
func show(title string) naming.FeedOnly {
	return naming.FeedOnly{Feed: naming.NewFeedMeta(feed.Channel{Title: title}, feedUrl)}
}

// resolveOne resolves a config of exactly one feed.
func resolveOne(t *testing.T, c *config.Config) resolvedFeed {
	t.Helper()
	feeds, err := resolve(c)
	if err != nil {
		t.Fatalf("resolve() = _, %v, want 1 feed, nil", err)
	}
	if len(feeds) != 1 {
		t.Fatalf("resolve() gave %d feeds, want 1", len(feeds))
	}
	return feeds[0]
}

func TestResolvePrecedence(t *testing.T) {
	tests := []struct {
		name                      string
		feed, defaults            *string
		wantDirname, wantFilename string
	}{
		{"built-in", nil, nil, "Show", "ep12.mp3"},
		{"defaults", nil, new("defaults"), "defaults", "defaults"},
		{"feed", new("feed"), nil, "feed", "feed"},
		{"feed over defaults", new("feed"), new("defaults"), "feed", "feed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := resolveOne(t, &config.Config{
				Defaults: config.Common{Dirname: tt.defaults, Filename: tt.defaults},
				Feeds: []config.Feed{{
					Url:    feedUrl,
					Common: config.Common{Dirname: tt.feed, Filename: tt.feed},
				}},
			})
			m := meta("Show", "https://cdn.example.com/ep12.mp3")
			gotDirname, err := f.dirname.Render(naming.FeedOnly{Feed: m.Feed})
			if err != nil {
				t.Fatalf("dirname.Render() = _, %v, want %q, nil", err, tt.wantDirname)
			}
			if gotDirname != tt.wantDirname {
				t.Errorf("dirname.Render() = %q, want %q", gotDirname, tt.wantDirname)
			}
			gotFilename, err := f.filename.Render(m)
			if err != nil {
				t.Fatalf("filename.Render() = _, %v, want %q, nil", err, tt.wantFilename)
			}
			if gotFilename != tt.wantFilename {
				t.Errorf("filename.Render() = %q, want %q", gotFilename, tt.wantFilename)
			}
		})
	}
}

func TestResolveSanitizePrecedence(t *testing.T) {
	const dirname = "{{.Feed.Title}}"
	tests := []struct {
		name           string
		feed, defaults *bool
		want           string
	}{
		{"built-in sanitizes", nil, nil, "Show- One"},
		{"defaults off", nil, new(false), "Show: One"},
		{"feed off", new(false), nil, "Show: One"},
		{"feed on over defaults off", new(true), new(false), "Show- One"},
		{"feed off over defaults on", new(false), new(true), "Show: One"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := resolveOne(t, &config.Config{
				Defaults: config.Common{Sanitize: tt.defaults},
				Feeds: []config.Feed{{
					Url:    feedUrl,
					Common: config.Common{Dirname: new(dirname), Sanitize: tt.feed},
				}},
			})
			got, err := f.dirname.Render(show("Show: One"))
			if err != nil {
				t.Fatalf("dirname.Render() = _, %v, want %q, nil", err, tt.want)
			}
			if got != tt.want {
				t.Errorf("dirname.Render() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveRejectsBadTemplates(t *testing.T) {
	tests := []struct {
		name string
		feed config.Common
		want string
	}{
		{"unknown field", config.Common{Dirname: new("{{.Feed.Titel}}")}, "dirname"},
		{"syntax error", config.Common{Filename: new("{{.Episode.Title")}, "filename"},
		{"unknown function", config.Common{Dirname: new("{{foobar .Feed.Title}}")}, "dirname"},
		{"absolute", config.Common{Dirname: new("/etc/{{.Feed.Title}}")}, "dirname"},
		{"traversal", config.Common{Filename: new("../{{.Episode.Filename}}")}, "filename"},
		{"names nothing", config.Common{Filename: new("")}, "filename"},
		{"dirname naming an episode date", config.Common{Dirname: new(`{{.Episode.PubDate | date "2006"}}`)}, "dirname"},
		{"dirname naming an episode title", config.Common{Dirname: new("{{.Episode.Title}}")}, "dirname"},
		{"dirname naming an episode among feed fields", config.Common{Dirname: new(`{{sanitize .Feed.Title}}/{{.Episode.PubDate | date "2006"}}`)}, "dirname"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feeds, err := resolve(&config.Config{Feeds: []config.Feed{{Url: feedUrl, Common: tt.feed}}})
			if err == nil {
				t.Fatalf("resolve() = %v, nil, want nil, an error", feeds)
			}
			if feeds != nil {
				t.Errorf("resolve() = %v alongside an error, want nil", feeds)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("resolve() = _, %v, want an error naming %q", err, tt.want)
			}
			if !strings.Contains(err.Error(), feedUrl) {
				t.Errorf("resolve() = _, %v, want an error naming the feed %q", err, feedUrl)
			}
		})
	}
}

func TestContained(t *testing.T) {
	// Render rejects the output that could escape a root,
	// so this is only ever reached if that is wrong.
	root := filepath.Join(t.TempDir(), "root")
	tests := []struct {
		name    string
		dest    string
		wantErr bool
	}{
		{"the root itself", root, false},
		{"a file in the root", filepath.Join(root, "ep.mp3"), false},
		{"a file below the root", filepath.Join(root, "Show", "2026", "ep.mp3"), false},
		{"a name that starts like the root", root + "-elsewhere", true},
		{"the parent", filepath.Dir(root), true},
		{"a sibling", filepath.Join(filepath.Dir(root), "elsewhere", "ep.mp3"), true},
		{"an unrelated absolute path", filepath.Join(t.TempDir(), "ep.mp3"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := contained(root, tt.dest)
			if tt.wantErr && err == nil {
				t.Errorf("contained(%q, %q) = nil, want an error", root, tt.dest)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("contained(%q, %q) = %v, want nil", root, tt.dest, err)
			}
		})
	}
}

func TestResolveAcceptsEpisodesInFilenames(t *testing.T) {
	const filename = `{{.Episode.PubDate | date "2006"}}/{{sanitize .Episode.Title}}{{.Episode.Ext}}`
	f := resolveOne(t, &config.Config{
		Feeds: []config.Feed{{Url: feedUrl, Common: config.Common{Filename: new(filename)}}},
	})
	m := meta("Show", "https://cdn.example.com/ep12.mp3")
	m.Episode.Title = "Episode 12"
	m.Episode.PubDate = time.Date(2026, time.August, 30, 9, 0, 0, 0, time.UTC)
	got, err := f.filename.Render(m)
	if err != nil {
		t.Fatalf("filename.Render() = _, %v, want a path, nil", err)
	}
	if want := filepath.Join("2026", "Episode 12.mp3"); got != want {
		t.Errorf("filename.Render() = %q, want %q", got, want)
	}
}

func TestShowDir(t *testing.T) {
	root := t.TempDir()
	tests := []struct {
		name, dirname, title, want string
	}{
		{"built-in template", "", "Show", "Show"},
		{"traversal is contained", "", "../../escape", "..-..-escape"},
		{"title that sanitizes to nothing falls back", "", "...", "show-fdaacbb9f51e9c6e"},
		{"nesting", "podcasts/{{sanitize .Feed.Title}}", "Show", filepath.Join("podcasts", "Show")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			common := config.Common{}
			if tt.dirname != "" {
				common.Dirname = new(tt.dirname)
			}
			f := resolveOne(t, &config.Config{Feeds: []config.Feed{{Url: feedUrl, Common: common}}})
			got, err := f.showDir(root, show(tt.title))
			if err != nil {
				t.Fatalf("showDir() = _, %v, want a directory, nil", err)
			}
			if want := filepath.Join(root, tt.want); got != want {
				t.Errorf("showDir() = %q, want %q", got, want)
			}
			if err := contained(root, got); err != nil {
				t.Errorf("showDir() = %q, which is not under %q: %v", got, root, err)
			}
		})
	}
}

func TestShowDirReportsAnUnusableName(t *testing.T) {
	f := resolveOne(t, &config.Config{Feeds: []config.Feed{{
		Url:    feedUrl,
		Common: config.Common{Dirname: new("{{.Feed.Title}}")},
	}}})
	got, err := f.showDir(t.TempDir(), show(""))
	if err == nil {
		t.Fatalf("showDir() for a feed with no title = %q, nil, want an error", got)
	}
	if got != "" {
		t.Errorf("showDir() = %q alongside an error, want an empty string", got)
	}
}

func TestEpisodeDest(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Show")
	tests := []struct {
		name, filename, episodeUrl, want string
	}{
		{"built-in template", "", "https://cdn.example.com/ep12.mp3", "ep12.mp3"},
		{"traversal is contained", "", "https://cdn.example.com/../../etc/passwd", "passwd"},
		{"no name falls back", "", "https://cdn.example.com/", "episode-f75d66546102fd87"},
		{"nesting", `{{.Episode.PubDate | date "2006"}}/{{.Episode.Filename}}`, "https://cdn.example.com/ep12.mp3", filepath.Join("2026", "ep12.mp3")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			common := config.Common{}
			if tt.filename != "" {
				common.Filename = new(tt.filename)
			}
			f := resolveOne(t, &config.Config{Feeds: []config.Feed{{Url: feedUrl, Common: common}}})
			m := meta("Show", tt.episodeUrl)
			m.Episode.PubDate = time.Date(2026, time.August, 30, 9, 0, 0, 0, time.UTC)
			got, err := f.episodeDest(dir, m)
			if err != nil {
				t.Fatalf("episodeDest() = _, %v, want a path, nil", err)
			}
			if want := filepath.Join(dir, tt.want); got != want {
				t.Errorf("episodeDest() = %q, want %q", got, want)
			}
			if err := contained(dir, got); err != nil {
				t.Errorf("episodeDest() = %q, which is not under %q: %v", got, dir, err)
			}
		})
	}
}

func TestEpisodeDestReportsAnUnusableName(t *testing.T) {
	f := resolveOne(t, &config.Config{Feeds: []config.Feed{{
		Url:    feedUrl,
		Common: config.Common{Filename: new("{{.Episode.Title}}")},
	}}})
	got, err := f.episodeDest(t.TempDir(), meta("Show", "https://cdn.example.com/ep12.mp3"))
	if err == nil {
		t.Fatalf("episodeDest() for an episode with no title = %q, nil, want an error", got)
	}
	if got != "" {
		t.Errorf("episodeDest() = %q alongside an error, want an empty string", got)
	}
}

func touch(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("creating %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, nil, 0644); err != nil {
		t.Fatalf("creating %s: %v", path, err)
	}
	return path
}

// withExisting resolves one feed with the named existing policy.
func withExisting(t *testing.T, policy string) resolvedFeed {
	t.Helper()
	return resolveOne(t, &config.Config{Feeds: []config.Feed{{
		Url:    feedUrl,
		Common: config.Common{Existing: new(policy)},
	}}})
}

func TestAvailableNoConflict(t *testing.T) {
	for _, policy := range []string{"increment", "skip", "replace"} {
		t.Run(policy, func(t *testing.T) {
			dest := filepath.Join(t.TempDir(), "ep12.mp3")
			got, free, err := withExisting(t, policy).available(dest)
			if err != nil {
				t.Fatalf("available(%q) = _, _, %v, want nil", dest, err)
			}
			if !free {
				t.Errorf("available(%q) reported the path taken, want it free", dest)
			}
			if got != dest {
				t.Errorf("available(%q) = %q, want the path unchanged", dest, got)
			}
		})
	}
}

func TestAvailableSkip(t *testing.T) {
	dest := touch(t, filepath.Join(t.TempDir(), "ep12.mp3"))
	got, free, err := withExisting(t, "skip").available(dest)
	if err != nil {
		t.Fatalf("available(%q) = _, _, %v, want nil", dest, err)
	}
	if free {
		t.Errorf("available(%q) reported a free path, want the episode skipped", dest)
	}
	if got != dest {
		t.Errorf("available(%q) = %q, want the path", dest, got)
	}
}

func TestAvailableReplace(t *testing.T) {
	dest := touch(t, filepath.Join(t.TempDir(), "ep12.mp3"))
	got, free, err := withExisting(t, "replace").available(dest)
	if err != nil {
		t.Fatalf("available(%q) = _, _, %v, want nil", dest, err)
	}
	if !free {
		t.Errorf("available(%q) reported the path taken, want it overwritten", dest)
	}
	if got != dest {
		t.Errorf("available(%q) = %q, want the path unchanged", dest, got)
	}
}

func TestAvailableIncrement(t *testing.T) {
	dir := t.TempDir()
	f := withExisting(t, "increment")
	dest := touch(t, filepath.Join(dir, "ep12.mp3"))
	for _, want := range []string{"ep12 (1).mp3", "ep12 (2).mp3", "ep12 (3).mp3"} {
		got, free, err := f.available(dest)
		if err != nil {
			t.Fatalf("available(%q) = _, _, %v, want nil", dest, err)
		}
		if !free {
			t.Fatalf("available(%q) reported no free name, want %q", dest, want)
		}
		if got != filepath.Join(dir, want) {
			t.Fatalf("available(%q) = %q, want %q", dest, got, filepath.Join(dir, want))
		}
		touch(t, got)
	}
}

func TestAvailableIncrementNumbersFile(t *testing.T) {
	dir := t.TempDir()
	f := withExisting(t, "increment")
	dest := touch(t, filepath.Join(dir, "2026", "ep12.mp3"))
	got, free, err := f.available(dest)
	if err != nil || !free {
		t.Fatalf("available(%q) = %q, %t, %v, want a free name, true, nil", dest, got, free, err)
	}
	if want := filepath.Join(dir, "2026", "ep12 (1).mp3"); got != want {
		t.Errorf("available(%q) = %q, want %q", dest, got, want)
	}
}

func TestAvailableIncrementBounded(t *testing.T) {
	dir := t.TempDir()
	f := withExisting(t, "increment")
	defer func(n int) { maxNumbered = n }(maxNumbered)
	maxNumbered = 4
	dest := touch(t, filepath.Join(dir, "ep12.mp3"))
	for n := 1; n <= maxNumbered; n++ {
		touch(t, filepath.Join(dir, naming.Numbered("ep12.mp3", n, true)))
	}
	got, free, err := f.available(dest)
	if err == nil {
		t.Fatalf("available(%q) = %q, %t, nil, want an error", dest, got, free)
	}
	if got != "" || free {
		t.Errorf("available(%q) = %q, %t alongside an error, want \"\", false", dest, got, free)
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("%q", dest)) {
		t.Errorf("available(%q) = _, _, %v, want an error naming the path", dest, err)
	}
}

func TestAvailableIncrementLongName(t *testing.T) {
	dir := t.TempDir()
	f := withExisting(t, "increment")
	long := strings.Repeat("a", fsname.MaxComponentLen-4) + ".mp3"
	dest := touch(t, filepath.Join(dir, long))
	got, free, err := f.available(dest)
	if err != nil || !free {
		t.Fatalf("available(<long>) = %q, %t, %v, want a free name, true, nil", got, free, err)
	}
	name := filepath.Base(got)
	if name == long {
		t.Error("available(<long>) = <input path>, want a name that differs from the one taken")
	}
	if len(name) > fsname.MaxComponentLen {
		t.Errorf("available(<long>) gave a %d byte name, want at most %d", len(name), fsname.MaxComponentLen)
	}
	if _, ok := fsname.Component(name); !ok {
		t.Errorf("available(<long>) = %q, want a usable component", name)
	}
}

func TestResolveExistingPrecedence(t *testing.T) {
	tests := []struct {
		name           string
		feed, defaults *string
		want           naming.Existing
	}{
		{"built-in", nil, nil, naming.Increment},
		{"defaults", nil, new("skip"), naming.Skip},
		{"feed", new("replace"), nil, naming.Replace},
		{"feed over defaults", new("replace"), new("skip"), naming.Replace},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := resolveOne(t, &config.Config{
				Defaults: config.Common{Existing: tt.defaults},
				Feeds:    []config.Feed{{Url: feedUrl, Common: config.Common{Existing: tt.feed}}},
			})
			if f.existing != tt.want {
				t.Errorf("resolve() gave existing = %v, want %v", f.existing, tt.want)
			}
		})
	}
}

func TestResolveRejectsUnknownExisting(t *testing.T) {
	feeds, err := resolve(&config.Config{Feeds: []config.Feed{{
		Url:    feedUrl,
		Common: config.Common{Existing: new("clobber")},
	}}})
	if err == nil {
		t.Fatalf("resolve() = %v, nil, want an error", feeds)
	}
	for _, want := range []string{"existing", feedUrl, "clobber"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("resolve() = _, %v, want an error naming %q", err, want)
		}
	}
}

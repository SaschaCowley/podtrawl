package naming

import (
	"mime"
	"strings"
	"testing"
	"time"

	"ssch.cc/podtrawl/internal/feed"
)

const episodeUrl = "https://cdn.example.com/ep12.mp3"

func TestSplitName(t *testing.T) {
	tests := []struct{ name, filename, wantStem, wantExt string }{
		{"ordinary filename", "episode.mp3", "episode", ".mp3"},
		{"No extension", "episode", "episode", ""},
		{"leading dot", ".episode", ".episode", ""},
		{"leading dot with extension", ".episode.mp3", ".episode", ".mp3"},
		{"leading dots", "..episode", ".", ".episode"},
		{"trailing dot", "episode.", "episode.", ""},
		{"trailing dots", "episode..", "episode..", ""},
		{"leading and trailing dot", ".episode.", ".episode.", ""},
		{"two extensions", "episode.tar.gz", "episode.tar", ".gz"},
		{"two separators", "episode..mp3", "episode.", ".mp3"},
		{"empty name", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStem, gotExt := splitName(tt.filename)
			if wantStem := tt.wantStem; wantStem != gotStem {
				t.Errorf("splitName(%q) = %q _, want %q _", tt.name, gotStem, wantStem)
			}
			if wantExt := tt.wantExt; wantExt != gotExt {
				t.Errorf("splitName(%q) = _ %q, want _ %q", tt.name, gotExt, wantExt)
			}
		})
	}
}

func TestBaseName(t *testing.T) {
	tests := []struct{ name, url, want string }{
		{"plane file", "https://example.com/episode.mp3", "episode.mp3"},
		{"plane dir", "https://example.com/episode/", "episode"},
		{"file with query", "https://example.com/episode.mp3?abc=def&ghi=jkl", "episode.mp3"},
		{"dir with query", "https://example.com/episode/?mno=pqr&stu=vwx", "episode"},
		{"file with anchor", "https://example.com/episode.mp3#y", "episode.mp3"},
		{"dir with anchor", "https://example.com/episode/#z", "episode"},
		{"empty path", "https://example.com/", ""},
		{"no path", "https://example.com", ""},
		{"not a URL", "https:/", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := baseName(tt.url); got != tt.want {
				t.Errorf("baseName(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestExtFromType(t *testing.T) {
	tests := []struct{ name, mime, ext string }{
		{"typical type", "audio/mpeg", ".mp3"},
		{"type with parameters", "audio/ogg; codecs=vorbis", ".ogg"},
		{"including capitalization", "Audio/WAV", ".wav"},
		{"mp4 audio", "audio/mp4", ".m4a"},
		{"apple mp4 audio", "audio/x-m4a", ".m4a"},
		{"invalid mime", "not a mime", ""},
		{"empty", "", ""},
		{"unregistered type", "audio/x-podtrawl-nonexistant", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extFromType(tt.mime); got != tt.ext {
				t.Errorf("extFromType(%q) = %q, want %q", tt.mime, got, tt.ext)
			}
		})
	}
}

func TestExtFromTypeFallsBackToTheSystemTable(t *testing.T) {
	// Types the table does not list are still worth an answer if the system has one.
	// Registering a type of our own keeps this independent
	// of whatever the host happens to have installed.
	const mediaType = "audio/x-podtrawl-registered"
	const extension = ".podtrawl"
	if err := mime.AddExtensionType(extension, mediaType); err != nil {
		t.Fatalf("mime.AddExtensionType() = %v, want nil", err)
	}
	if got := extFromType(mediaType); got != extension {
		t.Errorf("extFromType(%q) = %q, want %q", mediaType, got, extension)
	}
}

func TestNameParts(t *testing.T) {
	tests := []struct{ name, url, mediaType, wantName, wantStem, wantExt string }{
		{"ext but no mime", "https://example.com/episode.mp3", "", "episode.mp3", "episode", ".mp3"},
		{"ext and congruent mime", "https://example.com/episode.mp3", "audio/mpeg", "episode.mp3", "episode", ".mp3"},
		{"ext and incongruent mime", "https://example.com/episode.mp3", "audio/vnd.wave", "episode.mp3", "episode", ".mp3"},
		{"neither ext nor mime", "https://example.com/episode", "", "episode", "episode", ""},
		{"no ext but mime", "https://example.com/episode", "audio/mpeg", "episode", "episode", ".mp3"},
		{"neither name nor mime", "https://example.com/", "", "", "", ""},
		{"No name but mime", "https://example.com/", "audio/mpeg", "", "", ".mp3"},
		{"query string is ignored", "https://example.com/episode.mp3?token=abc&e=99", "audio/mpeg", "episode.mp3", "episode", ".mp3"},
		{"percent encoding is decoded", "https://example.com/episode%2012.mp3", "audio/mpeg", "episode 12.mp3", "episode 12", ".mp3"},
		{"only the last path element is used", "https://example.com/shows/1/episode.mp3", "audio/mpeg", "episode.mp3", "episode", ".mp3"},
		{"no sanitization", "https://example.com/a:b.mp3", "audio/mpeg", "a:b.mp3", "a:b", ".mp3"},
		{"no path at all", "https://cdn.example.com", "audio/mpeg", "", "", ".mp3"},
		{"unparseable url", "://not a url", "audio/mpeg", "", "", ".mp3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotStem, gotExt := nameParts(tt.url, tt.mediaType)
			if gotName != tt.wantName {
				t.Errorf("nameParts(%q, %q) gave name %q, want %q", tt.url, tt.mediaType, gotName, tt.wantName)
			}
			if gotStem != tt.wantStem {
				t.Errorf("nameParts(%q, %q) gave stem %q, want %q", tt.url, tt.mediaType, gotStem, tt.wantStem)
			}
			if gotExt != tt.wantExt {
				t.Errorf("nameParts(%q, %q) gave ext %q, want %q", tt.url, tt.mediaType, gotExt, tt.wantExt)
			}
		})
	}
}

func TestNewFeedMeta(t *testing.T) {
	const url = "https://example.com/podcast.rss"
	c := feed.Channel{
		Title:       "Test Podcast",
		Link:        "https://example.com/",
		Description: "A podcast for testing",
	}
	m := NewFeedMeta(c, url)
	if m.Url != url {
		t.Errorf("NewFeedMeta() gave Url = %q, want %q", m.Url, url)
	}
	if m.Title != c.Title {
		t.Errorf("NewFeedMeta() gave Title = %q, want %q", m.Title, c.Title)
	}
	if m.Link != c.Link {
		t.Errorf("NewFeedMeta() gave Link = %q, want %q", m.Link, c.Link)
	}
	if m.Description != c.Description {
		t.Errorf("NewFeedMeta() gave Description = %q, want %q", m.Description, c.Description)
	}
}

func checkName(t *testing.T, m EpisodeMeta, extFromName bool) {
	t.Helper()
	if m.Filename == "" && m.Stem != "" {
		t.Errorf("Filename is empty alongside Stem = %q, want an empty stem", m.Stem)
	}
	if extFromName {
		if want := m.Stem + m.Ext; m.Filename != want {
			t.Errorf("Filename = %q, want Stem+Ext = Filename, got %q", m.Filename, want)
		}
	}
	wantFilename, wantStem, wantExt := nameParts(m.Url, m.Type)
	if wantFilename != m.Filename {
		t.Errorf("Filename = %q, want %q", m.Filename, wantFilename)
	}
	if wantStem != m.Stem {
		t.Errorf("Stem = %q, want %q", m.Stem, wantStem)
	}
	if wantExt != m.Ext {
		t.Errorf("Ext = %q, want %q", m.Ext, wantExt)
	}
}

func TestNewEpisodeMeta(t *testing.T) {
	pubDate := time.Date(2026, time.August, 30, 9, 0, 0, 0, time.UTC)
	item := feed.Item{
		Title:       new("Episode 12"),
		Link:        new("https://example.com/12"),
		Description: new("The twelfth episode"),
		Guid:        &feed.Guid{Value: "tag:example.com,2026:12"},
		PubDate:     &feed.Rfc2822Datetime{Time: pubDate},
	}
	enclosure := feed.Enclosure{Url: episodeUrl, Type: "audio/mpeg", Length: 1234}
	m := NewEpisodeMeta(item, enclosure)
	checkName(t, m, true)
	if m.Title != "Episode 12" {
		t.Errorf("gave Title = %q, want %q", m.Title, "Episode 12")
	}
	if m.Link != "https://example.com/12" {
		t.Errorf("gave Link = %q, want %q", m.Link, "https://example.com/12")
	}
	if m.Description != "The twelfth episode" {
		t.Errorf("gave Description = %q, want %q", m.Description, "The twelfth episode")
	}
	if m.Guid != "tag:example.com,2026:12" {
		t.Errorf("gave Guid = %q, want %q", m.Guid, "tag:example.com,2026:12")
	}
	if !m.PubDate.Equal(pubDate) {
		t.Errorf("gave PubDate = %v, want %v", m.PubDate, pubDate)
	}
	if m.Url != enclosure.Url {
		t.Errorf("gave Url = %q, want %q", m.Url, enclosure.Url)
	}
	if m.Type != enclosure.Type {
		t.Errorf("gave Type = %q, want %q", m.Type, enclosure.Type)
	}
	if m.Length != enclosure.Length {
		t.Errorf("gave Length = %d, want %d", m.Length, enclosure.Length)
	}
}

func TestNewEpisodeMetaOmittedElements(t *testing.T) {
	m := NewEpisodeMeta(feed.Item{}, feed.Enclosure{})
	for _, f := range []struct{ name, got string }{
		{"Title", m.Title},
		{"Link", m.Link},
		{"Description", m.Description},
		{"Guid", m.Guid},
		{"Type", m.Type},
		{"Url", m.Url},
		{"Filename", m.Filename},
		{"Stem", m.Stem},
		{"Ext", m.Ext},
	} {
		if f.got != "" {
			t.Errorf("an item with nothing set gave %s = %q, want an empty string", f.name, f.got)
		}
	}
	if !m.PubDate.IsZero() {
		t.Errorf("an item with no pubDate gave PubDate = %v, want the zero time", m.PubDate)
	}
	if m.Length != 0 {
		t.Errorf("an enclosure with no length gave Length = %d, want 0", m.Length)
	}
}

func TestMimeToExt(t *testing.T) {
	for mediaType, ext := range mimeToExt {
		t.Run(mediaType, func(t *testing.T) {
			if !strings.HasPrefix(ext, ".") {
				t.Errorf("mimeToExt[%q] = %q, want a leading dot", mediaType, ext)
			}
			canonical, _, err := mime.ParseMediaType(mediaType)
			if err != nil {
				t.Fatalf("mime.ParseMediaType(%q) = _, _, %v, want nil", mediaType, err)
			}
			if canonical != mediaType {
				t.Errorf("key %q is not canonical; mime.ParseMediaType gives %q", mediaType, canonical)
			}
			if got := extFromType(mediaType); got != ext {
				t.Errorf("extFromType(%q) = %q, want %q", mediaType, got, ext)
			}
		})
	}
}

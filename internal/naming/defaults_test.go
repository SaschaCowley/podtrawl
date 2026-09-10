package naming

import (
	"fmt"
	"strings"
	"testing"

	"ssch.cc/podtrawl/internal/feed"
	"ssch.cc/podtrawl/internal/fsname"
)

func TestDefaultTemplates(t *testing.T) {
	const feedUrl = "https://example.com/feed.xml"
	long := strings.Repeat("a", 300)
	tests := []defaultCase{
		{"title and name in url", "Test Podcast", "https://cdn.example.com/shows/1/episode.mp3", "", "Test Podcast", "episode.mp3"},
		{"query string excluded from name", "Test Podcast", "https://cdn.example.com/ep12.mp3?token=abc&e=99", "", "Test Podcast", "ep12.mp3"},
		{"unusable characters replaced", "AC/DC: Live?", "https://cdn.example.com/a:b.mp3", "", "AC-DC- Live-", "a-b.mp3"},
		{"empty title falls back to feed url", "", "https://cdn.example.com/ep12.mp3", "", "show-2e86815cbc504197", "ep12.mp3"},
		{"no name falls back to episode url", "Test Podcast", "https://cdn.example.com/", "", "Test Podcast", "episode-f75d66546102fd87"},
		{"fallback name keeps the type's extension", "Test Podcast", "https://cdn.example.com/", "audio/mpeg", "Test Podcast", "episode-f75d66546102fd87.mp3"},
		{"unusable stem keeps the url's extension", "Test Podcast", "https://cdn.example.com/....mp3", "", "Test Podcast", "episode-e1be5fafa3b145eb.mp3"},
		{"title traversal contained", "../../escape", "https://cdn.example.com/ep12.mp3", "", "..-..-escape", "ep12.mp3"},
		{"overlong names truncated", long, "https://cdn.example.com/" + long + ".mp3", "", strings.Repeat("a", fsname.MaxComponentLen), strings.Repeat("a", fsname.MaxComponentLen-len(".mp3")) + ".mp3"},
	}
	for _, sanitize := range []bool{true, false} {
		t.Run(fmt.Sprintf("sanitize=%t", sanitize), func(t *testing.T) {
			testDefaultTemplates(t, sanitize, feedUrl, tests)
		})
	}
}

// defaultCase is one feed and episode put through both built-in templates.
type defaultCase struct {
	name, channelTitle, episodeUrl, episodeType, wantDirname, wantFilename string
}

func testDefaultTemplates(t *testing.T, sanitize bool, feedUrl string, tests []defaultCase) {
	dirname, err := ParseFeed("dirname", DefaultDirname, sanitize)
	if err != nil {
		t.Fatalf("ParseFeed(\"dirname\", DefaultDirname, %t) = _, %v, want Template, nil", sanitize, err)
	}
	filename, err := ParseEpisode("filename", DefaultFilename, sanitize)
	if err != nil {
		t.Fatalf("ParseEpisode(\"filename\", DefaultFilename, %t) = _, %v, want Template, nil", sanitize, err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Metadata{
				Feed:    NewFeedMeta(feed.Channel{Title: tt.channelTitle}, feedUrl),
				Episode: NewEpisodeMeta(feed.Item{}, feed.Enclosure{Url: tt.episodeUrl, Type: tt.episodeType}),
			}
			gotDirname, err := dirname.Render(FeedOnly{Feed: m.Feed})
			if err != nil {
				t.Fatalf("dirname.Render() = _, %v, want %q, nil", err, tt.wantDirname)
			}
			if gotDirname != tt.wantDirname {
				t.Errorf("dirname.Render() = %q, want %q", gotDirname, tt.wantDirname)
			}
			gotFilename, err := filename.Render(m)
			if err != nil {
				t.Fatalf("filename.Render() = _, %v, want %q, nil", err, tt.wantFilename)
			}
			if gotFilename != tt.wantFilename {
				t.Errorf("filename.Render() = %q, want %q", gotFilename, tt.wantFilename)
			}
			// Whatever a feed says, a rendered name is one usable component.
			for _, got := range []string{gotDirname, gotFilename} {
				if _, ok := fsname.Component(got); !ok {
					t.Errorf("rendered %q, want a usable path component", got)
				}
			}
		})
	}
}

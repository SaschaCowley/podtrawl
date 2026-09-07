package naming

import (
	"testing"

	"ssch.cc/podtrawl/internal/feed"
	"ssch.cc/podtrawl/internal/fsname"
)

func TestDefaultTemplates(t *testing.T) {
	const feedUrl = "https://example.com/feed.xml"
	tests := []struct{ name, channelTitle, episodeUrl, wantDirname, wantFilename string }{
		{"title and name in url", "Test Podcast", "https://cdn.example.com/shows/1/episode.mp3", "Test Podcast", "episode.mp3"},
		{"query string excluded from name", "Test Podcast", "https://cdn.example.com/ep12.mp3?token=abc&e=99", "Test Podcast", "ep12.mp3"},
		{"unusable characters replaced", "AC/DC: Live?", "https://cdn.example.com/a:b.mp3", "AC-DC- Live-", "a-b.mp3"},
		{"empty title falls back to feed url", "", "https://cdn.example.com/ep12.mp3", "show-2e86815cbc504197", "ep12.mp3"},
		{"no name falls back to episode url", "Test Podcast", "https://cdn.example.com/", "Test Podcast", "episode-f75d66546102fd87"},
		{"title traversal contained", "../../escape", "https://cdn.example.com/ep12.mp3", "..-..-escape", "ep12.mp3"},
	}
	dirname, err := ParseFeed("dirname", DefaultDirname, true)
	if err != nil {
		t.Fatalf("ParseFeed(\"dirname\", DefaultDirname, true) = _, %v, want Template, nil", err)
	}
	filename, err := ParseEpisode("filename", DefaultFilename, true)
	if err != nil {
		t.Fatalf("ParseEpisode(\"filename\", DefaultFilename, true) = _, %v, want Template, nil", err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Metadata{
				Feed:    NewFeedMeta(feed.Channel{Title: tt.channelTitle}, feedUrl),
				Episode: NewEpisodeMeta(feed.Item{}, feed.Enclosure{Url: tt.episodeUrl}),
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

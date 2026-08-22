package app

import "testing"

func TestEpisodeFileName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain url", "https://cdn.example.com/shows/1/episode.mp3", "episode.mp3"},
		{"query string is ignored", "https://cdn.example.com/ep12.mp3?token=abc&e=99", "ep12.mp3"},
		{"percent encoding is decoded", "https://cdn.example.com/ep%2012.mp3", "ep 12.mp3"},
		{"illegal characters are replaced", "https://cdn.example.com/a:b.mp3", "a-b.mp3"},
		{"traversal is contained", "https://cdn.example.com/../../etc/passwd", "passwd"},
		{
			"no path to use",
			"https://cdn.example.com/",
			"episode-" + shortHash("https://cdn.example.com/"),
		},
		{
			"unparseable url",
			"://not a url",
			"episode-" + shortHash("://not a url"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := episodeFileName(tt.input); got != tt.want {
				t.Errorf("episodeFileName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestShortHash(t *testing.T) {
	const url = "https://example.com/feed.xml"
	got := shortHash(url)
	if len(got) != 16 {
		t.Errorf("shortHash(%q) = %q, want 16 characters", url, got)
	}
	if again := shortHash(url); again != got {
		t.Errorf("shortHash(%q) = %q on a second call, want %q", url, again, got)
	}
	if other := shortHash(url + "x"); other == got {
		t.Errorf("shortHash collided on distinct inputs, both %q", got)
	}
}

package app

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"ssch.cc/podtrawl/internal/config"
	"ssch.cc/podtrawl/internal/feed"
)

func TestRootDir(t *testing.T) {
	type testCase struct {
		name    string
		root    string
		want    string
		wantErr bool
	}
	abs := t.TempDir()
	tests := []testCase{
		{name: "absolute path", root: abs, want: abs},
		{name: "parent is resolved", root: abs + "/sub/..", want: abs},
		{name: "current is resolved", root: abs + "/./", want: abs},
		{name: "empty", root: "", wantErr: true},
		{name: "relative", root: "podcasts", wantErr: true},
		{name: "working directory", root: ".", wantErr: true},
		{name: "tilde is not expanded", root: "~/Podcasts", wantErr: true},
	}
	if runtime.GOOS == "windows" {
		// Local absolute paths on Windows start with a volume.
		tests = append(tests, testCase{name: "unix-style path", root: "/podcasts", wantErr: true})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rootDir(&config.Config{Root: &tt.root})
			if tt.wantErr {
				if err == nil {
					t.Fatalf("rootDir(root=%q) = %q, nil, want an error", tt.root, got)
				}
				if got != "" {
					t.Errorf("rootDir(root=%q) = %q alongside an error, want an empty string", tt.root, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("rootDir(root=%q) = _, %v, want nil", tt.root, err)
			}
			if !filepath.IsAbs(got) {
				t.Errorf("rootDir(root=%q) = %q, want an absolute path", tt.root, got)
			}
			if got != tt.want {
				t.Errorf("rootDir(root=%q) = %q, want %q", tt.root, got, tt.want)
			}
		})
	}
}

func TestRootDirUnset(t *testing.T) {
	want, err := downloadRoot()
	if err != nil {
		t.Skipf("no downloads directory on this system: %v", err)
	}
	got, err := rootDir(&config.Config{})
	if err != nil {
		t.Fatalf("rootDir(root unset) = _, %v, want nil", err)
	}
	if got != want {
		t.Errorf("rootDir(root unset) = %q, want %q", got, want)
	}
}

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

func TestEpisodeKey(t *testing.T) {
	const enclosureUrl = "https://cdn.example.com/ep12.mp3"
	tests := []struct {
		name string
		guid *feed.Guid
		want string
	}{
		{"guid", &feed.Guid{Value: "tag:example.com,2026:12"}, "tag:example.com,2026:12"},
		{"no guid", nil, enclosureUrl},
		{"empty guid", &feed.Guid{}, enclosureUrl},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enclosure := feed.Enclosure{Url: enclosureUrl}
			item := feed.Item{Guid: tt.guid, Enclosures: []feed.Enclosure{enclosure}}
			if got := episodeKey(item, enclosure); got != tt.want {
				t.Errorf("episodeKey(%#v, %#v) = %q, want %q", item, enclosure, got, tt.want)
			}
		})
	}
}

func TestDownloadEpisodeReportsStatus(t *testing.T) {
	const name = "ep.mp3"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gone", http.StatusNotFound)
	}))
	defer srv.Close()

	base := t.TempDir()
	enclosure := feed.Enclosure{Url: srv.URL + "/" + name}
	if err := DownloadEpisode(t.Context(), slog.Default(), filepath.Join(base, name), enclosure); err == nil {
		t.Error("DownloadEpisode() with a 404 response returned nil, want an error")
	}
}

func TestDownloadEpisodeWritesToDest(t *testing.T) {
	const name = "ep.mp3"
	const body = "audio"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, body)
	}))
	defer srv.Close()

	// Using a nested dest ensures that
	// DownloadEpisode properly handles being given a complete path,
	// rather than one without the filename.
	dest := filepath.Join(t.TempDir(), "Show", "2026", name)
	enclosure := feed.Enclosure{Url: srv.URL + "/" + name}
	if err := DownloadEpisode(t.Context(), slog.Default(), dest, enclosure); err != nil {
		t.Fatalf("DownloadEpisode() = %v, want nil", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("DownloadEpisode() did not write to %q: %v", dest, err)
	}
	if string(got) != body {
		t.Errorf("contents of %q = %q, want %q", dest, got, body)
	}
}

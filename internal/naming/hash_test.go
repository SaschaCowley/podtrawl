package naming

import "testing"

func TestShortHash(t *testing.T) {
	const url = "https://example.com/feed.xml"
	got := ShortHash(url)
	if len(got) != 16 {
		t.Errorf("ShortHash(%q) = %q, want 16 characters", url, got)
	}
	if again := ShortHash(url); again != got {
		t.Errorf("ShortHash(%q) = %q on a second call, want %q", url, again, got)
	}
	if other := ShortHash(url + "x"); other == got {
		t.Errorf("ShortHash collided on distinct inputs, both %q", got)
	}
}

package feed

import (
	"strings"
	"testing"
	"time"
)

func TestItunesTypesAreNamespaced(t *testing.T) {
	t.Run("ItunesChannel", func(t *testing.T) { checkNamespace[ItunesChannel](t, nsItunes) })
	t.Run("ItunesItem", func(t *testing.T) { checkNamespace[ItunesItem](t, nsItunes) })
	t.Run("ItunesCategory", func(t *testing.T) { checkNamespace[ItunesCategory](t, nsItunes) })
	t.Run("ItunesOwner", func(t *testing.T) { checkNamespace[ItunesOwner](t, nsItunes) })
}

func TestLoadFillsBothSidesOfCollidingNames(t *testing.T) {
	r, err := Load(strings.NewReader(collidingFeed))
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	c := r.Channel

	t.Run("channel", func(t *testing.T) {
		checkString(t, "Title", c.Title, "Channel Title")
		checkStringPtr(t, "Itunes.Title", c.Itunes.Title, "iTunes Channel Title")
		if len(c.Itunes.Categories) != 1 {
			t.Fatalf("Itunes.Categories gave %d entries (%+v), want 1", len(c.Itunes.Categories), c.Itunes.Categories)
		}
		cat := c.Itunes.Categories[0]
		checkString(t, "Itunes.Categories[0].Text", cat.Text, "Technology")
		if len(cat.Subcategories) != 1 {
			t.Fatalf("Itunes.Categories[0].Subcategories gave %d entries (%+v), want 1", len(cat.Subcategories), cat.Subcategories)
		}
		checkString(t, "Itunes.Categories[0].Subcategories[0].Text", cat.Subcategories[0].Text, "Gadgets")
	})

	t.Run("item", func(t *testing.T) {
		if len(c.Items) != 1 {
			t.Fatalf("Items gave %d entries, want 1", len(c.Items))
		}
		i := c.Items[0]
		checkStringPtr(t, "Title", i.Title, "179: Episode Title")
		checkStringPtr(t, "Itunes.Title", i.Itunes.Title, "Episode Title")
		checkStringPtr(t, "Author", i.Author, "editor@example.com")
		checkStringPtr(t, "Itunes.Author", i.Itunes.Author, "Display Name")
	})
}

func TestLoadToleratesMalformedItunesValues(t *testing.T) {
	const f = `<rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd">
	 <channel>
	  <title>Channel Title</title>
	  <link>https://example.com/</link>
	  <description>Channel Description</description>
	  <itunes:explicit>sometimes</itunes:explicit>
	  <item>
	   <title>Episode Title</title>
	   <itunes:episode>twelve</itunes:episode>
	   <itunes:season></itunes:season>
	   <itunes:duration>about an hour</itunes:duration>
	  </item>
	 </channel>
	</rss>`
	r, err := Load(strings.NewReader(f))
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	if got := r.Channel.Itunes.Explicit; got != ExplicitUnknown {
		t.Errorf("Itunes.Explicit gave %v, want ExplicitUnknown", got)
	}
	if len(r.Channel.Items) != 1 {
		t.Fatalf("Items gave %d entries, want 1", len(r.Channel.Items))
	}
	i := r.Channel.Items[0]
	checkStringPtr(t, "Title", i.Title, "Episode Title")
	if i.Itunes.Episode != 0 {
		t.Errorf("Itunes.Episode gave %d, want 0", i.Itunes.Episode)
	}
	if i.Itunes.Season != 0 {
		t.Errorf("Itunes.Season gave %d, want 0", i.Itunes.Season)
	}
	if i.Itunes.Duration.Duration != 0 {
		t.Errorf("Itunes.Duration gave %v, want 0", i.Itunes.Duration)
	}
}

func TestItunesExplicit(t *testing.T) {
	tests := []struct {
		input string
		want  ItunesExplicit
	}{
		{"true", ExplicitYes},
		{"yes", ExplicitYes},
		{"explicit", ExplicitYes},
		{" True ", ExplicitYes},
		{"false", ExplicitNo},
		{"no", ExplicitNo},
		{"clean", ExplicitNo},
		{"NO", ExplicitNo},
		{"", ExplicitUnknown},
		{"maybe", ExplicitUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			e := ExplicitYes // Must be overwritten, even by junk.
			if tt.want == ExplicitYes {
				e = ExplicitNo
			}
			if err := e.UnmarshalText([]byte(tt.input)); err != nil {
				t.Fatalf("UnmarshalText(%q) error = %v, want nil", tt.input, err)
			}
			if e != tt.want {
				t.Errorf("UnmarshalText(%q) = %v, want %v", tt.input, e, tt.want)
			}
		})
	}
}

func TestItunesNumber(t *testing.T) {
	tests := []struct {
		input string
		want  ItunesNumber
	}{
		{"1", 1},
		{"42", 42},
		{" 7 ", 7},
		{"3.0", 3},
		{"", 0},
		{"0", 0},
		{"-2", 0},
		{"1.5", 0},
		{"twelve", 0},
		{"NaN", 0},
		{"1e100", 0},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			n := ItunesNumber(99) // Must be overwritten, even by junk.
			if err := n.UnmarshalText([]byte(tt.input)); err != nil {
				t.Fatalf("UnmarshalText(%q) error = %v, want nil", tt.input, err)
			}
			if n != tt.want {
				t.Errorf("UnmarshalText(%q) = %d, want %d", tt.input, n, tt.want)
			}
		})
	}
}

func TestItunesDuration(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"1079", 1079 * time.Second},
		{" 1079 ", 1079 * time.Second},
		{"1079.5", 1079*time.Second + 500*time.Millisecond},
		{"13:24", 13*time.Minute + 24*time.Second},
		{"1:02:03", time.Hour + 2*time.Minute + 3*time.Second},
		{"01:02:03.25", time.Hour + 2*time.Minute + 3*time.Second + 250*time.Millisecond},
		{"", 0},
		{"about an hour", 0},
		{"1:2:3:4", 0},
		{"1.5:00", 0},
		{"-5", 0},
		{"12:", 0},
		{"NaN", 0},
		{"Inf", 0},
		{"1e300", 0},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d := ItunesDuration{time.Hour} // Must be overwritten, even by junk.
			if err := d.UnmarshalText([]byte(tt.input)); err != nil {
				t.Fatalf("UnmarshalText(%q) error = %v, want nil", tt.input, err)
			}
			if d.Duration != tt.want {
				t.Errorf("UnmarshalText(%q) = %v, want %v", tt.input, d.Duration, tt.want)
			}
		})
	}
}

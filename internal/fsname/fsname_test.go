package fsname

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestComponent(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   string
		wantOk bool
	}{
		{"ordinary title", "The Rest Is History", "The Rest Is History", true},
		{"punctuation is kept", "Ep. 12 - Foo, Bar & Baz!", "Ep. 12 - Foo, Bar & Baz!", true},
		{"non-ascii is kept", "Café Society – 日本", "Café Society – 日本", true},

		{"forward slash", "AC/DC", "AC-DC", true},
		{"backslash", `a\b`, "a-b", true},
		{"colon", "Serial: Season 1", "Serial- Season 1", true},
		{"semicolon", "a;b", "a-b", true},
		{"other windows characters", `<>"|?*`, "------", true},
		{"nul", "a\x00b", "a-b", true},
		{"control characters", "a\tb\nc", "a-b-c", true},
		{"del", "a\x7fb", "a-b", true},
		{"everything at once", "AC/DC: Live? <1980>", "AC-DC- Live- -1980-", true},

		{"dot", ".", "", false},
		{"dot dot", "..", "", false},
		{"many dots", "...", "", false},
		{"traversal", "../../etc/passwd", "..-..-etc-passwd", true},
		{"windows traversal", `..\..\windows`, "..-..-windows", true},

		{"trailing dot", "foo.", "foo", true},
		{"trailing dots and spaces", "foo. . ", "foo", true},
		{"surrounding spaces", "  foo  ", "foo", true},
		{"leading dot is kept", ".hidden", ".hidden", true},

		{"empty", "", "", false},
		{"only spaces", "   ", "", false},
		{"only separators", "///", "---", true},

		{"reserved", "NUL", "NUL_", true},
		{"reserved lower case", "lpt1", "lpt1_", true},
		{"reserved mixed case", "Com1", "Com1_", true},
		{"reserved with extension", "CON.mp3", "CON_.mp3", true},
		{"reserved with two extensions", "aux.tar.gz", "aux_.tar.gz", true},
		{"reserved padded with spaces", "PRN .mp3", "PRN _.mp3", true},
		{"reserved-looking but longer", "COM10", "COM10", true},
		{"reserved-looking prefix", "CONSOLE", "CONSOLE", true},

		{"exactly at the limit", strings.Repeat("a", 255), strings.Repeat("a", 255), true},
		{"too long", strings.Repeat("a", 300), strings.Repeat("a", 255), true},
		{
			"too long keeps the extension",
			strings.Repeat("a", 300) + ".mp3",
			strings.Repeat("a", 251) + ".mp3",
			true,
		},
		{
			"too long with an implausible extension",
			strings.Repeat("b", 250) + ".thisisnotanextension",
			strings.Repeat("b", 250) + ".this",
			true,
		},
		{
			"truncation exposing trailing spaces",
			strings.Repeat("a", 250) + strings.Repeat(" ", 20) + strings.Repeat("b", 50),
			strings.Repeat("a", 250),
			true,
		},
		{
			"too long and multibyte",
			strings.Repeat("é", 200),
			strings.Repeat("é", 127),
			true,
		},
		{
			"too long and reserved",
			"NUL." + strings.Repeat("a", 300),
			"NUL_." + strings.Repeat("a", 250),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotOk := Component(tt.input)
			if got != tt.want || gotOk != tt.wantOk {
				t.Errorf("Component(%q) = %q, %v, want %q, %v", tt.input, got, gotOk, tt.want, tt.wantOk)
			}
			// Whatever the input, the result must be usable as a single
			// component on every target file system.
			if len(got) > maxComponentLen {
				t.Errorf("Component(%q) is %d bytes, want at most %d", tt.input, len(got), maxComponentLen)
			}
			if !utf8.ValidString(got) {
				t.Errorf("Component(%q) = %q, which is not valid UTF-8", tt.input, got)
			}
			if strings.ContainsAny(got, illegal) {
				t.Errorf("Component(%q) = %q, which contains an illegal character", tt.input, got)
			}
			if got == "." || got == ".." {
				t.Errorf("Component(%q) = %q, which traverses the parent directory", tt.input, got)
			}
		})
	}
}

func TestComponentOr(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		fallback string
		want     string
	}{
		{"usable name", "Foo", "fallback", "Foo"},
		{"unusable name", "..", "fallback", "fallback"},
		{"fallback is sanitized too", "", "a/b", "a-b"},
		{"neither is usable", "", "..", "_"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ComponentOr(tt.input, tt.fallback); got != tt.want {
				t.Errorf("ComponentOr(%q, %q) = %q, want %q", tt.input, tt.fallback, got, tt.want)
			}
		})
	}
}

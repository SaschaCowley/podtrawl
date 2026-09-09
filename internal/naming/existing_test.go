package naming

import (
	"maps"
	"strings"
	"testing"

	"ssch.cc/podtrawl/internal/fsname"
)

func TestParseExisting(t *testing.T) {
	tests := []struct {
		name, value string
		want        Existing
		wantErr     bool
	}{
		{"increment", "increment", Increment, false},
		{"skip", "skip", Skip, false},
		{"replace", "replace", Replace, false},
		{"unknown", "clobber", 0, true},
		{"empty", "", 0, true},
		{"wrong case", "Skip", 0, true},
		{"Go name", "Existing(1)", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseExisting(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseExisting(%q) = %v, nil, want an error", tt.value, got)
				}
				for name := range maps.Keys(existingNames) {
					if !strings.Contains(err.Error(), name) {
						t.Errorf("ParseExisting(%q) = _, %v, want an error naming %q", tt.value, err, name)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseExisting(%q) = _, %v, want %v, nil", tt.value, err, tt.want)
			}
			if got != tt.want {
				t.Errorf("ParseExisting(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestExistingDefaultsToIncrement(t *testing.T) {
	var zero Existing
	if zero != Increment {
		t.Errorf("the zero Existing is %v, want %v", zero, Increment)
	}
}

func TestExistingString(t *testing.T) {
	for name, want := range existingNames {
		if got := want.String(); got != name {
			t.Errorf("Existing(%v).String() = %q, want %q", int(want), got, name)
		}
	}
	if got := Existing(99).String(); !strings.Contains(got, "99") {
		t.Errorf("Existing(99).String() = %q, want it to include the value", got)
	}
}

func TestNumbered(t *testing.T) {
	tests := []struct {
		name, input string
		n           int
		sanitize    bool
		want        string
	}{
		{"before extension", "ep12.mp3", 2, true, "ep12 (2).mp3"},
		{"counting on", "ep12.mp3", 10, true, "ep12 (10).mp3"},
		{"no extension", "ep12", 2, true, "ep12 (2)"},
		{"two extensions", "ep12.tar.gz", 2, true, "ep12.tar (2).gz"},
		{"leading dot", ".hidden", 2, true, ".hidden (2)"},
		{"unsanitized", "a:b.mp3", 2, false, "a:b (2).mp3"},
		{"sanitized", "a:b.mp3", 2, true, "a-b (2).mp3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Numbered(tt.input, tt.n, tt.sanitize); got != tt.want {
				t.Errorf("Numbered(%q, %d, %t) = %q, want %q", tt.input, tt.n, tt.sanitize, got, tt.want)
			}
		})
	}
}

func TestNumberedFitsLengthBudget(t *testing.T) {
	long := strings.Repeat("a", fsname.MaxComponentLen-len(".mp3")) + ".mp3"
	if _, ok := fsname.Component(long); !ok || len(long) != fsname.MaxComponentLen {
		t.Fatalf("test setup: %d-byte name, want a usable one of %d", len(long), fsname.MaxComponentLen)
	}
	seen := map[string]bool{long: true}
	for n := 2; n < 12; n++ {
		got := Numbered(long, n, true)
		if len(got) > fsname.MaxComponentLen {
			t.Errorf("Numbered(<%d bytes>, %d, true) is %d bytes, want at most %d", len(long), n, len(got), fsname.MaxComponentLen)
		}
		if _, ok := fsname.Component(got); !ok {
			t.Errorf("Numbered(<long>, %d, true) = %q, want a usable component", n, got)
		}
		if !strings.HasSuffix(got, ".mp3") {
			t.Errorf("Numbered(<long>, %d, true) = %q, want it to keep the extension", n, got)
		}
		if seen[got] {
			t.Errorf("Numbered(<long>, %d, true) = %q, which is already taken", n, got)
		}
		seen[got] = true
	}
}

package naming

import (
	"fmt"
	"strings"

	"ssch.cc/podtrawl/internal/fsname"
)

// Existing is the file path conflict handling policy.
type Existing int

const (
	// Increment appends a monotonically increasing integer to the filename.
	Increment Existing = iota
	// Skip leaves the existing file alone and doesn't download the episode.
	Skip
	// Replace overwrites the existing file with the episode.
	Replace
)

// existingNames maps a string representation to each value of Existing.
var existingNames = map[string]Existing{
	"increment": Increment,
	"skip":      Skip,
	"replace":   Replace,
}

// ParseExisting converts a string to an Existing value.
func ParseExisting(s string) (Existing, error) {
	if e, ok := existingNames[s]; ok {
		return e, nil
	}
	return 0, fmt.Errorf("unknown value %q, want increment|skip|replace", s)
}

func (e Existing) String() string {
	for name, v := range existingNames {
		if v == e {
			return name
		}
	}
	return fmt.Sprintf("Existing(%d)", int(e))
}

// Numbered appends n, parenthesized, to the stem of name.
//
// If sanitize is true, the stem is truncated if inserting n would  cause the return to exceed fsname.MaxComponentLen bytes.
func Numbered(name string, n int, sanitize bool) string {
	stem, ext := splitName(name)
	number := fmt.Sprintf(" (%d)", n)
	if sanitize {
		if budget := fsname.MaxComponentLen - len(number) - len(ext); len(stem) > budget {
			if budget < 0 {
				budget = 0
			}
			// Cutting bytes can leave a partial rune, which is not a character.
			stem = strings.ToValidUTF8(stem[:budget], "")
		}
	}
	numbered := stem + number + ext
	if sanitize {
		return fsname.ComponentOr(numbered, "_")
	}
	return numbered
}

// Package fsname sanitizes arbitrary text for use as a directory or file name.
//
// The rules applied are the union of the restrictions imposed by common modern  file systems
// (NTFS, ReFS, ext4, ZFS, APFS, BTRFS),
// so a library can freely be moved, synchronized or archived.
// They are applied on every platform, not just the one being built for.
//
// The package is deliberately minimal and unopinionated:
// Unicode normalization, case folding, and "unusual" character substitution do not belong here.
package fsname

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// replacement stands in for any rune that cannot appear in a portable name.
const replacement = '-'

// maxComponentLen is the maximum length, in bytes,
// of a path segment that is acceptable on all of the target file systems.
// NTFS, ReFS and exFAT use UTF-16 code points  rather than bytes,
// but a string of at most 255 UTF-8 bytes cannot exceed 255 UTF-16 code units,
// so the difference doesn't matter in this case.
const maxComponentLen = 255

// maxExtLen bounds what is treated as an extension when a name has to be shortened.
// Beyond this, a trailing ".something" is  most likely part of the title.
const maxExtLen = 16

// illegal lists the printable characters that are not portable.
//   - Path separators (/ \)
//   - Path list separators (: ;)
//   - Characters reserved on Windows (* ? " < > |)
const illegal = `"*/:;<>?\|`

// reserved holds the device names Windows refuses to use as a file name, in lower case.
// They are reserved with any extension, or when surrounded by whitespace.
var reserved = map[string]bool{
	"con": true, "prn": true, "aux": true, "nul": true,
	"com1": true, "com2": true, "com3": true, "com4": true, "com5": true, "com6": true, "com7": true, "com8": true, "com9": true, "com¹": true, "com²": true, "com³": true,
	"lpt1": true, "lpt2": true, "lpt3": true, "lpt4": true, "lpt5": true, "lpt6": true, "lpt7": true, "lpt8": true, "lpt9": true, "lpt¹": true, "lpt²": true, "lpt³": true,
}

// Component converts s into a single path component that is safe and portable.
// It reports whether the result is usable: a name that sanitizes away to
// nothing, such as "" or "..", yields ("", false).
//
// Component never returns a path separator, so its result cannot escape the
// directory it is joined to. Callers must pass one component at a time, as
// separators in s are replaced rather than honoured.
func Component(s string) (string, bool) {
	s = strings.Map(sanitizeRune, s)
	s = trimEdges(s)
	if s == "" {
		return "", false
	}
	budget := maxComponentLen
	// Leave room for the underscore escapeReserved will add,
	// so that escaping cannot push a truncated name back over the limit.
	if isReserved(s) {
		budget--
	}
	if len(s) > budget {
		// Truncation only ever cuts the tail, so it cannot turn an unreserved name into a reserved one:
		// it leaves at least maxComponentLen - maxExtLen - 1 bytes of stem,
		// which is far more than any device name.
		s = trimEdges(truncate(s, budget))
		if s == "" {
			return "", false
		}
	}
	return escapeReserved(s), true
}

// ComponentOr is Component with a caller-supplied fallback,
// used when s yields nothing usable.
// If the fallback is used, it is also sanitized.
// If the fallback is also unusable, it returns "_".
func ComponentOr(s, fallback string) string {
	if name, ok := Component(s); ok {
		return name
	}
	if name, ok := Component(fallback); ok {
		return name
	}
	return "_"
}

// sanitizeRune replaces the runes that no portable name may contain.
func sanitizeRune(r rune) rune {
	if unicode.IsControl(r) || strings.ContainsRune(illegal, r) {
		return replacement
	}
	return r
}

// trimEdges removes leading spaces, and trailing spaces and dots.
// These are silently stripped on Windows ,
// so including them would result in a name that could be created but not reopened.
// Trimming trailing dots also disposes of "." and "..".
// Leading dots are kept, as dotfiles are legitimate names.
func trimEdges(s string) string {
	return strings.TrimLeft(strings.TrimRight(s, " ."), " ")
}

// stemOf returns the portion of name before its first dot,
// which is what Windows matches against its reserved device names.
func stemOf(name string) string {
	if i := strings.IndexByte(name, '.'); i >= 0 {
		return name[:i]
	}
	return name
}

func isReserved(name string) bool {
	return reserved[strings.ToLower(strings.TrimSpace(stemOf(name)))]
}

// escapeReserved suffixes the stem of a reserved device name with an underscore,
// leaving any extension in place, so NUL.mp3 becomes NUL_.mp3.
func escapeReserved(name string) string {
	if !isReserved(name) {
		return name
	}
	stem := stemOf(name)
	return stem + "_" + name[len(stem):]
}

// truncate shortens name to at most max bytes,
// cutting the stem rather than the extension
// so that the file stays recognisable to whatever plays it.
func truncate(name string, max int) string {
	if len(name) <= max {
		return name
	}
	stem, ext, hasExt := strings.CutLast(name, ".")
	if !hasExt || len(ext) >= maxExtLen {
		// A pathological extension leaves no room for a stem.
		return truncateBytes(name, max)
	}
	return truncateBytes(stem, max-len(ext)-1) + "." + ext
}

// truncateBytes cuts s to at most max bytes,
// ensuring that the cut happens at a  rune boundary
// so that the result remains valid UTF-8.
func truncateBytes(s string, max int) string {
	if len(s) <= max {
		return s
	}
	for max > 0 && !utf8.RuneStart(s[max]) {
		max--
	}
	return s[:max]
}

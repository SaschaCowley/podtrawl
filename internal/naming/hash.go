package naming

import (
	"fmt"
	"hash/fnv"
)

// ShortHash returns a stable, name-safe digest of s.
// FNV-1a isn't cryptographic; it is only used here to keep generated names
// distinct from one another.
func ShortHash(s string) string {
	h := fnv.New64a()
	// Write never returns an error.
	h.Write([]byte(s))
	return fmt.Sprintf("%016x", h.Sum64())
}

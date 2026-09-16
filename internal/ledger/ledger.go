// Package cache records which podcast episodes have already been downloaded.
// The cache exists only to save work, so implementations should never fail a run on their own.
package ledger

type Cache interface {
	// Save persists the current state of the cache.
	Save() error
	// Downloaded reports whether the episode identified by guid from the feed at url has previously been downloaded.
	Downloaded(url, guid string) bool
	// SetDownloaded records whether the episode identified by guid in the feed at url has been downloaded.
	SetDownloaded(url, guid string, downloaded bool)
}

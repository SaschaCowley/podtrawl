// Package ledger records which podcast episodes have already been downloaded.
// The ledger exists only to save work, so implementations should never fail a run on their own.
package ledger

type Store interface {
	// Save persists the current state of the ledger.
	Save() error
	// Downloaded reports whether the episode identified by guid from the feed at url has previously been downloaded.
	Downloaded(url, guid string) bool
	// SetDownloaded records whether the episode identified by guid in the feed at url has been downloaded.
	SetDownloaded(url, guid string, downloaded bool)
}

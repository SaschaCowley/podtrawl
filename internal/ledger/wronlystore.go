package ledger

import "errors"

// WronlyStore is a write-only Store.
// Writes are persisted, but reads are always treated like an empty ledger.
type WronlyStore struct {
	fsStore FsStore
}

// NewWronly opens the ledger belonging to the download root at root,
// on the same terms as [New], and then declines to read from it.
func NewWronly(root string) (*WronlyStore, error) {
	s := &WronlyStore{}
	err := s.fsStore.load(root)
	if err != nil && !errors.Is(err, ErrCorrupt) {
		return nil, err
	}
	return s, err
}

func (s *WronlyStore) Save() error                  { return s.fsStore.Save() }
func (*WronlyStore) Downloaded(string, string) bool { return false }
func (s *WronlyStore) SetDownloaded(url, guid string, downloaded bool) {
	s.fsStore.SetDownloaded(url, guid, downloaded)
}

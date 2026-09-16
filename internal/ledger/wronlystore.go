package ledger

import "errors"

// WronlyStore is a write-only Store.
// Writes are persisted, but reads are always treated like an empty ledger.
type WronlyStore struct {
	fsStore FsStore
}

func NewWronly(dir *string) (*WronlyStore, error) {
	s := &WronlyStore{}
	err := s.fsStore.load(dir)
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

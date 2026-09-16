package ledger

// NopStore never persists anything, and always behaves like an empty ledger.
type NopStore struct{}

func (NopStore) Save() error                        { return nil }
func (NopStore) Downloaded(string, string) bool     { return false }
func (NopStore) SetDownloaded(string, string, bool) {}

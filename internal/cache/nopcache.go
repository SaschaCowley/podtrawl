package cache

// NopCache never persists anything, and always behaves like a cold cache.
type NopCache struct{}

func (NopCache) Save() error                        { return nil }
func (NopCache) Downloaded(string, string) bool     { return false }
func (NopCache) SetDownloaded(string, string, bool) {}

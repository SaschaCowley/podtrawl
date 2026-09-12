package cache

import "errors"

// WronlyCache is a write-only Cache.
// Writes are persisted, but reads are always treated like a cold cache.
type WronlyCache struct {
	fsCache FsCache
}

func NewWronly(dir *string) (*WronlyCache, error) {
	c := &WronlyCache{}
	err := c.fsCache.load(dir)
	if err != nil && !errors.Is(err, ErrCorrupt) {
		return nil, err
	}
	return c, err
}

func (c *WronlyCache) Save() error                  { return c.fsCache.Save() }
func (*WronlyCache) Downloaded(string, string) bool { return false }
func (c *WronlyCache) SetDownloaded(url, guid string, downloaded bool) {
	c.fsCache.SetDownloaded(url, guid, downloaded)
}

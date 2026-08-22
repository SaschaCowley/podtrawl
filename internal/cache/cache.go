package cache

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
)

const cacheFileName = "cache.json"

type cacheFile struct {
	Feeds map[string]*feed `json:"feeds,omitempty"`
}

type feed struct {
	Downloaded []string `json:"downloaded,omitempty"`
}

type Cache struct {
	cacheFile cacheFile
	fd        *os.File
}

func Open(cacheDir string) (*Cache, error) {
	var cache Cache
	f, err := os.OpenFile(filepath.Join(cacheDir, cacheFileName), os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	cache.fd = f
	if err := json.UnmarshalDecode(jsontext.NewDecoder(f), &cache.cacheFile); err != nil && err != io.EOF {
		f.Close()
		return nil, err
	}
	return &cache, nil
}

func (c *Cache) Close() error {
	return c.fd.Close()
}

func (c *Cache) Save() error {
	if c.fd == nil {
		return fs.ErrClosed
	}
	out, err := json.Marshal(c.cacheFile)
	if err != nil {
		return err
	}
	n, err := c.fd.WriteAt(out, 0)
	if err != nil {
		return err
	}
	if err := c.fd.Truncate((int64)(n)); err != nil {
		return err
	}
	return nil
}

func (c *Cache) Downloaded(url, guid string) bool {
	feed := c.cacheFile.Feeds[url]
	if feed == nil {
		return false
	}
	return slices.Contains(feed.Downloaded, guid)
}

func (c *Cache) SetDownloaded(url, guid string, downloaded bool) {
	f := c.cacheFile.Feeds[url]
	if f == nil {
		f = &feed{}
		c.cacheFile.Feeds[url] = f
	}
	guids := f.Downloaded
	i := slices.Index(guids, guid)
	if downloaded && i == -1 {
		guids = append(guids, guid)
	} else if !downloaded && i >= 0 {
		if len(guids) > 1 {
			guids[i] = guids[len(guids)-1]
		}
		guids = guids[:len(guids)-1]
	} else {
		return
	}
	c.cacheFile.Feeds[url].Downloaded = guids
}

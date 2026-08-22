package cache

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
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

func Open(dir *string) (*Cache, error) {
	var cacheDir string
	if dir != nil {
		cacheDir = *dir
	} else {
		if dir, err := os.UserCacheDir(); err != nil {
			return nil, err
		} else {
			cacheDir = filepath.Join(dir, "podtrawl")
		}
	}
	var fname = filepath.Join(cacheDir, cacheFileName)
	const fmode = os.O_RDWR | os.O_CREATE
	const fperm = 0644
	f, err := os.OpenFile(fname, fmode, fperm)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			if err := os.MkdirAll(cacheDir, fperm); err != nil {
				return nil, err
			}
			f, err = os.OpenFile(fname, fmode, fperm)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	cache := Cache{fd: f}
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
		if !downloaded {
			return
		}
		f = &feed{}
		if c.cacheFile.Feeds == nil {
			c.cacheFile.Feeds = make(map[string]*feed)
		}
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

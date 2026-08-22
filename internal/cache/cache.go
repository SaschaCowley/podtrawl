// Package cache records which podcast episodes have already been downloaded.
//
// The cache exists only to save work,
// so it never fails a run on its own:
// a cache that is missing, empty or unreadable is a cold cache,
// and the cost of losing one is downloading an episode a second time.
//
// Save replaces the cache file rather than rewriting it in place,
// so an interrupted run leaves the previous cache intact.
// Nothing coordinates concurrent runs;
// the last one to save wins.
package cache

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
)

const cacheFileName = "cache.json"

const dperm = 0755

// ErrCorrupt reports a cache file that could not be decoded.
var ErrCorrupt = errors.New("unreadable cache")

type cacheFile struct {
	Feeds map[string]*feed `json:"feeds,omitempty"`
}

type feed struct {
	Downloaded []string `json:"downloaded,omitempty"`
}

type Cache struct {
	cacheFile cacheFile
	path      string
}

// New reads the cache in dir,
// defaulting to a podtrawl directory under the user cache dir.
//
//   - A cache that isn't there yet is not an error;
//     it opens empty and comes into existence on the first Save.
//   - A corrupted cache causes New to return a cold Cache and ErrCorrupt;
//     the invalid cache file will be overwritten the next time Save is called.
func New(dir *string) (*Cache, error) {
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
	if err := os.MkdirAll(cacheDir, dperm); err != nil {
		return nil, err
	}
	cache := Cache{path: filepath.Join(cacheDir, cacheFileName)}
	data, err := os.ReadFile(cache.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &cache, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return &cache, nil
	}
	if err := json.Unmarshal(data, &cache.cacheFile); err != nil {
		cache.cacheFile = cacheFile{}
		return &cache, fmt.Errorf("%w %s: %w", ErrCorrupt, cache.path, err)
	}
	return &cache, nil
}

// Save writes the whole cache to a new file and renames it over the old one.
// Replacing the file rather than overwriting it in place means an interrupted run
// leaves the previous cache intact instead of a half-written one.
func (c *Cache) Save() (err error) {
	out, err := json.Marshal(c.cacheFile)
	if err != nil {
		return err
	}
	// The temporary file has to share a directory with the cache
	// for the rename to stay within one filesystem.
	tmp, err := os.CreateTemp(filepath.Dir(c.path), cacheFileName+".*")
	if err != nil {
		return err
	}
	defer func() {
		tmpPath := tmp.Name()
		if cerr := tmp.Close(); cerr != nil && err == nil {
			err = cerr
		}
		if err == nil {
			err = os.Rename(tmpPath, c.path)
		}
		if err != nil {
			os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(out); err != nil {
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

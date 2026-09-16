package ledger

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const cacheFileName = "cache.json"
const dperm = 0755

// ErrCorrupt reports a cache file that could not be decoded.
var ErrCorrupt = errors.New("unreadable cache")

type cacheFile struct {
	Feeds map[string]*feed `json:"feeds,omitempty"`
}

type feed struct {
	Downloaded strSet `json:"downloaded,omitempty"`
}

// FsCache is a filesystem-backed Cache.
type FsCache struct {
	cacheFile cacheFile
	path      string
	dirty     bool
}

func (c *FsCache) load(dir *string) error {
	var cacheDir string
	if dir != nil {
		cacheDir = *dir
	} else {
		if dir, err := os.UserCacheDir(); err != nil {
			return err
		} else {
			cacheDir = filepath.Join(dir, "podtrawl")
		}
	}
	if err := os.MkdirAll(cacheDir, dperm); err != nil {
		return err
	}
	c.path = filepath.Join(cacheDir, cacheFileName)
	data, err := os.ReadFile(c.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, &c.cacheFile); err != nil {
		c.cacheFile = cacheFile{}
		return fmt.Errorf("%w %s: %w", ErrCorrupt, c.path, err)
	}
	return nil
}

// New reads the cache in dir, defaulting to a podtrawl directory under the user cache dir.
//
//   - A cache that isn't there yet is not an error;
//     it opens empty and comes into existence on the first Save.
//   - A corrupted cache causes New to return a cold Cache and ErrCorrupt;
//     the invalid cache file will be overwritten the next time Save is called.
func New(dir *string) (*FsCache, error) {
	cache := &FsCache{}
	err := cache.load(dir)
	if err != nil && !errors.Is(err, ErrCorrupt) {
		return nil, err
	}
	return cache, err
}

// Save writes the whole cache to a new file and renames it over the old one.
// Replacing the file rather than overwriting it in place means an interrupted run
// leaves the previous cache intact instead of a half-written one.
func (c *FsCache) Save() (err error) {
	if !c.dirty {
		return nil
	}
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
		if err == nil {
			c.dirty = false
		} else {
			os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(out); err != nil {
		return err
	}
	return nil
}

func (c *FsCache) Downloaded(url, guid string) bool {
	feed := c.cacheFile.Feeds[url]
	if feed == nil {
		return false
	}
	_, downloaded := feed.Downloaded[guid]
	return downloaded
}

func (c *FsCache) SetDownloaded(url, guid string, downloaded bool) {
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
	if _, exists := f.Downloaded[guid]; exists == downloaded {
		return
	}
	c.dirty = true
	if !downloaded {
		delete(f.Downloaded, guid)
		return
	}
	if f.Downloaded == nil {
		f.Downloaded = make(strSet)
	}
	f.Downloaded[guid] = struct{}{}
}

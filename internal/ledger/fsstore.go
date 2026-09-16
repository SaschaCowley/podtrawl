package ledger

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const ledgerFileName = "cache.json"
const dperm = 0755

// ErrCorrupt reports a ledger that could not be decoded.
var ErrCorrupt = errors.New("unreadable ledger")

type ledgerFile struct {
	Feeds map[string]*feed `json:"feeds,omitempty"`
}

type feed struct {
	Downloaded strSet `json:"downloaded,omitempty"`
}

// FsStore is a filesystem-backed Store.
type FsStore struct {
	ledgerFile ledgerFile
	path       string
	dirty      bool
}

func (s *FsStore) load(dir *string) error {
	var ledgerDir string
	if dir != nil {
		ledgerDir = *dir
	} else {
		if dir, err := os.UserCacheDir(); err != nil {
			return err
		} else {
			ledgerDir = filepath.Join(dir, "podtrawl")
		}
	}
	if err := os.MkdirAll(ledgerDir, dperm); err != nil {
		return err
	}
	s.path = filepath.Join(ledgerDir, ledgerFileName)
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, &s.ledgerFile); err != nil {
		s.ledgerFile = ledgerFile{}
		return fmt.Errorf("%w %s: %w", ErrCorrupt, s.path, err)
	}
	return nil
}

// New reads the ledger in dir, defaulting to a podtrawl directory under the user cache dir.
//
//   - A ledger that isn't there yet is not an error;
//     it opens empty and comes into existence on the first Save.
//   - A corrupted ledger causes New to return an empty ledger and ErrCorrupt;
//     the invalid ledger file will be overwritten the next time Save is called.
func New(dir *string) (*FsStore, error) {
	ledger := &FsStore{}
	err := ledger.load(dir)
	if err != nil && !errors.Is(err, ErrCorrupt) {
		return nil, err
	}
	return ledger, err
}

// Save writes the whole ledger to a new file and renames it over the old one.
// Replacing the file rather than overwriting it in place means an interrupted run
// leaves the previous ledger intact instead of a half-written one.
func (s *FsStore) Save() (err error) {
	if !s.dirty {
		return nil
	}
	out, err := json.Marshal(s.ledgerFile)
	if err != nil {
		return err
	}
	// The temporary file has to share a directory with the ledger
	// for the rename to stay within one filesystem.
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ledgerFileName+".*")
	if err != nil {
		return err
	}
	defer func() {
		tmpPath := tmp.Name()
		if cerr := tmp.Close(); cerr != nil && err == nil {
			err = cerr
		}
		if err == nil {
			err = os.Rename(tmpPath, s.path)
		}
		if err == nil {
			s.dirty = false
		} else {
			os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(out); err != nil {
		return err
	}
	return nil
}

func (s *FsStore) Downloaded(url, guid string) bool {
	feed := s.ledgerFile.Feeds[url]
	if feed == nil {
		return false
	}
	_, downloaded := feed.Downloaded[guid]
	return downloaded
}

func (s *FsStore) SetDownloaded(url, guid string, downloaded bool) {
	f := s.ledgerFile.Feeds[url]
	if f == nil {
		if !downloaded {
			return
		}
		f = &feed{}
		if s.ledgerFile.Feeds == nil {
			s.ledgerFile.Feeds = make(map[string]*feed)
		}
		s.ledgerFile.Feeds[url] = f
	}
	if _, exists := f.Downloaded[guid]; exists == downloaded {
		return
	}
	s.dirty = true
	if !downloaded {
		delete(f.Downloaded, guid)
		return
	}
	if f.Downloaded == nil {
		f.Downloaded = make(strSet)
	}
	f.Downloaded[guid] = struct{}{}
}

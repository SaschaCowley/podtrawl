package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ssch.cc/podtrawl/internal/config"
	"ssch.cc/podtrawl/internal/fsname"
	"ssch.cc/podtrawl/internal/naming"
)

// resolvedFeed is one configured feed with its names settled.
type resolvedFeed struct {
	url      string
	sanitize bool
	existing naming.Existing
	dirname  *naming.Template[naming.FeedOnly]
	filename *naming.Template[naming.Metadata]
}

// resolve turns a config into the per-feed state a run needs.
// Parses and validates templates so that failure happens upfront,
// rather than in the middle of a run.
func resolve(c *config.Config) ([]resolvedFeed, error) {
	feeds := make([]resolvedFeed, 0, len(c.Feeds))
	for _, f := range c.Feeds {
		sanitize := setting(true, f.Sanitize, c.Defaults.Sanitize)
		dirname, err := naming.ParseFeed("dirname", setting(naming.DefaultDirname, f.Dirname, c.Defaults.Dirname), sanitize)
		if err != nil {
			return nil, fmt.Errorf("feed %q: dirname: %w", f.Url, err)
		}
		filename, err := naming.ParseEpisode("filename", setting(naming.DefaultFilename, f.Filename, c.Defaults.Filename), sanitize)
		if err != nil {
			return nil, fmt.Errorf("feed %q: filename: %w", f.Url, err)
		}
		existing, err := naming.ParseExisting(setting("increment", f.Existing, c.Defaults.Existing))
		if err != nil {
			return nil, fmt.Errorf("feed %q: existing: %w", f.Url, err)
		}
		feeds = append(feeds, resolvedFeed{
			url:      f.Url,
			sanitize: sanitize,
			existing: existing,
			dirname:  dirname,
			filename: filename,
		})
	}
	return feeds, nil
}

// setting is the first non-nil member of values, or builtin if none are.
func setting[T any](builtin T, values ...*T) T {
	for _, v := range values {
		if v != nil {
			return *v
		}
	}
	return builtin
}

// contained checks whether dest is somewhere under root, and returns an error if it is not.
func contained(root, dest string) error {
	rel, err := filepath.Rel(root, dest)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("destination %q is outside %q", dest, root)
	}
	return nil
}

// showDir is the directory this feed's episodes are written to,
// named by the feed's dirname template and kept under root.
func (f resolvedFeed) showDir(root string, m naming.FeedOnly) (string, error) {
	name, err := f.dirname.Render(m)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, name)
	if err := contained(root, dir); err != nil {
		return "", err
	}
	return dir, nil
}

// episodeDest is the file one episode is written to,
// named by the feed's filename template and kept under dir.
func (f resolvedFeed) episodeDest(dir string, m naming.Metadata) (string, error) {
	name, err := f.filename.Render(m)
	if err != nil {
		return "", err
	}
	dest := filepath.Join(dir, name)
	if err := contained(dir, dest); err != nil {
		return "", err
	}
	return dest, nil
}

// maxNumbered is the upper bound for filename incrementing.
var maxNumbered = 1000

// available is where an episode is to be written, given what is already there.
// It reports false when the episode is to be skipped.
func (f resolvedFeed) available(dest string) (string, bool, error) {
	if _, err := os.Stat(dest); errors.Is(err, os.ErrNotExist) {
		return dest, true, nil
	} else if err != nil {
		return "", false, statErr(dest, err)
	}
	switch f.existing {
	case naming.Skip:
		return dest, false, nil
	case naming.Replace:
		return dest, true, nil
	case naming.Increment:
		dir, name := filepath.Split(dest)
		for n := 1; n <= maxNumbered; n++ {
			numbered := filepath.Join(dir, naming.Numbered(name, n, f.sanitize))
			if _, err := os.Stat(numbered); errors.Is(err, os.ErrNotExist) {
				return numbered, true, nil
			} else if err != nil {
				return "", false, statErr(numbered, err)
			}
		}
		return "", false, fmt.Errorf("no free name for %q after %d tries", dest, maxNumbered)
	default:
		return dest, false, fmt.Errorf("unsupported value of existing: %v", f.existing)
	}
}

// statErr adds the length of a path's final segment to the error from statting it, if that length is over fsname.MaxComponentLen.
func statErr(path string, err error) error {
	if name := filepath.Base(path); len(name) > fsname.MaxComponentLen {
		return fmt.Errorf("%w: %q is %d bytes, over the %d-byte limit for one name", err, name, len(name), fsname.MaxComponentLen)
	}
	return err
}

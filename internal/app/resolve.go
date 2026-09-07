package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"ssch.cc/podtrawl/internal/config"
	"ssch.cc/podtrawl/internal/naming"
)

// resolvedFeed is one configured feed with its names settled.
type resolvedFeed struct {
	url      string
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
		feeds = append(feeds, resolvedFeed{url: f.Url, dirname: dirname, filename: filename})
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

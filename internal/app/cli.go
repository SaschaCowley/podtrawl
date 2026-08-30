package app

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/spf13/pflag"

	"ssch.cc/podtrawl/internal/cache"
	"ssch.cc/podtrawl/internal/clilog"
	"ssch.cc/podtrawl/internal/config"
	"ssch.cc/podtrawl/internal/feed"
	"ssch.cc/podtrawl/internal/fsname"
)

const dperm = 0755
const appDirName string = "podtrawl"

func CLI(args []string) int {
	f := flags{}
	if err := f.parse(args); err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			return 0
		}
		fmt.Fprint(os.Stderr, err)
		return 2
	}
	log := newLogger(f.verbosity, f.json)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, log); err != nil {
		log.Error(err.Error())
		return 1
	}
	return 0
}

func run(ctx context.Context, log *slog.Logger) error {
	conf, err := config.Get(nil, log)
	if err != nil {
		return err
	}
	cache, err := newCache(log)
	if err != nil {
		return err
	}
	root, err := downloadRoot()
	if err != nil {
		return err
	}
	log.Debug("download root", "dir", root)
	// A failing feed or episode shouldn't stop the run,
	// but it must still be visible in the exit code so a scheduled run can be seen to have failed.
	var failures int
	for _, f := range conf.Feeds {
		if err := ctx.Err(); err != nil {
			return err
		}
		// Every message about this feed carries its url,
		// so the messages below don't have to repeat it.
		fLog := log.With("feed", f.Url)
		rss, err := getFeed(ctx, f.Url)
		if err != nil {
			fLog.Error("fetch feed", "err", err)
			failures++
			continue
		}
		fLog.Debug("fetched feed", "title", rss.Channel.Title, "items", len(rss.Channel.Items))
		if len(rss.Channel.Items) == 0 {
			continue
		}
		// The title comes straight from the feed,
		// so it has to be made safe before it becomes a directory name.
		// Feeds without a usable title fall back to the feed url,
		// which is stable across runs.
		showDir := fsname.ComponentOr(rss.Channel.Title, "show-"+shortHash(f.Url))
		dirName := filepath.Join(root, showDir)
		fLog.Debug("show directory", "dir", dirName)
		for _, item := range rss.Channel.Items {
			if err := ctx.Err(); err != nil {
				return err
			}
			if len(item.Enclosures) == 0 {
				fLog.Debug("item has no enclosure", "item", itemTitle(item))
				continue
			}
			enclosure := item.Enclosures[0]
			key := episodeKey(item, enclosure)
			if cache.Downloaded(f.Url, key) {
				fLog.Debug("already downloaded", "episode", key)
				continue
			}
			dest := filepath.Join(dirName, episodeFileName(enclosure.Url))
			if err := DownloadEpisode(ctx, fLog, dest, enclosure); err != nil {
				fLog.Error("download episode", "url", enclosure.Url, "err", err)
				failures++
				continue
			}
			cache.SetDownloaded(f.Url, key, true)
			// An unsaved cache means we might have to do more work later,
			// but shouldn't mean we abandon all the remaining downloads.
			if err := cache.Save(); err != nil {
				fLog.Error("save cache", "url", enclosure.Url, "err", err)
				failures++
			}
		}
	}
	if failures > 0 {
		return fmt.Errorf("completed with %d error(s)", failures)
	}
	return nil
}

// newCache creates the download cache,
// reporting a cache that could not be read only as a warning.
// Starting from an empty cache costs a re-download;
// stopping here would cost the whole run.
func newCache(logger *slog.Logger) (*cache.Cache, error) {
	c, err := cache.New(nil)
	if err != nil {
		if !errors.Is(err, cache.ErrCorrupt) {
			return nil, err
		}
		logger.Warn("cache unreadable, starting cold", "err", err)
	}
	return c, nil
}

func getFeed(ctx context.Context, url string) (*feed.Rss, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch feed: %s", resp.Status)
	}
	rss, err := feed.Load(resp.Body)
	if err != nil {
		return nil, err
	}
	return rss, nil
}

// DownloadEpisode writes an episode to dest.
func DownloadEpisode(ctx context.Context, log *slog.Logger, dest string, enclosure feed.Enclosure) error {
	log.Debug("fetching episode",
		"url", enclosure.Url, "length", enclosure.Length, "type", enclosure.Type)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, enclosure.Url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download episode: %s", resp.Status)
	}
	if err := os.MkdirAll(filepath.Dir(dest), dperm); err != nil {
		return err
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	written, err := io.Copy(f, resp.Body)
	if err != nil {
		return err
	}
	// Length is required by the RSS spec but is frequently absent or wrong,
	// so only check it when the feed gives us something to check against.
	if enclosure.Length > 0 && written != enclosure.Length {
		log.Warn("unexpected episode size", "file", dest, "expected", enclosure.Length, "actual", written)
	}
	log.Info("downloaded episode", "file", dest)
	return nil
}

// itemTitle names an item for a message.
// Title is optional in RSS,
// and an item without one is exactly the sort of item worth mentioning.
func itemTitle(item feed.Item) string {
	if item.Title != nil && *item.Title != "" {
		return *item.Title
	}
	return "(untitled)"
}

// episodeKey identifies an episode within its feed.
// Guid is optional in RSS and some feeds send it empty,
// so fall back to the enclosure url,
// which is the next most stable identifier the item offers.
func episodeKey(item feed.Item, enclosure feed.Enclosure) string {
	if item.Guid != nil && item.Guid.Value != "" {
		return item.Guid.Value
	}
	return enclosure.Url
}

// episodeFileName derives a safe file name from an enclosure url.
// The query string is ignored,
// as signed urls would otherwise put their whole token in the name.
// A url that yields nothing usable,
// such as one ending in a slash,
// falls back to a hash of the url itself.
func episodeFileName(rawUrl string) string {
	var name string
	if u, err := url.Parse(rawUrl); err == nil {
		// Path is already percent-decoded.
		// Base reports an empty path as "."
		// and a path of nothing but slashes as "/",
		// neither of which is a name.
		if base := path.Base(u.Path); base != "." && base != "/" {
			name = base
		}
	}
	return fsname.ComponentOr(name, "episode-"+shortHash(rawUrl))
}

// shortHash returns a stable, name-safe digest of s.
// FNV-1a isn't cryptographic; it is only used here to keep fallback names
// distinct from one another.
func shortHash(s string) string {
	h := fnv.New64a()
	// Write never returns an error.
	h.Write([]byte(s))
	return fmt.Sprintf("%016x", h.Sum64())
}

// downloadRoot is the directory episodes are stored under.
// Uses Known Folders  on Windows
// and XDG user directories on Unix-like systems,
// falling back to a sensible, platform-dependent default if those fail.
// If even the fallback fails, the zero string and an error are returned.
func downloadRoot() (string, error) {
	if xdg.UserDirs.Download == "" {
		return "", errors.New("unable to determine downloads directory")
	}
	return filepath.Join(xdg.UserDirs.Download, appDirName), nil
}

// newLogger creates a new logger.
//
// If json is true, the logger will use slog.JSONHandler;
// otherwise, it will use clilog.Handler.
//
// Verbosity ranges from -1 to 2, inclusive, for
// slog.LevelDebug, slog.LevelWarn, slog.LevelError, slog.LevelError+1, respectively.
// Values outside of this range are clamped.
func newLogger(verbosity int, json bool) *slog.Logger {
	var level slog.Level
	switch min(max(verbosity, -1), 2) {
	case -1:
		level = slog.LevelDebug
	case 0:
		level = slog.LevelWarn
	case 1:
		level = slog.LevelError
	case 2:
		level = slog.LevelError + slog.Level(1)
	}
	var handler slog.Handler
	if json {
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	} else {
		handler = clilog.NewHandler(os.Stderr, level)
	}
	return slog.New(handler)
}

package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/spf13/pflag"

	"ssch.cc/podtrawl/internal/cache"
	"ssch.cc/podtrawl/internal/clilog"
	"ssch.cc/podtrawl/internal/config"
	"ssch.cc/podtrawl/internal/feed"
	"ssch.cc/podtrawl/internal/naming"
)

const dperm = 0755
const appDirName string = "podtrawl"

func CLI(args []string) int {
	f, err := parseArgs(args)
	if err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			return 0
		}
		fmt.Fprint(os.Stderr, err)
		return 2
	}
	log := newLogger(f.verbosity, f.json)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, log, f); err != nil {
		log.Error(err.Error())
		return 1
	}
	return 0
}

// run downloads every new episode of every configured feed.
func run(ctx context.Context, log *slog.Logger, opts *flags) error {
	conf, err := config.Get(opts.configPath, log)
	if err != nil {
		return err
	}
	if len(conf.Feeds) == 0 {
		log.Warn("config has no feeds")
		return nil
	}
	feeds, err := resolve(conf)
	if err != nil {
		return err
	}
	cache, err := newCache(log)
	if err != nil {
		return err
	}
	root, err := rootDir(conf)
	if err != nil {
		return err
	}
	log.Debug("download root", "dir", root)
	// A failing feed or episode shouldn't stop the run,
	// but it must still be visible in the exit code so a scheduled run can be seen to have failed.
	var failures int
	for _, f := range feeds {
		if err := ctx.Err(); err != nil {
			return err
		}
		// Every message about this feed carries its url,
		// so the messages below don't have to repeat it.
		fLog := log.With("feed", f.url)
		rss, err := getFeed(ctx, f.url)
		if err != nil {
			fLog.Error("fetch feed", "err", err)
			failures++
			continue
		}
		fLog.Debug("fetched feed", "title", rss.Channel.Title, "items", len(rss.Channel.Items))
		if len(rss.Channel.Items) == 0 {
			fLog.Debug("channel has no items")
			continue
		}
		// What the channel says is the same for every episode of this feed.
		feedMeta := naming.NewFeedMeta(rss.Channel, f.url)
		dirName, err := f.showDir(root, naming.FeedOnly{Feed: feedMeta})
		if err != nil {
			fLog.Error("show directory", "err", err)
			failures++
			continue
		}
		fLog.Debug("show directory", "dir", dirName)
		meta := naming.Metadata{Feed: feedMeta}
		for _, item := range rss.Channel.Items {
			if err := ctx.Err(); err != nil {
				return err
			}
			if len(item.Enclosures) == 0 {
				fLog.Debug("item has no enclosures", "item", itemTitle(item))
				continue
			}
			enclosure := item.Enclosures[0]
			key := episodeKey(item, enclosure)
			cached := cache.Downloaded(f.url, key)
			if cached && !opts.dryRun {
				fLog.Debug("already downloaded", "episode", key)
				continue
			}
			meta.Episode = naming.NewEpisodeMeta(item, enclosure)
			dest, err := f.episodeDest(dirName, meta)
			if err != nil {
				fLog.Error("episode name", "item", itemTitle(item), "err", err)
				failures++
				continue
			}
			if !cached {
				var free bool
				want := dest
				if dest, free, err = f.available(dest); err != nil {
					fLog.Error("episode destination", "item", itemTitle(item), "err", err)
					failures++
					continue
				}
				if !free {
					if opts.dryRun {
						fmt.Printf("%-8s  %s\n", "skip", dest)
					} else {
						fLog.Warn("file exists, skipping episode", "item", itemTitle(item), "file", dest)
					}
					continue
				}
				if dest != want {
					fLog.Warn("file exists, incrementing", "item", itemTitle(item), "taken", want, "file", dest)
				}
			}
			fLog.Debug("episode destination", "item", itemTitle(item), "dest", dest)
			if opts.dryRun {
				status := "download"
				if cached {
					status = "cached"
				}
				fmt.Printf("%-8s  %s\n", status, dest)
				continue
			}
			if err := DownloadEpisode(ctx, fLog, dest, enclosure); err != nil {
				fLog.Error("download episode", "url", enclosure.Url, "err", err)
				failures++
				continue
			}
			cache.SetDownloaded(f.url, key, true)
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

// rootDir gets the download root directory.
// If specified in config, `root` is used;
// otherwise, `podtrawl` in the user download directory is used.
// An error is returned if a root is provided and is not absolute.
// The root is simplified prior to being returned.
func rootDir(conf *config.Config) (string, error) {
	if conf.Root == nil {
		return downloadRoot()
	}
	if !filepath.IsAbs(*conf.Root) {
		return "", fmt.Errorf("config: root must be absolute, got %q", *conf.Root)
	}
	return filepath.Clean(*conf.Root), nil
}

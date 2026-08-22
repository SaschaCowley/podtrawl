package app

import (
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"

	"ssch.cc/podtrawl/internal/cache"
	"ssch.cc/podtrawl/internal/config"
	"ssch.cc/podtrawl/internal/feed"
	"ssch.cc/podtrawl/internal/fsname"
)

func CLI([]string) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func run(ctx context.Context) error {
	conf, err := config.Get(nil)
	if err != nil {
		return err
	}
	cache, err := cache.New(nil)
	if err != nil {
		return err
	}
	// A failing feed or episode shouldn't stop the run,
	// but it must still be visible in the exit code so a scheduled run can be seen to have failed.
	var failures int
	for _, feed := range conf.Feeds {
		if err := ctx.Err(); err != nil {
			return err
		}
		rss, err := getFeed(ctx, feed.Url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", feed.Url, err)
			failures++
			continue
		}
		for _, item := range rss.Channel.Items {
			if err := ctx.Err(); err != nil {
				return err
			}
			if len(item.Enclosures) != 0 {
				enclosure := item.Enclosures[0]
				key := episodeKey(item, enclosure)
				if cache.Downloaded(feed.Url, key) {
					continue
				}
				if err := DownloadEpisode(ctx, feed.Url, rss.Channel.Title, enclosure); err != nil {
					fmt.Fprintf(os.Stderr, "%s: %v\n", enclosure.Url, err)
					failures++
				} else {
					cache.SetDownloaded(feed.Url, key, true)
					if err := cache.Save(); err != nil {
						return err
					}
				}
			}
		}
	}
	if failures > 0 {
		return fmt.Errorf("completed with %d error(s)", failures)
	}
	return nil
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

func DownloadEpisode(ctx context.Context, feedUrl, showTitle string, enclosure feed.Enclosure) error {
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
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	// The title comes straight from the feed,
	// so it has to be made safe before it becomes a directory name.
	// Feeds without a usable title fall back to the feed url,
	// which is stable across runs.
	showDir := fsname.ComponentOr(showTitle, "show-"+shortHash(feedUrl))
	dirName := filepath.Join(filepath.Dir(exe), showDir)
	if err := os.MkdirAll(dirName, 0755); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(dirName, episodeFileName(enclosure.Url)))
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
		fmt.Fprintf(os.Stderr, "download episode: expected %d bytes, got %d\n", enclosure.Length, written)
	}
	return nil
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

package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"

	"ssch.cc/podtrawl/internal/config"
	"ssch.cc/podtrawl/internal/feed"
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
				if err := DownloadEpisode(ctx, rss.Channel.Title, item.Enclosures[0]); err != nil {
					fmt.Fprintf(os.Stderr, "%s: %v\n", item.Enclosures[0].Url, err)
					failures++
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

func DownloadEpisode(ctx context.Context, showTitle string, enclosure feed.Enclosure) error {
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
	dirName := filepath.Join(filepath.Dir(exe), showTitle)
	if err := os.MkdirAll(dirName, 0755); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(dirName, path.Base(enclosure.Url)))
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

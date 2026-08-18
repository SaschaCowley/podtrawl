package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"

	"ssch.cc/podtrawl/config"
	"ssch.cc/podtrawl/feed"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	run(ctx)
}

func run(ctx context.Context) error {
	conf, err := config.Get(nil)
	if err != nil {
		return err
	}
	for _, feed := range conf.Feeds {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			rss, err := getFeed(ctx, feed.Url)
			if err != nil {
				fmt.Println(err)
				continue
			}
			for _, item := range rss.Channel.Items {
				if len(item.Enclosures) != 0 {
					DownloadEpisode(ctx, rss.Channel.Title, item.Enclosures[0])
				}
			}
		}
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
	if written != enclosure.Length {
		fmt.Printf("Expected download to be %d bytes, got %d bytes instead.\n", enclosure.Length, written)
	}
	return nil
}

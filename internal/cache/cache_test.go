package cache

import (
	"slices"
	"testing"
)

func sampleCache() *Cache {
	return &Cache{
		cacheFile: cacheFile{
			Feeds: map[string]*feed{
				"url1": &feed{
					Downloaded: []string{"guid1"},
				},
				"url2": &feed{
					Downloaded: []string{"guid2"},
				},
			},
		},
	}
}

func TestDownloaded(t *testing.T) {
	tests := []struct {
		name string
		url  string
		guid string
		want bool
	}{
		{"downloaded episode", "url1", "guid1", true},
		{"undownloaded episode", "url1", "guid999", false},
		{"show with no downloads", "url999", "guid999", false},
		{"wrong url", "url2", "guid1", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sampleCache().Downloaded(tt.url, tt.guid); got != tt.want {
				t.Errorf("Cache.Downloaded(%q, %q) = %t, want %t", tt.url, tt.guid, got, tt.want)
			}
		})
	}
}

func TestSetDownloaded(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		guid       string
		downloaded bool
		prior      bool
	}{
		{"mark undownloaded episode as downloaded", "url1", "guid99", true, false},
		{"mark undownloaded episode as not downloaded", "url1", "guid99", false, false},
		{"mark downloaded episode as downloaded", "url1", "guid1", true, true},
		{"mark downloaded episode as not downloaded", "url1", "guid1", false, true},
		{"mark episode in new show as downloaded", "url99", "guid99", true, false},
		{"mark episode in new show as not downloaded", "url99", "guid99", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := sampleCache()
			feed := cache.cacheFile.Feeds[tt.url]
			var oldGuids, newGuids []string
			if feed != nil {
				oldGuids = slices.Clone(feed.Downloaded)
			} else {
				oldGuids = nil
			}
			cache.SetDownloaded(tt.url, tt.guid, tt.downloaded)
			if got := cache.Downloaded(tt.url, tt.guid); got != tt.downloaded {
				t.Errorf("Cache.SetDownloaded(%q, %q, %t(; Cache.downloaded(%[1]q, %q) = %[4]t, want %[3]t", tt.url, tt.guid, tt.downloaded, got)
			}
			if feed != nil {
				newGuids = feed.Downloaded
			} else {
				newGuids = nil
			}
			if tt.downloaded == tt.prior && !slices.Equal(oldGuids, newGuids) {
				t.Errorf("expected downloaded guids to be unchanged.\nold: %#v\nnew: %#v", oldGuids, newGuids)
			}
		})
	}
}

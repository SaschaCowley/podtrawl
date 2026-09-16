package ledger

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func sampleFsStore() *FsStore {
	return &FsStore{
		ledgerFile: ledgerFile{
			Feeds: map[string]*feed{
				"url1": &feed{
					Downloaded: strSet{"guid1": {}},
				},
				"url2": &feed{
					Downloaded: strSet{"guid2": {}},
				},
			},
		},
	}
}

func emptyLedger() *FsStore {
	return &FsStore{}
}

func TestDownloaded(t *testing.T) {
	tests := []struct {
		name   string
		ledger func() *FsStore
		url    string
		guid   string
		want   bool
	}{
		{"downloaded episode", sampleFsStore, "url1", "guid1", true},
		{"undownloaded episode", sampleFsStore, "url1", "guid999", false},
		{"show with no downloads", sampleFsStore, "url999", "guid999", false},
		{"wrong url", sampleFsStore, "url2", "guid1", false},
		{"empty ledger", emptyLedger, "url1", "guid1", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ledger := tt.ledger()
			if got := ledger.Downloaded(tt.url, tt.guid); got != tt.want {
				t.Errorf("FsStore.Downloaded(%q, %q) = %t, want %t", tt.url, tt.guid, got, tt.want)
			}
		})
	}
}

func TestSetDownloaded(t *testing.T) {
	tests := []struct {
		name       string
		ledger     func() *FsStore
		url        string
		guid       string
		downloaded bool
		prior      bool
	}{
		{"mark undownloaded episode as downloaded", sampleFsStore, "url1", "guid99", true, false},
		{"mark undownloaded episode as not downloaded", sampleFsStore, "url1", "guid99", false, false},
		{"mark downloaded episode as downloaded", sampleFsStore, "url1", "guid1", true, true},
		{"mark downloaded episode as not downloaded", sampleFsStore, "url1", "guid1", false, true},
		{"mark episode in new show as downloaded", sampleFsStore, "url99", "guid99", true, false},
		{"mark episode in new show as not downloaded", sampleFsStore, "url99", "guid99", false, false},
		{"mark episode as downloaded in empty ledger", emptyLedger, "url1", "guid1", true, false},
		{"mark episode as not downloaded in empty ledger", emptyLedger, "url1", "guid1", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ledger := tt.ledger()
			feed := ledger.ledgerFile.Feeds[tt.url]
			var oldGuids, newGuids strSet
			if feed != nil {
				oldGuids = maps.Clone(feed.Downloaded)
			} else {
				oldGuids = nil
			}
			ledger.SetDownloaded(tt.url, tt.guid, tt.downloaded)
			if got := ledger.Downloaded(tt.url, tt.guid); got != tt.downloaded {
				t.Errorf("FsStore.SetDownloaded(%q, %q, %t); FsStore.Downloaded(%[1]q, %q) = %[4]t, want %[3]t", tt.url, tt.guid, tt.downloaded, got)
			}
			if feed != nil {
				newGuids = feed.Downloaded
			} else {
				newGuids = nil
			}
			if tt.downloaded == tt.prior && !maps.Equal(oldGuids, newGuids) {
				t.Errorf("expected downloaded guids to be unchanged.\nold: %#v\nnew: %#v", oldGuids, newGuids)
			}
			if feed == nil && !tt.downloaded {
				if _, ok := ledger.ledgerFile.Feeds[tt.url]; ok {
					t.Errorf("FsStore.SetDownloaded(%q, %q, false) created an entry for an unknown show", tt.url, tt.guid)
				}
			}
		})
	}
}

func TestSaveRoundTrip(t *testing.T) {
	// A directory that isn't there yet also exercises the MkdirAll in New.
	dir := filepath.Join(t.TempDir(), "podtrawl")
	ledger, err := New(&dir)
	if err != nil {
		t.Fatalf("New(%q) = %v", dir, err)
	}
	if ledger.Downloaded("url1", "guid1") {
		t.Errorf(`FsStore.Downloaded("url1", "guid1") = true in a ledger that does not exist yet, want false`)
	}
	ledger.SetDownloaded("url1", "guid1", true)
	if err := ledger.Save(); err != nil {
		t.Fatalf("FsStore.Save() = %v", err)
	}
	reopened, err := New(&dir)
	if err != nil {
		t.Fatalf("New(%q) after Save = %v", dir, err)
	}
	if !reopened.Downloaded("url1", "guid1") {
		t.Errorf(`FsStore.Downloaded("url1", "guid1") = false after reopening, want true`)
	}
	// Save renames a temporary file into place; none of them should survive it.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	if !slices.Equal(names, []string{ledgerFileName}) {
		t.Errorf("ledger dir holds %q after Save, want only %q", names, ledgerFileName)
	}
}

func TestNewCorruptLedger(t *testing.T) {
	dir := t.TempDir()
	// A save that was interrupted before rename-in-place existed.
	partial := []byte(`{"feeds":{"url1":{"downloaded":["guid1"`)
	if err := os.WriteFile(filepath.Join(dir, ledgerFileName), partial, 0600); err != nil {
		t.Fatal(err)
	}
	ledger, err := New(&dir)
	if !errors.Is(err, ErrCorrupt) {
		t.Errorf("New(%q) on an undecodable ledger = %v, want %v", dir, err, ErrCorrupt)
	}
	if ledger == nil {
		t.Fatalf("New(%q) on an undecodable ledger = nil, want a usable empty ledger", dir)
	}
	if ledger.Downloaded("url1", "guid1") {
		t.Errorf(`FsStore.Downloaded("url1", "guid1") = true, want false: nothing decoded is to be trusted`)
	}
	// The bad file must not outlive the run that found it.
	ledger.SetDownloaded("url1", "guid1", true)
	if err := ledger.Save(); err != nil {
		t.Fatalf("FsStore.Save() = %v", err)
	}
	reopened, err := New(&dir)
	if err != nil {
		t.Fatalf("New(%q) after saving over an undecodable ledger = %v", dir, err)
	}
	if !reopened.Downloaded("url1", "guid1") {
		t.Errorf(`FsStore.Downloaded("url1", "guid1") = false after reopening, want true`)
	}
}

// noLedgerFile asserts that a ledger file does not exist.
func noLedgerFile(t *testing.T, dir, after string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, ledgerFileName)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("%s wrote %s, want no write (stat: %v)", after, ledgerFileName, err)
	}
}

func TestSaveSkipsUnchanged(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "podtrawl")
	ledger, err := New(&dir)
	if err != nil {
		t.Fatalf("New(%q) = %v", dir, err)
	}
	if err := ledger.Save(); err != nil {
		t.Fatalf("FsStore.Save() on an unchanged ledger = %v", err)
	}
	noLedgerFile(t, dir, "FsStore.Save() on a ledger that has never changed")

	ledger.SetDownloaded("url1", "guid1", true)
	if err := ledger.Save(); err != nil {
		t.Fatalf("FsStore.Save() = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ledgerFileName)); err != nil {
		t.Fatalf("stat %s after a save that changed something = %v", ledgerFileName, err)
	}

	// Delete the ledger file so that, if saving writes, it is easily observable.
	if err := os.Remove(filepath.Join(dir, ledgerFileName)); err != nil {
		t.Fatal(err)
	}
	if err := ledger.Save(); err != nil {
		t.Fatalf("FsStore.Save() straight after a successful save = %v", err)
	}
	noLedgerFile(t, dir, "a second FsStore.Save() with nothing changed in between")

	ledger.SetDownloaded("url1", "guid1", true)
	if err := ledger.Save(); err != nil {
		t.Fatalf("FsStore.Save() after a redundant SetDownloaded = %v", err)
	}
	noLedgerFile(t, dir, "FsStore.Save() after a redundant SetDownloaded")
}

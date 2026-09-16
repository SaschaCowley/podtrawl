package ledger

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func sampleWronlyStore() *WronlyStore {
	return &WronlyStore{fsStore: *sampleFsStore()}
}

func TestWronlyDownloaded(t *testing.T) {
	c := sampleWronlyStore()
	if !c.fsStore.Downloaded("url1", "guid1") {
		t.Fatal(`wrapped store does not hold "url1"/"guid1"`)
	}
	if c.Downloaded("url1", "guid1") {
		t.Errorf(`WronlyStore.Downloaded("url1", "guid1") = true for an episode the wrapped store holds, want false`)
	}
}

func TestWronlySetDownloaded(t *testing.T) {
	c := sampleWronlyStore()
	c.SetDownloaded("url3", "guid3", true)
	if !c.fsStore.Downloaded("url3", "guid3") {
		t.Errorf(`WronlyStore.SetDownloaded("url3", "guid3", true) did not reach the wrapped store`)
	}
}

// TestWronlySaveKeepsExistingEntries checks that a redownload run adds to the
// ledger rather than replacing it: refusing to read is not the same as starting empty.
func TestWronlySaveKeepsExistingEntries(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "podtrawl")
	warm, err := New(&dir)
	if err != nil {
		t.Fatalf("New(%q) = %v", dir, err)
	}
	warm.SetDownloaded("url1", "guid1", true)
	if err := warm.Save(); err != nil {
		t.Fatalf("WronlyStore.Save() = %v", err)
	}

	c, err := NewWronly(&dir)
	if err != nil {
		t.Fatalf("NewWronly(%q) = %v", dir, err)
	}
	if c.Downloaded("url1", "guid1") {
		t.Errorf(`WronlyStore.Downloaded("url1", "guid1") = true for a saved episode, want false`)
	}
	c.SetDownloaded("url2", "guid2", true)
	if err := c.Save(); err != nil {
		t.Fatalf("WronlyStore.Save() = %v", err)
	}

	reopened, err := New(&dir)
	if err != nil {
		t.Fatalf("New(%q) after WronlyStore.Save = %v", dir, err)
	}
	for _, tt := range []struct{ url, guid string }{{"url1", "guid1"}, {"url2", "guid2"}} {
		if !reopened.Downloaded(tt.url, tt.guid) {
			t.Errorf("WronlyStore.Downloaded(%q, %q) = false after reopening, want true", tt.url, tt.guid)
		}
	}
}

func TestNewWronlyCorruptLedger(t *testing.T) {
	dir := t.TempDir()
	// A save that was interrupted before rename-in-place existed.
	partial := []byte(`{"feeds":{"url1":{"downloaded":["guid1"`)
	if err := os.WriteFile(filepath.Join(dir, ledgerFileName), partial, 0600); err != nil {
		t.Fatal(err)
	}
	c, err := NewWronly(&dir)
	if !errors.Is(err, ErrCorrupt) {
		t.Errorf("NewWronly(%q) on an undecodable ledger = %v, want %v", dir, err, ErrCorrupt)
	}
	if c == nil {
		t.Fatalf("NewWronly(%q) on an undecodable ledger = nil, want a usable empty ledger", dir)
	}
	c.SetDownloaded("url1", "guid1", true)
	if err := c.Save(); err != nil {
		t.Fatalf("WronlyStore.Save() = %v", err)
	}
	reopened, err := New(&dir)
	if err != nil {
		t.Fatalf("New(%q) after saving over an undecodable ledger = %v", dir, err)
	}
	if !reopened.Downloaded("url1", "guid1") {
		t.Errorf(`WronlyStore.Downloaded("url1", "guid1") = false after reopening, want true`)
	}
}

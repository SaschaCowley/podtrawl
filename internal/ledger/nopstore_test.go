package ledger

import "testing"

func TestNopCacheForgetsEverything(t *testing.T) {
	var c NopCache
	if c.Downloaded("url1", "guid1") {
		t.Errorf(`NopCache.Downloaded("url1", "guid1") = true, want false`)
	}
	c.SetDownloaded("url1", "guid1", true)
	if c.Downloaded("url1", "guid1") {
		t.Errorf(`NopCache.Downloaded("url1", "guid1") = true after SetDownloaded, want false`)
	}
	if err := c.Save(); err != nil {
		t.Errorf("NopCache.Save() = %v, want nil", err)
	}
}

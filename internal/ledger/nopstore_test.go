package ledger

import "testing"

func TestNopStoreForgetsEverything(t *testing.T) {
	var s NopStore
	if s.Downloaded("url1", "guid1") {
		t.Errorf(`NopStore.Downloaded("url1", "guid1") = true, want false`)
	}
	s.SetDownloaded("url1", "guid1", true)
	if s.Downloaded("url1", "guid1") {
		t.Errorf(`NopStore.Downloaded("url1", "guid1") = true after SetDownloaded, want false`)
	}
	if err := s.Save(); err != nil {
		t.Errorf("NopStore.Save() = %v, want nil", err)
	}
}

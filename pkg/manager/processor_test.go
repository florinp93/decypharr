package manager

import (
	"testing"

	debridTypes "github.com/sirrobot01/decypharr/pkg/debrid/types"
	"github.com/sirrobot01/decypharr/pkg/storage"
)

// newUncachedQueueEntry mirrors addNewTorrent for a provider that returns no
// files while the torrent is still downloading (AllDebrid, Premiumize).
func newUncachedQueueEntry(t *testing.T) *storage.Entry {
	t.Helper()

	entry := &storage.Entry{
		InfoHash:  "1c0d835fb91a7169e67a758aec140f2e1c532388",
		Name:      "Inside.Llewyn.Davis.2013.1080p.BluRay.DD5.1.x264-EbP",
		Status:    debridTypes.TorrentStatusQueued,
		Providers: make(map[string]*storage.ProviderEntry),
		Files:     make(map[string]*storage.File),
	}
	applyDebridTorrentToEntry(entry, &debridTypes.Torrent{
		Id:     "805618259",
		Name:   entry.Name,
		Debrid: "alldebrid",
		Status: debridTypes.TorrentStatusDownloading,
		Files:  make(map[string]debridTypes.File),
	})
	if len(entry.Files) != 0 {
		t.Fatalf("precondition: expected no files on submit, got %d", len(entry.Files))
	}
	return entry
}

func TestSyncEntryFromDebridStatusPopulatesFilesOnCompletion(t *testing.T) {
	entry := newUncachedQueueEntry(t)

	const fileName = "Inside.Llewyn.Davis.2013.1080p.BluRay.DD5.1.x264-EbP.mkv"
	syncEntryFromDebridStatus(entry, &debridTypes.Torrent{
		Id:       "805618259",
		Name:     entry.Name,
		Debrid:   "alldebrid",
		Status:   debridTypes.TorrentStatusDownloaded,
		Progress: 100,
		Bytes:    1234,
		Files: map[string]debridTypes.File{
			fileName: {Id: "0", Name: fileName, Size: 1234, Link: "https://alldebrid.com/f/abc", Path: fileName},
		},
	})

	if _, ok := entry.Files[fileName]; !ok {
		t.Fatalf("expected entry files to contain %q, got %d files", fileName, len(entry.Files))
	}
	placement := entry.GetActiveProvider()
	if placement == nil {
		t.Fatal("expected active provider to remain set")
	}
	if pf, ok := placement.Files[fileName]; !ok || pf.Link == "" {
		t.Fatalf("expected provider file with link for %q", fileName)
	}
	if placement.Progress != 1.0 {
		t.Fatalf("expected placement progress 1.0, got %v", placement.Progress)
	}
}

func TestSyncEntryFromDebridStatusKeepsProviderWhenDebridNameMissing(t *testing.T) {
	entry := newUncachedQueueEntry(t)

	syncEntryFromDebridStatus(entry, &debridTypes.Torrent{
		Id:       "805618259",
		Name:     entry.Name,
		Status:   debridTypes.TorrentStatusDownloaded,
		Progress: 100,
		Files: map[string]debridTypes.File{
			"a.mkv": {Name: "a.mkv", Size: 1},
		},
	})

	if entry.ActiveProvider != "alldebrid" {
		t.Fatalf("expected active provider alldebrid, got %q", entry.ActiveProvider)
	}
	if len(entry.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(entry.Files))
	}
}

func TestSyncEntryFromDebridStatusOnlyUpdatesProgressWhileDownloading(t *testing.T) {
	entry := newUncachedQueueEntry(t)

	syncEntryFromDebridStatus(entry, &debridTypes.Torrent{
		Id:       "805618259",
		Name:     entry.Name,
		Debrid:   "alldebrid",
		Status:   debridTypes.TorrentStatusDownloading,
		Progress: 50,
		Speed:    100,
	})

	if entry.Progress != 0.5 {
		t.Fatalf("expected progress 0.5, got %v", entry.Progress)
	}
	if entry.Speed != 100 {
		t.Fatalf("expected speed 100, got %v", entry.Speed)
	}
	if placement := entry.GetActiveProvider(); placement == nil || placement.Progress != 0.5 {
		t.Fatal("expected placement progress 0.5")
	}
}

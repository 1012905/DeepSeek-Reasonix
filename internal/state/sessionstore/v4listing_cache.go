package sessionstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"reasonix/internal/state/sessionv4"
)

// v4ListingName remembers what replaying a 1.x conversation answered. 1.x's own
// cache is missing or stale for many of them, and the store beside the folder
// is read-only to us, so the answer is kept in the folder we own.
const v4ListingName = ".v4-listing.json"

const v4ListingFormat = 1

type v4ListingEntry struct {
	LogSize int64             `json:"s"`
	LogMod  int64             `json:"m"`
	Listing sessionv4.Listing `json:"l"`
}

type v4ListingCache struct {
	Format  int                       `json:"format"`
	Entries map[string]v4ListingEntry `json:"entries"`
	dirty   bool
}

var v4ListingMu sync.Mutex

func loadV4Listings(dir string) *v4ListingCache {
	c := &v4ListingCache{Format: v4ListingFormat, Entries: map[string]v4ListingEntry{}}
	raw, err := os.ReadFile(filepath.Join(dir, v4ListingName))
	if err != nil {
		return c
	}
	var read v4ListingCache
	if json.Unmarshal(raw, &read) != nil || read.Format != v4ListingFormat || read.Entries == nil {
		return c
	}
	c.Entries = read.Entries
	return c
}

func (c *v4ListingCache) get(s sessionv4.Session) (sessionv4.Listing, bool) {
	e, ok := c.Entries[s.Manifest.SessionID]
	if !ok || e.LogSize != s.LogSize || e.LogMod != s.LogModTime.UnixNano() {
		return sessionv4.Listing{}, false
	}
	return e.Listing, true
}

func (c *v4ListingCache) put(s sessionv4.Session, l sessionv4.Listing) {
	c.Entries[s.Manifest.SessionID] = v4ListingEntry{LogSize: s.LogSize, LogMod: s.LogModTime.UnixNano(), Listing: l}
	c.dirty = true
}

// save is best effort: a folder that cannot be written replays next time.
func (c *v4ListingCache) save(dir string) {
	if !c.dirty {
		return
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return
	}
	v4ListingMu.Lock()
	defer v4ListingMu.Unlock()
	tmp, err := os.CreateTemp(dir, v4ListingName+".*")
	if err != nil {
		return
	}
	_, werr := tmp.Write(raw)
	if cerr := tmp.Close(); werr != nil || cerr != nil || os.Rename(tmp.Name(), filepath.Join(dir, v4ListingName)) != nil {
		os.Remove(tmp.Name())
	}
}

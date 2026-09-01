package gw

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CacheTTL is how long name/description records stay valid. Upstream tool
// renames self-heal after this window; schemas are never cached.
const CacheTTL = 10 * time.Minute

// ToolCache holds name/target/description rows so search does not pay the
// gateway's 8-target fanout on every run.
type ToolCache struct {
	path string
	url  string
}

func NewToolCache(url string) (*ToolCache, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	sum := fmt.Sprintf("%x", sha1.Sum([]byte(url)))[:12]
	return &ToolCache{path: filepath.Join(dir, "agwctl", "tools-"+sum+".json"), url: url}, nil
}

type cachedTools struct {
	FetchedAt time.Time `json:"fetchedAt"`
	URL       string    `json:"url"`
	Rows      []ToolRow `json:"rows"`
}

func (c *ToolCache) Load(now time.Time) ([]ToolRow, bool, error) {
	raw, err := os.ReadFile(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	var ct cachedTools
	if err := json.Unmarshal(raw, &ct); err != nil {
		return nil, false, nil // corrupt cache behaves like a miss
	}
	if ct.URL != c.url {
		return nil, false, nil
	}
	if now.Sub(ct.FetchedAt) > CacheTTL {
		return nil, false, nil
	}
	return ct.Rows, true, nil
}

func (c *ToolCache) Store(rows []ToolRow, now time.Time) error {
	if err := os.MkdirAll(filepath.Dir(c.path), 0o700); err != nil {
		return err
	}
	raw, err := json.Marshal(cachedTools{FetchedAt: now.UTC(), URL: c.url, Rows: rows})
	if err != nil {
		return err
	}
	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, c.path)
}

package release

import "sync"

type Artifact struct {
	Manifest   ReleaseManifest        `json:"manifest"`
	Credential VerificationCredential `json:"credential"`
	Digest     string                 `json:"digest"`
}
type Catalog struct {
	mu    sync.RWMutex
	items map[string]Artifact
}

func NewCatalog() *Catalog        { return &Catalog{items: map[string]Artifact{}} }
func (c *Catalog) Put(a Artifact) { c.mu.Lock(); defer c.mu.Unlock(); c.items[a.Manifest.CaseID] = a }
func (c *Catalog) Get(caseID string) (Artifact, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	a, ok := c.items[caseID]
	return a, ok
}
func (c *Catalog) Count() int { c.mu.RLock(); defer c.mu.RUnlock(); return len(c.items) }
func (c *Catalog) Clear()     { c.mu.Lock(); defer c.mu.Unlock(); c.items = map[string]Artifact{} }

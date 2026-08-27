package release

type Artifact struct {
	Manifest   ReleaseManifest        `json:"manifest"`
	Credential VerificationCredential `json:"credential"`
	Digest     string                 `json:"digest"`
}
type Catalog struct {
	items map[string]Artifact
}

func NewCatalog() *Catalog        { return &Catalog{items: map[string]Artifact{}} }
func (c *Catalog) Put(a Artifact) { c.items[a.Manifest.CaseID] = a }
func (c *Catalog) Get(caseID string) (Artifact, bool) {
	a, ok := c.items[caseID]
	return a, ok
}
func (c *Catalog) Count() int { return len(c.items) }
func (c *Catalog) Clear()     { c.items = map[string]Artifact{} }

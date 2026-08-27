package release

import (
	"archive-release/internal/domain"
	"sync"
	"time"
)

type Service struct {
	Key               string
	verificationMu    sync.RWMutex
	verificationCache map[string]VerificationResult
}

func New(key string) *Service {
	return &Service{Key: key, verificationCache: make(map[string]VerificationResult)}
}
func (s *Service) Freeze(c *domain.ArchiveCase, actor string, atUnix int64) (ReleaseManifest, VerificationCredential, error) {
	r := c.CurrentRevision()
	at := timeFromUnix(atUnix)
	m, d, e := CanonicalManifest(c, r, "man-"+domain.Digest(struct{ C, R string }{c.CaseID, r.RevisionID})[:12], at)
	if e != nil {
		return m, VerificationCredential{}, e
	}
	cred := IssueCredential(s.Key, m, d, actor, at)
	m.CredentialID = cred.CredentialID
	return m, cred, nil
}
func timeFromUnix(v int64) time.Time { return time.Unix(v, 0).UTC() }

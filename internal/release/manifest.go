package release

import (
	"archive-release/internal/domain"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"time"
)

type ReleaseManifest struct {
	ManifestID    string    `json:"manifestID"`
	CaseID        string    `json:"caseID"`
	RevisionID    string    `json:"revisionID"`
	Title         string    `json:"title"`
	ArchiveUnit   string    `json:"archiveUnit"`
	Creator       string    `json:"creator"`
	ContentDigest string    `json:"contentDigest"`
	FrozenAt      time.Time `json:"frozenAt"`
	CredentialID  string    `json:"credentialID"`
}
type VerificationCredential struct {
	CredentialID string    `json:"credentialID"`
	ManifestID   string    `json:"manifestID"`
	IssuedTo     string    `json:"issuedTo"`
	IssuedAt     time.Time `json:"issuedAt"`
	Algorithm    string    `json:"algorithm"`
	Signature    string    `json:"signature"`
	Digest       string    `json:"digest"`
}

func CanonicalManifest(c *domain.ArchiveCase, r *domain.ScanRevision, id string, at time.Time) (ReleaseManifest, string, error) {
	if c == nil || r == nil {
		return ReleaseManifest{}, "", errors.New("缺少冻结修订")
	}
	m := ReleaseManifest{ManifestID: id, CaseID: c.CaseID, RevisionID: r.RevisionID, Title: c.Title, ArchiveUnit: c.ArchiveUnit, Creator: c.Creator, ContentDigest: r.ContentDigest, FrozenAt: at}
	// FrozenAt and CredentialID are publication metadata; the digest covers the
	// canonical identity and content so a preview remains valid at freeze time.
	digest, err := canonicalManifestDigest(m)
	if err != nil {
		return m, "", err
	}
	return m, digest, nil
}
func canonicalManifestDigest(m ReleaseManifest) (string, error) {
	b, err := json.Marshal(struct {
		ManifestID    string `json:"manifestID"`
		CaseID        string `json:"caseID"`
		RevisionID    string `json:"revisionID"`
		Title         string `json:"title"`
		ArchiveUnit   string `json:"archiveUnit"`
		Creator       string `json:"creator"`
		ContentDigest string `json:"contentDigest"`
	}{m.ManifestID, m.CaseID, m.RevisionID, m.Title, m.ArchiveUnit, m.Creator, m.ContentDigest})
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
func IssueCredential(key string, m ReleaseManifest, digest, actor string, at time.Time) VerificationCredential {
	id := "cred-" + digest[:12]
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(digest))
	return VerificationCredential{CredentialID: id, ManifestID: m.ManifestID, IssuedTo: actor, IssuedAt: at, Algorithm: "HMAC-SHA256", Signature: hex.EncodeToString(mac.Sum(nil)), Digest: digest}
}
func VerifyCredential(key string, m ReleaseManifest, c VerificationCredential, digest string) bool {
	if len(digest) < 12 {
		return false
	}
	if c.CredentialID == "" || c.CredentialID != "cred-"+digest[:12] || c.ManifestID != m.ManifestID || c.Algorithm != "HMAC-SHA256" || c.Digest != digest {
		return false
	}
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(digest))
	want := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(want), []byte(c.Signature))
}
func StablePageDigest(r *domain.ScanRevision) string {
	pages := append([]domain.PageMeasurement(nil), r.Pages...)
	sort.Slice(pages, func(i, j int) bool { return pages[i].PageNumber < pages[j].PageNumber })
	return domain.Digest(struct {
		Metadata []string                 `json:"metadata"`
		Pages    []domain.PageMeasurement `json:"pages"`
	}{domain.CanonicalMap(r.Metadata), pages})
}

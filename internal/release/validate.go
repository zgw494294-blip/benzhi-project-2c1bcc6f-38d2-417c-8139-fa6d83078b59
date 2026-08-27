package release

import (
	"archive-release/internal/domain"
	"errors"
	"strings"
)

func ValidateManifest(m ReleaseManifest) error {
	if m.ManifestID == "" || m.CaseID == "" || m.RevisionID == "" {
		return errors.New("清单标识不完整")
	}
	if strings.TrimSpace(m.ContentDigest) == "" {
		return errors.New("清单摘要为空")
	}
	return nil
}
func ValidateArtifact(a Artifact) error {
	if err := ValidateManifest(a.Manifest); err != nil {
		return err
	}
	if a.Credential.ManifestID != a.Manifest.ManifestID {
		return errors.New("凭据与清单不匹配")
	}
	return nil
}
func BuildArtifact(c *domain.ArchiveCase, m ReleaseManifest, cred VerificationCredential, digest string) (Artifact, error) {
	a := Artifact{Manifest: m, Credential: cred, Digest: digest}
	return a, ValidateArtifact(a)
}

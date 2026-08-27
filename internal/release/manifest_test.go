package release

import (
	"archive-release/internal/domain"
	"testing"
	"time"
)

func TestCredentialVerification(t *testing.T) {
	c := &domain.ArchiveCase{CaseID: "c", Title: "t", ArchiveUnit: "u", Creator: "x"}
	r := &domain.ScanRevision{RevisionID: "r", ContentDigest: "d"}
	m, d, e := CanonicalManifest(c, r, "m", time.Unix(1, 0))
	if e != nil {
		t.Fatal(e)
	}
	cred := IssueCredential("secret", m, d, "reviewer", time.Unix(1, 0))
	if !VerifyCredential("secret", m, cred, d) {
		t.Fatal("凭据验证失败")
	}
	if VerifyCredential("bad", m, cred, d) {
		t.Fatal("错误密钥不应通过")
	}
}

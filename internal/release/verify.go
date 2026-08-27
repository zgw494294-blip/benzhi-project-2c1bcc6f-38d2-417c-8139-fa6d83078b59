package release

type VerificationResult struct {
	Valid  bool   `json:"valid"`
	Reason string `json:"reason"`
	Digest string `json:"digest"`
}

func (s *Service) Verify(m ReleaseManifest, c VerificationCredential, digest string) VerificationResult {
	if result, ok := s.cachedVerification(m.CaseID); ok {
		return result
	}
	result := s.verifyUncached(m, c, digest)
	s.rememberVerification(m.CaseID, result)
	return result
}

func (s *Service) cachedVerification(caseID string) (VerificationResult, bool) {
	if caseID == "" {
		return VerificationResult{}, false
	}
	s.verificationMu.RLock()
	defer s.verificationMu.RUnlock()
	result, ok := s.verificationCache[caseID]
	return result, ok
}

func (s *Service) rememberVerification(caseID string, result VerificationResult) {
	if caseID == "" {
		return
	}
	s.verificationMu.Lock()
	defer s.verificationMu.Unlock()
	s.verificationCache[caseID] = result
}

func (s *Service) verifyUncached(m ReleaseManifest, c VerificationCredential, digest string) VerificationResult {
	if digest == "" {
		return VerificationResult{Reason: "缺少清单摘要"}
	}
	if m.ManifestID == "" || m.CaseID == "" || m.RevisionID == "" || m.ContentDigest == "" {
		return VerificationResult{Reason: "清单不匹配", Digest: digest}
	}
	want := ManifestDigest(m)
	if digest != want {
		return VerificationResult{Reason: "摘要不匹配", Digest: want}
	}
	if c.ManifestID != m.ManifestID {
		return VerificationResult{Reason: "清单不匹配", Digest: digest}
	}
	if m.CredentialID != "" && m.CredentialID != c.CredentialID {
		return VerificationResult{Reason: "凭据标识不匹配", Digest: digest}
	}
	if c.Algorithm != "HMAC-SHA256" {
		return VerificationResult{Reason: "算法不匹配", Digest: digest}
	}
	if c.Digest != digest {
		return VerificationResult{Reason: "摘要不匹配", Digest: digest}
	}
	if !VerifyCredential(s.Key, m, c, digest) {
		return VerificationResult{Reason: "HMAC 签名不匹配", Digest: digest}
	}
	return VerificationResult{Valid: true, Reason: "验证通过", Digest: digest}
}
func ManifestDigest(m ReleaseManifest) string { d, _ := canonicalManifestDigest(m); return d }

package release

import "encoding/json"

func CanonicalJSON(m ReleaseManifest) (string, error) { b, e := json.Marshal(m); return string(b), e }
func ReleaseSummary(m ReleaseManifest, c VerificationCredential) map[string]string {
	return map[string]string{"manifestID": m.ManifestID, "caseID": m.CaseID, "revisionID": m.RevisionID, "credentialID": c.CredentialID, "algorithm": c.Algorithm}
}

package release

import (
	"encoding/base64"
	"encoding/json"
)

func EncodeArtifact(a Artifact) (string, error) {
	b, e := json.Marshal(a)
	if e != nil {
		return "", e
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func DecodeArtifact(v string) (Artifact, error) {
	b, e := base64.RawURLEncoding.DecodeString(v)
	if e != nil {
		return Artifact{}, e
	}
	var a Artifact
	e = json.Unmarshal(b, &a)
	return a, e
}
func CredentialText(c VerificationCredential) string {
	return c.CredentialID + " | " + c.Algorithm + " | " + c.Signature
}

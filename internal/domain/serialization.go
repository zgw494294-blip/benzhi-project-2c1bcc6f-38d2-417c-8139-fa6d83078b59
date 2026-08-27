package domain

import "encoding/json"

func MarshalCase(c *ArchiveCase) ([]byte, error) { return json.Marshal(c) }
func UnmarshalCase(b []byte) (*ArchiveCase, error) {
	var c ArchiveCase
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}
func MarshalEvent(e Event) ([]byte, error) { return json.Marshal(e) }

package domain

import "errors"

type Command struct {
	Key             string `json:"key"`
	Actor           string `json:"actor"`
	Role            Role   `json:"role"`
	ExpectedVersion int    `json:"expectedVersion"`
}

func (c Command) Validate() error {
	if c.Key == "" {
		return errors.New("command key 不能为空")
	}
	if c.Actor == "" {
		return errors.New("actor 不能为空")
	}
	if c.ExpectedVersion < 0 {
		return errors.New("expectedVersion 不能为负数")
	}
	return ValidateRoleActor(c.Actor, c.Role)
}
func (c *ArchiveCase) ApplyFindingResolution(findingID, note, actor string) bool {
	r := c.CurrentRevision()
	if r == nil {
		return false
	}
	for i := range r.Findings {
		if r.Findings[i].FindingID == findingID && r.Findings[i].Status == FindingOpen {
			r.Findings[i].Status = FindingResolved
			r.Findings[i].ResolutionNote = note
			r.Findings[i].ResolvedBy = actor
			return true
		}
	}
	return false
}

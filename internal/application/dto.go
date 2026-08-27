package application

import "archive-release/internal/domain"

type CaseView struct {
	Case     *domain.ArchiveCase `json:"case"`
	Findings []domain.Finding    `json:"findings"`
	Events   []domain.Event      `json:"events"`
}

func (s *Service) View(id string) (CaseView, error) {
	c, e := s.repo.Get(id)
	if e != nil {
		return CaseView{}, e
	}
	var f []domain.Finding
	if r := c.CurrentRevision(); r != nil {
		f = r.Findings
	}
	return CaseView{Case: c, Findings: f, Events: s.Timeline(id)}, nil
}

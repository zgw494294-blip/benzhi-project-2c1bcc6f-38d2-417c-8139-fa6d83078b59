package application

import (
	"archive-release/internal/domain"
	"strings"
)

func FilterCases(cases []*domain.ArchiveCase, status domain.CaseStatus, query string) []*domain.ArchiveCase {
	out := make([]*domain.ArchiveCase, 0)
	q := strings.ToLower(strings.TrimSpace(query))
	for _, c := range cases {
		if status != "" && c.Status != status {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(c.Title), q) && !strings.Contains(strings.ToLower(c.ArchiveUnit), q) {
			continue
		}
		out = append(out, c)
	}
	SortCases(out)
	return out
}
func (s *Service) Search(status domain.CaseStatus, q string) []*domain.ArchiveCase {
	return FilterCases(s.repo.List(), status, q)
}
func (s *Service) Findings(id string) []domain.Finding {
	c, e := s.repo.Get(id)
	if e != nil || c.CurrentRevision() == nil {
		return nil
	}
	return append([]domain.Finding(nil), c.CurrentRevision().Findings...)
}

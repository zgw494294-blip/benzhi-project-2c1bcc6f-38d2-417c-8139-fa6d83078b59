package application

import (
	"archive-release/internal/domain"
	"archive-release/internal/persistence"
	"sort"
)

type Dashboard struct {
	Total         int `json:"total"`
	Draft         int `json:"draft"`
	ReviewPending int `json:"reviewPending"`
	Approved      int `json:"approved"`
	Frozen        int `json:"frozen"`
	Blocking      int `json:"blocking"`
}

func (s *Service) Dashboard() Dashboard {
	d := Dashboard{}
	for _, c := range s.repo.List() {
		d.Total++
		switch c.Status {
		case domain.StatusDraft:
			d.Draft++
		case domain.StatusReviewPending:
			d.ReviewPending++
		case domain.StatusApproved:
			d.Approved++
		case domain.StatusFrozen:
			d.Frozen++
		}
		d.Blocking += c.BlockingOpen()
	}
	return d
}
func (s *Service) Audit(id string) []domain.AuditEntry {
	return domain.AuditTrail(persistence.Timeline(s.repo, id))
}
func SortCases(cases []*domain.ArchiveCase) {
	sort.Slice(cases, func(i, j int) bool { return cases[i].UpdatedAt.After(cases[j].UpdatedAt) })
}

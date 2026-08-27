package persistence

import (
	"archive-release/internal/domain"
	"sort"
)

func Timeline(r Repository, caseID string) []domain.Event {
	ev := r.Events(caseID)
	sort.Slice(ev, func(i, j int) bool { return ev[i].Seq < ev[j].Seq })
	return ev
}
func VerifyChain(events []domain.Event) bool {
	prev := ""
	for _, e := range events {
		if e.PrevDigest != prev {
			return false
		}
		prev = e.Digest
	}
	return true
}

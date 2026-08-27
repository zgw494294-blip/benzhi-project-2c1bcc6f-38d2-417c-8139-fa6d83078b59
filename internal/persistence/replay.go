package persistence

import (
	"archive-release/internal/domain"
	"errors"
)

func Replay(events []domain.Event) (map[string]int, error) {
	counts := map[string]int{}
	if !VerifyChain(events) {
		return nil, errors.New("事件摘要链校验失败")
	}
	for _, e := range events {
		counts[e.Type]++
	}
	return counts, nil
}
func LastSequence(events []domain.Event) int64 {
	if len(events) == 0 {
		return 0
	}
	return events[len(events)-1].Seq
}
func EventTypes(events []domain.Event) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, e := range events {
		if !seen[e.Type] {
			seen[e.Type] = true
			out = append(out, e.Type)
		}
	}
	return out
}

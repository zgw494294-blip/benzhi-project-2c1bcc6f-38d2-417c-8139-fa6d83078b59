package persistence

import "archive-release/internal/domain"

type Snapshot struct {
	Case    *domain.ArchiveCase `json:"case"`
	LastSeq int64               `json:"lastSeq"`
	Digest  string              `json:"digest"`
}

func BuildSnapshot(c *domain.ArchiveCase, events []domain.Event) Snapshot {
	var seq int64
	dig := ""
	if len(events) > 0 {
		seq = events[len(events)-1].Seq
		dig = events[len(events)-1].Digest
	}
	return Snapshot{Case: c, LastSeq: seq, Digest: dig}
}

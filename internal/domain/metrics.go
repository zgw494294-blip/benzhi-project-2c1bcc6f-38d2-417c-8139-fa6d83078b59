package domain

import "time"

type QualityMetrics struct {
	Pages          int       `json:"pages"`
	Blocking       int       `json:"blocking"`
	Notices        int       `json:"notices"`
	AverageClarity float64   `json:"averageClarity"`
	MaxSkew        float64   `json:"maxSkew"`
	MeasuredAt     time.Time `json:"measuredAt"`
}

func Measure(r *ScanRevision, at time.Time) QualityMetrics {
	m := QualityMetrics{MeasuredAt: at}
	if r == nil {
		return m
	}
	var total float64
	for _, p := range r.Pages {
		m.Pages++
		total += p.Clarity
		if p.Skew > m.MaxSkew {
			m.MaxSkew = p.Skew
		}
	}
	for _, f := range r.Findings {
		if f.Status != FindingOpen {
			continue
		}
		if f.Severity == SeverityBlocking {
			m.Blocking++
		} else {
			m.Notices++
		}
	}
	if m.Pages > 0 {
		m.AverageClarity = total / float64(m.Pages)
	}
	return m
}
func (c *ArchiveCase) IsPublishable() bool {
	return c.Status == StatusApproved && c.BlockingOpen() == 0
}
func (c *ArchiveCase) RevisionCount() int { return len(c.Revisions) }

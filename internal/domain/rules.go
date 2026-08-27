package domain

import (
	"fmt"
	"sort"
)

const RuleSetVersion = "2026.1"

func EvaluateRules(revisionID string, pages []PageMeasurement) []Finding {
	pages = append([]PageMeasurement(nil), pages...)
	sort.Slice(pages, func(i, j int) bool { return pages[i].PageNumber < pages[j].PageNumber })
	out := []Finding{}
	idx := 1
	for _, p := range pages {
		add := func(code string, sev Severity, evidence string) {
			out = append(out, Finding{FindingID: fmt.Sprintf("%s-f%03d", revisionID, idx), RevisionID: revisionID, PageNumber: p.PageNumber, RuleCode: code, Severity: sev, Evidence: evidence, Status: FindingOpen})
			idx++
		}
		if p.Clarity < 80 {
			add("CLARITY", SeverityBlocking, fmt.Sprintf("清晰度 %.1f 低于 80", p.Clarity))
		} else if p.Clarity < 90 {
			add("CLARITY_NOTICE", SeverityNotice, fmt.Sprintf("清晰度 %.1f 建议复核", p.Clarity))
		}
		if p.Skew > 3 {
			add("SKEW", SeverityBlocking, fmt.Sprintf("倾斜 %.1f° 超过 3°", p.Skew))
		} else if p.Skew > 1.5 {
			add("SKEW_NOTICE", SeverityNotice, fmt.Sprintf("倾斜 %.1f° 需要关注", p.Skew))
		}
		if p.CropRatio < 0.98 {
			add("CROP", SeverityBlocking, fmt.Sprintf("裁切完整度 %.2f 低于 0.98", p.CropRatio))
		}
		if !p.ColorTarget {
			add("COLOR_TARGET", SeverityNotice, "未检测到色标")
		}
	}
	return out
}

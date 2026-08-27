package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type CaseStatus string

const (
	StatusDraft         CaseStatus = "draft"
	StatusReviewPending CaseStatus = "review_pending"
	StatusApproved      CaseStatus = "approved"
	StatusReturned      CaseStatus = "returned"
	StatusFrozen        CaseStatus = "frozen"
)

type Severity string

const (
	SeverityBlocking Severity = "blocking"
	SeverityNotice   Severity = "notice"
)

type FindingStatus string

const (
	FindingOpen     FindingStatus = "open"
	FindingResolved FindingStatus = "resolved"
)

type Role string

const (
	RoleArchivist   Role = "archivist"
	RoleConservator Role = "conservator"
	RoleReviewer    Role = "reviewer"
)

type PageMeasurement struct {
	PageNumber  int     `json:"pageNumber"`
	Clarity     float64 `json:"clarity"`
	Skew        float64 `json:"skew"`
	CropRatio   float64 `json:"cropRatio"`
	ColorTarget bool    `json:"colorTarget"`
}
type ScanRevision struct {
	RevisionID       string            `json:"revisionID"`
	CaseID           string            `json:"caseID"`
	ParentRevisionID string            `json:"parentRevisionID"`
	SubmittedBy      string            `json:"submittedBy"`
	SubmittedAt      time.Time         `json:"submittedAt"`
	Metadata         map[string]string `json:"metadata"`
	Pages            []PageMeasurement `json:"pages"`
	RuleSetVersion   string            `json:"ruleSetVersion"`
	ContentDigest    string            `json:"contentDigest"`
	Findings         []Finding         `json:"findings"`
	Stats            []FindingStats    `json:"stats,omitempty"`
}
type Finding struct {
	FindingID      string        `json:"findingID"`
	RevisionID     string        `json:"revisionID"`
	PageNumber     int           `json:"pageNumber"`
	RuleCode       string        `json:"ruleCode"`
	Severity       Severity      `json:"severity"`
	Evidence       string        `json:"evidence"`
	Status         FindingStatus `json:"status"`
	ResolutionNote string        `json:"resolutionNote,omitempty"`
	ResolvedBy     string        `json:"resolvedBy,omitempty"`
	ResolvedAt     *time.Time    `json:"resolvedAt,omitempty"`
}
type FindingStats struct {
	RuleCode       string   `json:"ruleCode"`
	Severity       Severity `json:"severity"`
	PageNumber     int      `json:"pageNumber"`
	Open           int      `json:"open"`
	Resolved       int      `json:"resolved"`
	AverageClarity float64  `json:"averageClarity"`
	MaxSkew        float64  `json:"maxSkew"`
}
type ReviewDecision struct {
	Approved bool      `json:"approved"`
	Reviewer string    `json:"reviewer"`
	Note     string    `json:"note"`
	At       time.Time `json:"at"`
}
type ArchiveCase struct {
	CaseID            string          `json:"caseID"`
	Title             string          `json:"title"`
	ArchiveUnit       string          `json:"archiveUnit"`
	Creator           string          `json:"creator"`
	NormalizedTitle   string          `json:"normalizedTitle,omitempty"`
	NormalizedUnit    string          `json:"normalizedArchiveUnit,omitempty"`
	NormalizedCreator string          `json:"normalizedCreator,omitempty"`
	MetadataDigest    string          `json:"metadataDigest,omitempty"`
	Status            CaseStatus      `json:"status"`
	CurrentRevisionID string          `json:"currentRevisionID"`
	Version           int             `json:"version"`
	CreatedAt         time.Time       `json:"createdAt"`
	UpdatedAt         time.Time       `json:"updatedAt"`
	Revisions         []ScanRevision  `json:"revisions"`
	Review            *ReviewDecision `json:"review,omitempty"`
	ManifestID        string          `json:"manifestID,omitempty"`
	ManifestJSON      string          `json:"manifestJSON,omitempty"`
	CredentialJSON    string          `json:"credentialJSON,omitempty"`
	ArtifactDigest    string          `json:"artifactDigest,omitempty"`
}

func ValidateCaseInput(title, unit, creator string) error {
	if strings.TrimSpace(title) == "" || strings.TrimSpace(unit) == "" || strings.TrimSpace(creator) == "" {
		return errors.New("标题、馆藏单元和创建者不能为空")
	}
	return nil
}
func ValidateRevision(r ScanRevision) error {
	if r.CaseID == "" || r.SubmittedBy == "" || len(r.Pages) == 0 {
		return errors.New("修订必须包含案卷、提交者和页面")
	}
	seen := map[int]bool{}
	var errs []string
	for _, p := range r.Pages {
		if p.PageNumber < 1 || seen[p.PageNumber] {
			errs = append(errs, fmt.Sprintf("page %d: 页码必须唯一且从1开始", p.PageNumber))
		} else {
			seen[p.PageNumber] = true
		}
		if p.Clarity < 0 || p.Clarity > 100 {
			errs = append(errs, fmt.Sprintf("page %d clarity 必须在0-100之间", p.PageNumber))
		}
		if p.Skew < 0 || p.Skew > 45 {
			errs = append(errs, fmt.Sprintf("page %d skew 必须在0-45之间", p.PageNumber))
		}
		if p.CropRatio < 0 || p.CropRatio > 1 {
			errs = append(errs, fmt.Sprintf("page %d cropRatio 必须在0-1之间", p.PageNumber))
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}
func Digest(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func (c *ArchiveCase) CurrentRevision() *ScanRevision {
	for i := range c.Revisions {
		if c.Revisions[i].RevisionID == c.CurrentRevisionID {
			return &c.Revisions[i]
		}
	}
	return nil
}
func (c *ArchiveCase) BlockingOpen() int {
	n := 0
	for _, r := range c.Revisions {
		if r.RevisionID != c.CurrentRevisionID {
			continue
		}
		for _, f := range r.Findings {
			if f.Severity == SeverityBlocking && f.Status == FindingOpen {
				n++
			}
		}
	}
	return n
}
func (c *ArchiveCase) FindingStats() []FindingStats {
	r := c.CurrentRevision()
	if r == nil {
		return []FindingStats{}
	}
	type key struct {
		code string
		sev  Severity
		page int
	}
	groups := map[key]*FindingStats{}
	clarity := map[int]float64{}
	for _, p := range r.Pages {
		clarity[p.PageNumber] = p.Clarity
	}
	for _, f := range r.Findings {
		k := key{f.RuleCode, f.Severity, f.PageNumber}
		g := groups[k]
		if g == nil {
			g = &FindingStats{RuleCode: f.RuleCode, Severity: f.Severity, PageNumber: f.PageNumber}
			groups[k] = g
		}
		if f.Status == FindingOpen {
			g.Open++
		} else {
			g.Resolved++
		}
		if v, ok := clarity[f.PageNumber]; ok {
			g.AverageClarity = v
		}
	}
	for _, g := range groups {
		for _, p := range r.Pages {
			if p.PageNumber == g.PageNumber && p.Skew > g.MaxSkew {
				g.MaxSkew = p.Skew
			}
		}
	}
	out := make([]FindingStats, 0, len(groups))
	for _, g := range groups {
		out = append(out, *g)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PageNumber != out[j].PageNumber {
			return out[i].PageNumber < out[j].PageNumber
		}
		if out[i].RuleCode != out[j].RuleCode {
			return out[i].RuleCode < out[j].RuleCode
		}
		return out[i].Severity < out[j].Severity
	})
	return out
}
func CanonicalMap(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+"="+m[k])
	}
	return out
}

package application

import (
	"archive-release/internal/domain"
	"archive-release/internal/release"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type ReviewQueueItem struct {
	CaseID         string            `json:"caseID"`
	Status         domain.CaseStatus `json:"status"`
	Version        int               `json:"version"`
	RevisionID     string            `json:"revisionID"`
	ContentDigest  string            `json:"contentDigest"`
	RuleSetVersion string            `json:"ruleSetVersion"`
	SubmittedBy    string            `json:"submittedBy"`
	SubmittedAt    time.Time         `json:"submittedAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
	Blocking       int               `json:"blocking"`
	Notices        int               `json:"notices"`
}

func (s *Service) FindingsFiltered(id string, severity, rule string, page int) ([]domain.Finding, error) {
	if severity != "" && severity != string(domain.SeverityBlocking) && severity != string(domain.SeverityNotice) {
		return nil, errors.New("severity 参数无效")
	}
	if page < 0 {
		return nil, errors.New("pageNumber 参数必须为正数")
	}
	if rule != "" {
		valid := map[string]bool{"CLARITY": true, "CLARITY_NOTICE": true, "SKEW": true, "SKEW_NOTICE": true, "CROP": true, "COLOR_TARGET": true}
		if !valid[rule] {
			return nil, errors.New("ruleCode 参数无效")
		}
	}
	c, err := s.repo.Get(id)
	if err != nil {
		return nil, err
	}
	r := c.CurrentRevision()
	if r == nil {
		return []domain.Finding{}, nil
	}
	out := make([]domain.Finding, 0)
	for _, f := range r.Findings {
		if severity != "" && string(f.Severity) != severity || rule != "" && f.RuleCode != rule || page > 0 && f.PageNumber != page {
			continue
		}
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PageNumber != out[j].PageNumber {
			return out[i].PageNumber < out[j].PageNumber
		}
		return out[i].FindingID < out[j].FindingID
	})
	return out, nil
}

func (s *Service) FindingStats(id string) ([]domain.FindingStats, error) {
	c, err := s.repo.Get(id)
	if err != nil {
		return nil, err
	}
	if c.CurrentRevision() == nil {
		return []domain.FindingStats{}, nil
	}
	return c.FindingStats(), nil
}

func (s *Service) ResolveFindings(id, actor string, role domain.Role, expected int, key string, resolutions map[string]string) (*domain.ArchiveCase, error) {
	if err := authorize(role, domain.RoleConservator); err != nil {
		return nil, err
	}
	if len(resolutions) == 0 {
		return nil, errors.New("整改列表不能为空")
	}
	if key != "" {
		if v, ok := s.repo.FindCommand(key); ok {
			return v.(*domain.ArchiveCase), nil
		}
	}
	c, err := s.repo.Get(id)
	if err != nil {
		return nil, err
	}
	if c.Version != expected {
		return nil, errors.New("版本冲突")
	}
	if c.Status == domain.StatusFrozen {
		return nil, errors.New("案卷已冻结")
	}
	r := c.CurrentRevision()
	if r == nil {
		return nil, errors.New("没有当前修订")
	}
	indices := make(map[string]int, len(r.Findings))
	for i := range r.Findings {
		indices[r.Findings[i].FindingID] = i
	}
	for fid, note := range resolutions {
		i, ok := indices[fid]
		if !ok {
			return nil, fmt.Errorf("FindingID 不存在: %s", fid)
		}
		if r.Findings[i].Status != domain.FindingOpen {
			return nil, fmt.Errorf("FindingID 已解决: %s", fid)
		}
		if strings.TrimSpace(note) == "" {
			return nil, fmt.Errorf("FindingID %s 整改说明不能为空", fid)
		}
	}
	now := s.clock.Now()
	for fid, note := range resolutions {
		f := &r.Findings[indices[fid]]
		f.Status = domain.FindingResolved
		f.ResolutionNote = strings.TrimSpace(note)
		f.ResolvedBy = actor
		f.ResolvedAt = &now
	}
	c.Version++
	c.UpdatedAt = now
	r.Stats = c.FindingStats()
	if c.BlockingOpen() == 0 {
		c.Status = domain.StatusReviewPending
	}
	e := s.event("FindingsResolved", c, actor, role, key, resolutions)
	if err = s.repo.Save(c, e); err != nil {
		return nil, err
	}
	if key != "" {
		s.repo.SetCommand(key, c)
	}
	return c, nil
}

func (s *Service) ReviewQueue() []*ReviewQueueItem {
	items := []*ReviewQueueItem{}
	for _, c := range s.repo.List() {
		if c.Status != domain.StatusReviewPending {
			continue
		}
		r := c.CurrentRevision()
		if r == nil {
			continue
		}
		notices := 0
		for _, f := range r.Findings {
			if f.Severity == domain.SeverityNotice {
				notices++
			}
		}
		items = append(items, &ReviewQueueItem{CaseID: c.CaseID, Status: c.Status, Version: c.Version, RevisionID: r.RevisionID, ContentDigest: r.ContentDigest, RuleSetVersion: r.RuleSetVersion, SubmittedBy: r.SubmittedBy, SubmittedAt: r.SubmittedAt, UpdatedAt: c.UpdatedAt, Blocking: c.BlockingOpen(), Notices: notices})
	}
	sort.Slice(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if !a.UpdatedAt.Equal(b.UpdatedAt) {
			return a.UpdatedAt.Before(b.UpdatedAt)
		}
		return a.SubmittedAt.Before(b.SubmittedAt)
	})
	return items
}

func (s *Service) Preview(id string) (map[string]any, error) {
	c, err := s.repo.Get(id)
	if err != nil {
		return nil, err
	}
	if c.Status == domain.StatusFrozen {
		a, e := s.ArtifactForCase(id)
		if e != nil {
			return nil, e
		}
		return map[string]any{"manifest": a.Manifest, "manifestID": a.Manifest.ManifestID, "revisionID": a.Manifest.RevisionID, "digest": a.Digest, "frozenAt": a.Manifest.FrozenAt, "previewDigest": a.Digest, "expectedVersion": c.Version}, nil
	}
	if c.Status != domain.StatusApproved {
		return nil, errors.New("仅可预览已批准案卷")
	}
	if c.BlockingOpen() > 0 {
		return nil, errors.New("仍有阻断发现")
	}
	r := c.CurrentRevision()
	at := s.clock.Now()
	mid := "man-" + domain.Digest(struct{ C, R string }{c.CaseID, r.RevisionID})[:12]
	m, digest, err := release.CanonicalManifest(c, r, mid, at)
	if err != nil {
		return nil, err
	}
	return map[string]any{"manifest": m, "manifestID": m.ManifestID, "revisionID": m.RevisionID, "digest": digest, "frozenAt": at, "previewDigest": digest, "expectedVersion": c.Version}, nil
}

func (s *Service) FreezeWithPreview(id, actor string, role domain.Role, expected int, key, previewDigest string) (*domain.ArchiveCase, release.ReleaseManifest, release.VerificationCredential, error) {
	if err := authorize(role, domain.RoleReviewer); err != nil {
		return nil, release.ReleaseManifest{}, release.VerificationCredential{}, err
	}
	if previewDigest == "" {
		c, a, b, e := s.Freeze(id, actor, role, expected, key)
		var m release.ReleaseManifest
		var cred release.VerificationCredential
		if a != nil {
			m = a.(release.ReleaseManifest)
		}
		if b != nil {
			cred = b.(release.VerificationCredential)
		}
		return c, m, cred, e
	}
	if key != "" {
		if v, ok := s.repo.FindCommand(key); ok {
			c := v.(*domain.ArchiveCase)
			a, err := s.ArtifactForCase(id)
			if err != nil {
				return c, release.ReleaseManifest{}, release.VerificationCredential{}, nil
			}
			return c, a.Manifest, a.Credential, nil
		}
	}
	c, err := s.repo.Get(id)
	if err != nil {
		return nil, release.ReleaseManifest{}, release.VerificationCredential{}, err
	}
	if c.Version != expected {
		return nil, release.ReleaseManifest{}, release.VerificationCredential{}, errors.New("版本冲突")
	}
	if c.Status != domain.StatusApproved || c.BlockingOpen() > 0 {
		return nil, release.ReleaseManifest{}, release.VerificationCredential{}, errors.New("仅可冻结无阻断项的已批准案卷")
	}
	r := c.CurrentRevision()
	mid := "man-" + domain.Digest(struct{ C, R string }{c.CaseID, r.RevisionID})[:12]
	m, d, e := release.CanonicalManifest(c, r, mid, s.clock.Now())
	if e != nil {
		return nil, m, release.VerificationCredential{}, e
	}
	if d != previewDigest {
		return nil, m, release.VerificationCredential{}, errors.New("预览摘要不一致")
	}
	cred := release.IssueCredential(s.release.Key, m, d, actor, m.FrozenAt)
	m.CredentialID = cred.CredentialID
	c.Status = domain.StatusFrozen
	c.ManifestID = m.ManifestID
	c.Version++
	c.UpdatedAt = s.clock.Now()
	mb, _ := json.Marshal(m)
	cb, _ := json.Marshal(cred)
	c.ManifestJSON = string(mb)
	c.CredentialJSON = string(cb)
	c.ArtifactDigest = d
	ev := s.event("CaseFrozen", c, actor, role, key, m)
	s.catalog.Put(release.Artifact{Manifest: m, Credential: cred, Digest: d})
	if e = s.repo.Save(c, ev); e != nil {
		return nil, m, cred, e
	}
	if key != "" {
		s.repo.SetCommand(key, c)
	}
	return c, m, cred, nil
}

func (s *Service) ArtifactForCase(id string) (release.Artifact, error) {
	if a, ok := s.catalog.Get(id); ok {
		return a, nil
	}
	c, err := s.repo.Get(id)
	if err != nil {
		return release.Artifact{}, err
	}
	if c.Status != domain.StatusFrozen || c.ManifestJSON == "" {
		return release.Artifact{}, errors.New("发布物不存在")
	}
	var m release.ReleaseManifest
	var cred release.VerificationCredential
	if json.Unmarshal([]byte(c.ManifestJSON), &m) != nil || json.Unmarshal([]byte(c.CredentialJSON), &cred) != nil {
		return release.Artifact{}, errors.New("发布物不存在")
	}
	a := release.Artifact{Manifest: m, Credential: cred, Digest: c.ArtifactDigest}
	s.catalog.Put(a)
	return a, nil
}

func (s *Service) VerifySubmitted(id string, a release.Artifact) release.VerificationResult {
	c, err := s.repo.Get(id)
	if err != nil || c.Status != domain.StatusFrozen {
		return release.VerificationResult{Reason: "发布物不存在"}
	}
	var result release.VerificationResult
	if a.Manifest.CaseID != id {
		result = release.VerificationResult{Reason: "清单不匹配", Digest: a.Digest}
	} else {
		result = s.release.Verify(a.Manifest, a.Credential, a.Digest)
	}
	if c, err := s.repo.Get(id); err == nil {
		ev := s.event("ArtifactVerified", c, "system", domain.RoleReviewer, "", result)
		_ = s.repo.Save(c, ev)
	}
	return result
}

func (s *Service) VerifySubmittedContext(ctx context.Context, id string, a release.Artifact) release.VerificationResult {
	select {
	case <-ctx.Done():
		return release.VerificationResult{Reason: "请求已取消"}
	default:
		return s.VerifySubmitted(id, a)
	}
}

package application

import (
	"archive-release/internal/domain"
	"archive-release/internal/persistence"
	"archive-release/internal/release"
	"encoding/json"
	"errors"
	"strings"
)

func (s *Service) ResolveFinding(id, findingID, actor string, role domain.Role, expected int, key, note string) (*domain.ArchiveCase, error) {
	if err := authorize(role, domain.RoleConservator); err != nil {
		return nil, err
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
	found := false
	for i := range r.Findings {
		if r.Findings[i].FindingID == findingID {
			if r.Findings[i].Status != domain.FindingOpen {
				return nil, errors.New("发现已解决")
			}
			if strings.TrimSpace(note) == "" {
				return nil, errors.New("整改说明不能为空")
			}
			now := s.clock.Now()
			r.Findings[i].Status = domain.FindingResolved
			r.Findings[i].ResolutionNote = note
			r.Findings[i].ResolvedBy = actor
			r.Findings[i].ResolvedAt = &now
			found = true
		}
	}
	if !found {
		return nil, errors.New("FindingID 不存在")
	}
	c.Version++
	c.UpdatedAt = s.clock.Now()
	if current := c.CurrentRevision(); current != nil {
		current.Stats = c.FindingStats()
	}
	if c.BlockingOpen() == 0 {
		c.Status = domain.StatusReviewPending
	}
	e := s.event("FindingResolved", c, actor, role, key, findingID)
	if err = s.repo.Save(c, e); err != nil {
		return nil, err
	}
	if key != "" {
		s.repo.SetCommand(key, c)
	}
	return c, nil
}
func (s *Service) Review(id, actor string, role domain.Role, expected int, key string, approved bool, note string) (*domain.ArchiveCase, error) {
	if err := authorize(role, domain.RoleReviewer); err != nil {
		return nil, err
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
	if err := c.CanReview(); err != nil {
		return nil, err
	}
	r := c.CurrentRevision()
	if r == nil || r.SubmittedBy == actor {
		return nil, errors.New("复核员必须独立于提交者")
	}
	if c.Status != domain.StatusReviewPending {
		return nil, errors.New("案卷尚未达到待复核状态")
	}
	if !approved && note == "" {
		return nil, errors.New("退回必须填写原因")
	}
	c.Review = &domain.ReviewDecision{Approved: approved, Reviewer: actor, Note: note, At: s.clock.Now()}
	if approved {
		c.Status = domain.StatusApproved
	} else {
		c.Status = domain.StatusReturned
	}
	c.Version++
	c.UpdatedAt = s.clock.Now()
	e := s.event("ReviewDecided", c, actor, role, key, c.Review)
	if err = s.repo.Save(c, e); err != nil {
		return nil, err
	}
	if key != "" {
		s.repo.SetCommand(key, c)
	}
	return c, nil
}
func (s *Service) Freeze(id, actor string, role domain.Role, expected int, key string) (*domain.ArchiveCase, any, any, error) {
	if err := authorize(role, domain.RoleReviewer); err != nil {
		return nil, nil, nil, err
	}
	if key != "" {
		if v, ok := s.repo.FindCommand(key); ok {
			return v.(*domain.ArchiveCase), nil, nil, nil
		}
	}
	c, err := s.repo.Get(id)
	if err != nil {
		return nil, nil, nil, err
	}
	if c.Version != expected {
		return nil, nil, nil, errors.New("版本冲突")
	}
	if c.Status != domain.StatusApproved || c.BlockingOpen() > 0 {
		return nil, nil, nil, errors.New("仅可冻结无阻断项的已批准案卷")
	}
	m, cred, e := s.release.Freeze(c, actor, s.clock.Now().Unix())
	if e != nil {
		return nil, nil, nil, e
	}
	c.Status = domain.StatusFrozen
	c.ManifestID = m.ManifestID
	c.Version++
	c.UpdatedAt = s.clock.Now()
	ev := s.event("CaseFrozen", c, actor, role, key, m)
	mb, _ := json.Marshal(m)
	cb, _ := json.Marshal(cred)
	c.ManifestJSON, c.CredentialJSON, c.ArtifactDigest = string(mb), string(cb), cred.Digest
	s.catalog.Put(release.Artifact{Manifest: m, Credential: cred, Digest: cred.Digest})
	if e = s.repo.Save(c, ev); e != nil {
		return nil, nil, nil, e
	}
	if key != "" {
		s.repo.SetCommand(key, c)
	}
	return c, m, cred, nil
}
func (s *Service) Timeline(id string) []domain.Event { return persistence.Timeline(s.repo, id) }

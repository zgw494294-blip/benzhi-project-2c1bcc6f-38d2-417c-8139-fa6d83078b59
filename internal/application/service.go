package application

import (
	"archive-release/internal/domain"
	"archive-release/internal/persistence"
	"archive-release/internal/release"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

type Clock interface{ Now() time.Time }
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now().UTC() }

type Service struct {
	repo    persistence.Repository
	release *release.Service
	clock   Clock
	mu      sync.Mutex
	seq     int64
	catalog *release.Catalog
}

func New(repo persistence.Repository, key string) *Service {
	return &Service{repo: repo, release: release.New(key), clock: RealClock{}, catalog: release.NewCatalog()}
}

func (s *Service) Artifact(caseID string) (release.Artifact, bool) {
	if a, ok := s.catalog.Get(caseID); ok {
		return a, true
	}
	a, err := s.ArtifactForCase(caseID)
	return a, err == nil
}
func (s *Service) VerifyArtifact(caseID string) release.VerificationResult {
	a, ok := s.catalog.Get(caseID)
	if !ok {
		var err error
		a, err = s.ArtifactForCase(caseID)
		if err != nil {
			return release.VerificationResult{Reason: "发布物不存在"}
		}
	}
	result := s.release.Verify(a.Manifest, a.Credential, a.Digest)
	if c, err := s.repo.Get(caseID); err == nil {
		ev := s.event("ArtifactVerified", c, "system", domain.RoleReviewer, "", result)
		_ = s.repo.Save(c, ev)
	}
	return result
}
func (s *Service) next() int64 { s.mu.Lock(); defer s.mu.Unlock(); s.seq++; return s.seq }
func (s *Service) event(typ string, c *domain.ArchiveCase, actor string, role domain.Role, key string, payload any) domain.Event {
	e := domain.NewEvent(s.next(), typ, c.CaseID, actor, role, key, s.clock.Now(), payload, lastDigest(s.repo.Events(c.CaseID)))
	e.Version = c.Version
	e.Digest = domain.Digest(e)
	return e
}
func lastDigest(es []domain.Event) string {
	if len(es) == 0 {
		return ""
	}
	return es[len(es)-1].Digest
}
func authorize(role domain.Role, allowed ...domain.Role) error {
	for _, r := range allowed {
		if role == r {
			return nil
		}
	}
	return errors.New("角色无权执行此操作")
}
func (s *Service) CreateCase(title, unit, creator, actor string, role domain.Role, key string) (*domain.ArchiveCase, error) {
	if err := authorize(role, domain.RoleArchivist); err != nil {
		return nil, err
	}
	if key != "" {
		if v, ok := s.repo.FindCommand(key); ok {
			return v.(*domain.ArchiveCase), nil
		}
	}
	if err := domain.ValidateCaseInput(title, unit, creator); err != nil {
		return nil, err
	}
	nt, nu, nc := domain.NormalizeText(title), domain.NormalizeText(unit), domain.NormalizeText(creator)
	md := domain.Digest(struct{ Unit, Title string }{nu, nt})
	if finder, ok := s.repo.(persistence.MetadataFinder); ok {
		if existing, found := finder.FindMetadata(md); found {
			return nil, fmt.Errorf("重复案卷，冲突案卷标识: %s", existing.CaseID)
		}
	}
	now := s.clock.Now()
	c := &domain.ArchiveCase{CaseID: fmt.Sprintf("case-%d", now.UnixNano()), Title: title, ArchiveUnit: unit, Creator: creator, NormalizedTitle: nt, NormalizedUnit: nu, NormalizedCreator: nc, MetadataDigest: md, Status: domain.StatusDraft, Version: 1, CreatedAt: now, UpdatedAt: now}
	if err := s.repo.Create(c); err != nil {
		return nil, err
	}
	e := s.event("CaseCreated", c, actor, role, key, nil)
	_ = s.repo.Save(c, e)
	if key != "" {
		s.repo.SetCommand(key, c)
	}
	return c, nil
}
func (s *Service) GetCase(id string) (*domain.ArchiveCase, error) { return s.repo.Get(id) }
func (s *Service) ListCases() []*domain.ArchiveCase               { return s.repo.List() }
func (s *Service) SubmitRevision(id, actor string, role domain.Role, expected int, key string, pages []domain.PageMeasurement, metadata map[string]string, parent string) (*domain.ArchiveCase, error) {
	if err := authorize(role, domain.RoleArchivist, domain.RoleConservator); err != nil {
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
	if parent != "" && parent != c.CurrentRevisionID {
		return nil, errors.New("父修订不匹配")
	}
	r := domain.ScanRevision{RevisionID: fmt.Sprintf("rev-%d", s.clock.Now().UnixNano()), CaseID: id, ParentRevisionID: parent, SubmittedBy: actor, SubmittedAt: s.clock.Now(), Metadata: metadata, Pages: pages, RuleSetVersion: domain.RuleSetVersion}
	if err := domain.ValidateRevision(r); err != nil {
		return nil, err
	}
	sort.Slice(r.Pages, func(i, j int) bool { return r.Pages[i].PageNumber < r.Pages[j].PageNumber })
	r.ContentDigest = release.StablePageDigest(&r)
	r.Findings = domain.EvaluateRules(r.RevisionID, pages)
	// A replacement revision re-evaluates the parent's open findings.
	if parent != "" {
		if prev := c.CurrentRevision(); prev != nil {
			now := s.clock.Now()
			present := map[string]bool{}
			for _, f := range r.Findings {
				if f.Severity == domain.SeverityBlocking {
					present[fmt.Sprintf("%s/%d", f.RuleCode, f.PageNumber)] = true
				}
			}
			for i := range prev.Findings {
				f := &prev.Findings[i]
				k := fmt.Sprintf("%s/%d", f.RuleCode, f.PageNumber)
				if f.Severity == domain.SeverityBlocking && f.Status == domain.FindingOpen && !present[k] {
					f.Status = domain.FindingResolved
					f.ResolvedBy = actor
					f.ResolvedAt = &now
					f.ResolutionNote = "跨修订复验已通过"
				}
			}
			prev.Stats = c.FindingStats()
		}
	}
	c.Revisions = append(c.Revisions, r)
	c.CurrentRevisionID = r.RevisionID
	if current := c.CurrentRevision(); current != nil {
		current.Stats = c.FindingStats()
	}
	c.Version++
	c.UpdatedAt = s.clock.Now()
	if c.BlockingOpen() == 0 {
		c.Status = domain.StatusReviewPending
	} else {
		c.Status = domain.StatusReturned
	}
	e := s.event("RevisionSubmitted", c, actor, role, key, r)
	if err = s.repo.Save(c, e); err != nil {
		return nil, err
	}
	if key != "" {
		s.repo.SetCommand(key, c)
	}
	return c, nil
}

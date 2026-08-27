package application

import (
	"archive-release/internal/domain"
	"archive-release/internal/persistence"
	"testing"
)

func TestWorkflowAndIdempotency(t *testing.T) {
	s := New(persistence.NewMemory(), "key")
	c, e := s.CreateCase("标题", "单元", "创建者", "a", domain.RoleArchivist, "k1")
	if e != nil {
		t.Fatal(e)
	}
	same, e := s.CreateCase("其他", "其他", "其他", "a", domain.RoleArchivist, "k1")
	if e != nil || same.CaseID != c.CaseID {
		t.Fatal("幂等失败")
	}
	c, e = s.SubmitRevision(c.CaseID, "a", domain.RoleArchivist, c.Version, "k2", []domain.PageMeasurement{{PageNumber: 1, Clarity: 95, Skew: 0, CropRatio: 1, ColorTarget: true}}, map[string]string{"collection": "x"}, "")
	if e != nil {
		t.Fatal(e)
	}
	if c.Status != domain.StatusReviewPending {
		t.Fatalf("状态错误 %s", c.Status)
	}
	c, e = s.Review(c.CaseID, "r", domain.RoleReviewer, c.Version, "k3", true, "")
	if e != nil {
		t.Fatal(e)
	}
	_, _, _, e = s.Freeze(c.CaseID, "r", domain.RoleReviewer, c.Version, "k4")
	if e != nil {
		t.Fatal(e)
	}
}

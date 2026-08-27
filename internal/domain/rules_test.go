package domain

import "testing"

func TestEvaluateRulesDeterministic(t *testing.T) {
	p := []PageMeasurement{{PageNumber: 1, Clarity: 75, Skew: 4, CropRatio: .9}}
	a := EvaluateRules("rev", p)
	b := EvaluateRules("rev", p)
	if len(a) != len(b) || a[0].FindingID != b[0].FindingID {
		t.Fatal("规则结果不稳定")
	}
	if a[0].Severity != SeverityBlocking {
		t.Fatalf("期望阻断, 得到 %s", a[0].Severity)
	}
}
func TestValidateRevision(t *testing.T) {
	if err := ValidateRevision(ScanRevision{CaseID: "c", SubmittedBy: "u", Pages: []PageMeasurement{{PageNumber: 0}}}); err == nil {
		t.Fatal("应拒绝无效页码")
	}
}

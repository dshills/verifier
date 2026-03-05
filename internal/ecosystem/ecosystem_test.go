package ecosystem

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dshills/verifier/internal/domain"
)

func TestLoadSpecCritic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "speccritic.json")
	data := `{"issues": [{"id": "SC-1", "severity": "high", "title": "Missing constraint", "anchor": "REQ-001"}]}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	report, err := LoadSpecCritic(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Issues) != 1 {
		t.Fatalf("issues = %d, want 1", len(report.Issues))
	}
	if report.Issues[0].Severity != "high" {
		t.Errorf("severity = %q, want high", report.Issues[0].Severity)
	}
}

func TestLoadSpecCriticInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{invalid}"), 0644); err != nil {
		t.Fatal(err)
	}

	report, err := LoadSpecCritic(path)
	if err != nil {
		t.Fatal("should not return error for invalid JSON")
	}
	if report != nil {
		t.Error("should return nil for invalid JSON")
	}
}

func TestLoadPrism(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prism.json")
	data := `{"findings": [{"id": "P-1", "severity": "medium", "file": "main.go", "line_start": 10, "line_end": 20, "message": "Error not checked"}]}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	report, err := LoadPrism(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 1 {
		t.Fatalf("findings = %d, want 1", len(report.Findings))
	}
}

func TestLoadRealityCheck(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rc.json")
	data := `{"deltas": [{"id": "D-1", "kind": "added", "description": "New endpoint", "spec_ref": "REQ-001", "code_ref": "api.go"}]}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	report, err := LoadRealityCheck(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Deltas) != 1 {
		t.Fatalf("deltas = %d, want 1", len(report.Deltas))
	}
}

func TestRealityCheckUnknownKind(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rc.json")
	data := `{"deltas": [{"id": "D-1", "kind": "unknown_kind", "description": "test"}]}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	report, err := LoadRealityCheck(path)
	if err != nil {
		t.Fatal(err)
	}
	if report.Deltas[0].Kind != "changed" {
		t.Errorf("kind = %q, want 'changed'", report.Deltas[0].Kind)
	}
}

func TestNormalizeSeverity(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"critical", "critical"},
		{"high", "high"},
		{"medium", "medium"},
		{"low", "low"},
		{"error", "low"},
		{"warning", "low"},
		{"", "low"},
	}
	for _, tt := range tests {
		got := normalizeSeverity(tt.input)
		if got != tt.want {
			t.Errorf("normalizeSeverity(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestApplySpecCriticBoost(t *testing.T) {
	recs := []domain.Recommendation{
		{
			Severity: domain.SeverityMedium,
			Covers:   domain.Covers{Requirements: []string{"REQ-001"}},
		},
	}
	sc := &SpecCriticReport{
		Issues: []SpecCriticIssue{
			{Anchor: "REQ-001", Severity: "high"},
		},
	}

	ApplySpecCritic(&recs, sc)
	if recs[0].Severity != domain.SeverityHigh {
		t.Errorf("severity = %q, want high (boosted from medium)", recs[0].Severity)
	}
}

func TestApplyPlanCritic(t *testing.T) {
	var recs []domain.Recommendation
	pc := &PlanCriticReport{
		Issues: []PlanCriticIssue{
			{ID: "PC-1", Severity: "high", Title: "Missing error handling", Component: "auth"},
		},
	}

	ApplyPlanCritic(&recs, pc)
	if len(recs) != 1 {
		t.Fatalf("recs = %d, want 1", len(recs))
	}
	if recs[0].Target.Name != "auth" {
		t.Errorf("target = %q, want auth", recs[0].Target.Name)
	}
}

func TestApplyRealityCheck(t *testing.T) {
	var recs []domain.Recommendation
	rc := &RealityCheckReport{
		Deltas: []RealityCheckDelta{
			{ID: "D-1", Kind: "added", Description: "New endpoint", CodeRef: "api.go"},
		},
	}

	ApplyRealityCheck(&recs, rc)
	if len(recs) != 1 {
		t.Fatalf("recs = %d, want 1", len(recs))
	}
	if recs[0].Covers.Requirements[0] != "RC-D-1" {
		t.Errorf("requirement = %q, want RC-D-1", recs[0].Covers.Requirements[0])
	}
}

func TestApplyPrism(t *testing.T) {
	var recs []domain.Recommendation
	pr := &PrismReport{
		Findings: []PrismFinding{
			{ID: "P-1", Severity: "high", File: "main.go", LineStart: 10, Message: "unchecked error"},
		},
	}

	ApplyPrism(&recs, pr)
	if len(recs) != 1 {
		t.Fatalf("recs = %d, want 1", len(recs))
	}
	if recs[0].Target.File != "main.go" {
		t.Errorf("file = %q, want main.go", recs[0].Target.File)
	}
}

func TestBoostSeverity(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"low", "medium"},
		{"medium", "high"},
		{"high", "critical"},
		{"critical", "critical"},
	}
	for _, tt := range tests {
		got := boostSeverity(tt.input)
		if got != tt.want {
			t.Errorf("boostSeverity(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

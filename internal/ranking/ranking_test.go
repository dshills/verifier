package ranking

import (
	"testing"

	"github.com/dshills/verifier/internal/domain"
)

func TestComputeTESTRECHash(t *testing.T) {
	hash := computeTESTRECHash("REQ-001", "CreateUser", "unit")
	if len(hash) != 8 {
		t.Errorf("hash length = %d, want 8", len(hash))
	}

	// Deterministic
	hash2 := computeTESTRECHash("REQ-001", "CreateUser", "unit")
	if hash != hash2 {
		t.Error("hash should be deterministic")
	}

	// Different inputs produce different hashes
	hash3 := computeTESTRECHash("REQ-002", "CreateUser", "unit")
	if hash == hash3 {
		t.Error("different inputs should produce different hashes")
	}
}

func TestAssignIDs(t *testing.T) {
	recs := []domain.Recommendation{
		{
			Target:   domain.Target{Name: "CreateUser"},
			Category: "unit",
			Covers:   domain.Covers{Requirements: []string{"REQ-001"}},
		},
		{
			Target:   domain.Target{Name: "DeleteUser"},
			Category: "unit",
			Covers:   domain.Covers{Requirements: []string{"REQ-002"}},
		},
	}

	AssignIDs(recs)

	if recs[0].ID == "" || recs[1].ID == "" {
		t.Error("IDs should be assigned")
	}
	if recs[0].ID == recs[1].ID {
		t.Error("IDs should be unique")
	}
	if recs[0].ID[:8] != "TESTREC-" {
		t.Errorf("ID = %q, want TESTREC- prefix", recs[0].ID)
	}
}

func TestIDCollision(t *testing.T) {
	// Same inputs should get collision suffix
	recs := []domain.Recommendation{
		{Target: domain.Target{Name: "CreateUser"}, Category: "unit", Covers: domain.Covers{Requirements: []string{"REQ-001"}}},
		{Target: domain.Target{Name: "CreateUser"}, Category: "unit", Covers: domain.Covers{Requirements: []string{"REQ-001"}}},
	}

	AssignIDs(recs)

	if recs[0].ID == recs[1].ID {
		t.Error("colliding IDs should get suffix")
	}
}

func TestSortRecommendations(t *testing.T) {
	recs := []domain.Recommendation{
		{ID: "B", Severity: domain.SeverityLow, Confidence: 0.5},
		{ID: "A", Severity: domain.SeverityCritical, Confidence: 0.8},
		{ID: "C", Severity: domain.SeverityHigh, Confidence: 0.9},
	}

	SortRecommendations(recs)

	if recs[0].Severity != domain.SeverityCritical {
		t.Errorf("first should be critical, got %q", recs[0].Severity)
	}
	if recs[1].Severity != domain.SeverityHigh {
		t.Errorf("second should be high, got %q", recs[1].Severity)
	}
}

func TestComputeRiskScore(t *testing.T) {
	recs := []domain.Recommendation{
		{Severity: domain.SeverityCritical},
		{Severity: domain.SeverityHigh},
		{Severity: domain.SeverityMedium},
		{Severity: domain.SeverityLow},
	}

	score := ComputeRiskScore(recs)
	// 10 + 5 + 2 + 1 = 18
	if score != 18 {
		t.Errorf("risk score = %d, want 18", score)
	}
}

func TestComputeRiskScoreCapped(t *testing.T) {
	var recs []domain.Recommendation
	for range 20 {
		recs = append(recs, domain.Recommendation{Severity: domain.SeverityCritical})
	}

	score := ComputeRiskScore(recs)
	if score != 100 {
		t.Errorf("risk score = %d, want 100 (capped)", score)
	}
}

func TestTruncate(t *testing.T) {
	recs := make([]domain.Recommendation, 10)
	truncated, wasTruncated := Truncate(recs, 5)
	if !wasTruncated {
		t.Error("expected truncation")
	}
	if len(truncated) != 5 {
		t.Errorf("len = %d, want 5", len(truncated))
	}
}

func TestTruncateNoLimit(t *testing.T) {
	recs := make([]domain.Recommendation, 10)
	truncated, wasTruncated := Truncate(recs, 0)
	if wasTruncated {
		t.Error("should not truncate with limit 0")
	}
	if len(truncated) != 10 {
		t.Errorf("len = %d, want 10", len(truncated))
	}
}

func TestCheckFailOn(t *testing.T) {
	recs := []domain.Recommendation{
		{Severity: domain.SeverityHigh},
		{Severity: domain.SeverityLow},
	}

	if !CheckFailOn(recs, "high") {
		t.Error("should fail on high")
	}
	if CheckFailOn(recs, "critical") {
		t.Error("should not fail on critical")
	}
	if CheckFailOn(recs, "none") {
		t.Error("should not fail on none")
	}
}

func TestAssignSeveritySecurity(t *testing.T) {
	recs := []domain.Recommendation{
		{Covers: domain.Covers{Requirements: []string{"REQ-001"}}},
	}
	reqs := &domain.RequirementSet{
		Requirements: []domain.Requirement{
			{ID: "REQ-001", Text: "authentication tokens must be secure"},
		},
	}

	AssignSeverity(recs, reqs)
	if recs[0].Severity != domain.SeverityCritical {
		t.Errorf("severity = %q, want critical", recs[0].Severity)
	}
}

func TestAssignSeverityConcurrency(t *testing.T) {
	recs := []domain.Recommendation{
		{Covers: domain.Covers{Risks: []string{domain.RiskConcurrency}}},
	}

	AssignSeverity(recs, nil)
	if recs[0].Severity != domain.SeverityCritical {
		t.Errorf("severity = %q, want critical", recs[0].Severity)
	}
}

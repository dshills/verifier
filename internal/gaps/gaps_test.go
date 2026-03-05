package gaps

import (
	"testing"

	"github.com/dshills/verifier/internal/domain"
)

func TestAnnotateExistingTests(t *testing.T) {
	recs := []domain.Recommendation{
		{
			Target: domain.Target{Name: "CreateUser", File: "/app/user.go"},
			Covers: domain.Covers{Risks: []string{domain.RiskErrorPath}},
		},
	}

	tests := &domain.TestInventory{
		Tests: []domain.TestInfo{
			{FuncName: "TestCreateUser", File: "/app/user_test.go", Package: "user"},
		},
	}

	AnnotateExistingTests(recs, tests)

	if len(recs[0].ExistingTests) == 0 {
		t.Error("expected existing test annotation")
	}
}

func TestDegradedFindings(t *testing.T) {
	arts := &domain.Artifacts{
		RepoGraph: &domain.RepoGraph{
			Packages: []domain.PackageInfo{
				{Name: "handler", Dir: "/app/handler", GoFiles: []string{"handler.go"}, HasTests: false},
				{Name: "util", Dir: "/app/util", GoFiles: []string{"util.go"}, HasTests: true},
			},
		},
		TestInventory: &domain.TestInventory{
			Tests: []domain.TestInfo{
				{FuncName: "TestHelper", Package: "util"},
			},
		},
	}

	recs := DegradedFindings(arts)

	// Should have at least a zero-test package finding for "handler"
	found := false
	for _, rec := range recs {
		if rec.Target.Name == "handler" {
			found = true
		}
	}
	if !found {
		t.Error("expected finding for untested handler package")
	}
}

func TestDetectGapMissingNegative(t *testing.T) {
	tests := []domain.TestInfo{
		{FuncName: "TestCreateUser", Package: "user"},
	}
	rec := &domain.Recommendation{
		Target: domain.Target{Name: "CreateUser"},
		Covers: domain.Covers{Risks: []string{domain.RiskErrorPath}},
	}

	gap := detectGap("TestCreateUser", tests, rec)
	if gap != "missing_negative_paths" {
		t.Errorf("gap = %q, want missing_negative_paths", gap)
	}
}

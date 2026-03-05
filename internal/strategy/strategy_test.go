package strategy

import (
	"testing"

	"github.com/dshills/verifier/internal/domain"
)

func TestAssignCategoryHTTPHandler(t *testing.T) {
	cat := assignCategory([]string{domain.RiskHTTPHandler}, nil, false)
	if cat != domain.CategoryIntegration {
		t.Errorf("got %q, want integration", cat)
	}

	cat = assignCategory([]string{domain.RiskHTTPHandler}, nil, true)
	if cat != domain.CategoryContract {
		t.Errorf("got %q, want contract", cat)
	}
}

func TestAssignCategoryDB(t *testing.T) {
	cat := assignCategory([]string{domain.RiskDBQuery}, nil, false)
	if cat != domain.CategoryIntegration {
		t.Errorf("got %q, want integration", cat)
	}
}

func TestAssignCategoryConcurrency(t *testing.T) {
	cat := assignCategory([]string{domain.RiskConcurrency}, nil, false)
	if cat != domain.CategoryConcurrency {
		t.Errorf("got %q, want concurrency", cat)
	}
}

func TestAssignCategoryFuzz(t *testing.T) {
	cat := assignCategory([]string{domain.RiskErrorPath, domain.RiskInputValidation}, nil, false)
	if cat != domain.CategoryFuzz {
		t.Errorf("got %q, want fuzz", cat)
	}
}

func TestAssignCategorySecurity(t *testing.T) {
	req := &domain.Requirement{Text: "authentication tokens must be validated"}
	cat := assignCategory(nil, req, false)
	if cat != domain.CategorySecurity {
		t.Errorf("got %q, want security", cat)
	}
}

func TestAssignCategoryDefault(t *testing.T) {
	cat := assignCategory(nil, nil, false)
	if cat != domain.CategoryUnit {
		t.Errorf("got %q, want unit", cat)
	}
}

func TestAssignStrategies(t *testing.T) {
	arts := &domain.Artifacts{
		CoverageMap: &domain.CoverageMap{
			Mappings: []domain.CoverageMapping{
				{RequirementID: "REQ-001", Symbols: []string{"CreateUser"}, Confidence: 0.8},
			},
		},
		RequirementSet: &domain.RequirementSet{
			Requirements: []domain.Requirement{
				{ID: "REQ-001", Text: "Create user accounts"},
			},
		},
		SymbolIndex: &domain.SymbolIndex{
			Symbols: []domain.Symbol{
				{Name: "CreateUser", Kind: "function", File: "user.go", Exported: true},
			},
		},
		RiskSignals: &domain.RiskSignals{
			Signals: []domain.RiskSignal{
				{Symbol: "CreateUser", Risks: []string{domain.RiskErrorPath}},
			},
		},
	}

	recs := AssignStrategies(arts)
	if len(recs) == 0 {
		t.Fatal("expected at least one recommendation")
	}
	if recs[0].Target.Name != "CreateUser" {
		t.Errorf("target = %q, want CreateUser", recs[0].Target.Name)
	}
}

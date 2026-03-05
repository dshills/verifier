package mapping

import (
	"testing"

	"github.com/dshills/verifier/internal/domain"
)

func TestMapRequirements(t *testing.T) {
	reqs := &domain.RequirementSet{
		Requirements: []domain.Requirement{
			{ID: "REQ-001", Text: "Create user accounts"},
			{ID: "REQ-002", Text: "Something completely unrelated xyz123"},
		},
	}
	idx := &domain.SymbolIndex{
		Symbols: []domain.Symbol{
			{Name: "CreateUser", Package: "user", Exported: true},
			{Name: "DeleteUser", Package: "user", Exported: true},
		},
	}

	cm, unmapped := MapRequirements(reqs, idx)

	if len(cm.Mappings) == 0 {
		t.Fatal("expected at least one mapping")
	}

	// REQ-001 should map to CreateUser
	found := false
	for _, m := range cm.Mappings {
		if m.RequirementID == "REQ-001" {
			found = true
			hasCreate := false
			for _, s := range m.Symbols {
				if s == "CreateUser" {
					hasCreate = true
				}
			}
			if !hasCreate {
				t.Error("REQ-001 should map to CreateUser")
			}
		}
	}
	if !found {
		t.Error("REQ-001 not found in mappings")
	}

	// REQ-002 should be unmapped
	hasUnmapped := false
	for _, id := range unmapped.IDs {
		if id == "REQ-002" {
			hasUnmapped = true
		}
	}
	if !hasUnmapped {
		t.Error("REQ-002 should be unmapped")
	}
}

func TestJaccardSimilarity(t *testing.T) {
	tests := []struct {
		a, b map[string]bool
		want float64
	}{
		{map[string]bool{"a": true, "b": true}, map[string]bool{"a": true, "b": true}, 1.0},
		{map[string]bool{"a": true, "b": true}, map[string]bool{"c": true, "d": true}, 0.0},
		{map[string]bool{"a": true, "b": true}, map[string]bool{"a": true, "c": true}, 1.0 / 3.0},
		{map[string]bool{}, map[string]bool{}, 0.0},
	}
	for _, tt := range tests {
		got := jaccardSimilarity(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("jaccard(%v, %v) = %f, want %f", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestTokenize(t *testing.T) {
	tokens := tokenize("The system must create user accounts")
	if tokens["the"] {
		t.Error("stop word 'the' should be removed")
	}
	if !tokens["create"] {
		t.Error("'create' should be a token")
	}
	if !tokens["user"] {
		t.Error("'user' should be a token")
	}
}

func TestTokenizeSymbol(t *testing.T) {
	tokens := tokenizeSymbol("CreateUserAccount")
	if !tokens["create"] {
		t.Error("'create' should be a token")
	}
	if !tokens["user"] {
		t.Error("'user' should be a token")
	}
	if !tokens["account"] {
		t.Error("'account' should be a token")
	}
}

func TestConfidenceLevels(t *testing.T) {
	req := domain.Requirement{Text: "user management", HeadingContext: "User Requirements"}
	symSamePkg := domain.Symbol{Name: "CreateUser", Package: "user"}

	// High Jaccard + same package mention
	conf := computeConfidence(0.9, req, symSamePkg)
	if conf != 1.0 {
		t.Errorf("expected 1.0, got %f", conf)
	}

	// Moderate Jaccard + same package
	conf = computeConfidence(0.4, req, symSamePkg)
	if conf != 0.7 {
		t.Errorf("expected 0.7, got %f", conf)
	}

	// Moderate Jaccard + different package
	symOther := domain.Symbol{Name: "CreateUser", Package: "other"}
	conf = computeConfidence(0.4, req, symOther)
	if conf != 0.5 {
		t.Errorf("expected 0.5, got %f", conf)
	}
}

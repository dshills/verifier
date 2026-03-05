package strategy

import (
	"strings"

	"github.com/dshills/verifier/internal/domain"
)

// AssignStrategies assigns test categories and proposals for each coverage mapping.
func AssignStrategies(arts *domain.Artifacts) []domain.Recommendation {
	riskIndex := buildRiskIndex(arts.RiskSignals)
	symIndex := buildSymIndex(arts.SymbolIndex)

	var recs []domain.Recommendation

	for _, mapping := range arts.CoverageMap.Mappings {
		req := findRequirement(arts.RequirementSet, mapping.RequirementID)

		for _, symName := range mapping.Symbols {
			risks := riskIndex[symName]
			sym := symIndex[symName]
			category := assignCategory(risks, req, arts.HasOpenAPI)

			rec := domain.Recommendation{
				Confidence: mapping.Confidence,
				Category:   category,
				Target: domain.Target{
					Kind: targetKind(sym),
					Name: symName,
					File: sym.File,
				},
				Covers: domain.Covers{
					Requirements: []string{mapping.RequirementID},
					Risks:        risks,
				},
				Proposal: buildProposal(symName, category),
			}

			if sym.LineStart > 0 {
				rec.Target.LineStart = sym.LineStart
				rec.Target.LineEnd = sym.LineEnd
			}

			recs = append(recs, rec)
		}
	}

	return recs
}

func assignCategory(risks []string, req *domain.Requirement, hasOpenAPI bool) string {
	riskSet := make(map[string]bool)
	for _, r := range risks {
		riskSet[r] = true
	}

	if riskSet[domain.RiskHTTPHandler] {
		if hasOpenAPI {
			return domain.CategoryContract
		}
		return domain.CategoryIntegration
	}
	if riskSet[domain.RiskDBQuery] {
		return domain.CategoryIntegration
	}
	if riskSet[domain.RiskConcurrency] {
		return domain.CategoryConcurrency
	}
	if riskSet[domain.RiskErrorPath] && riskSet[domain.RiskInputValidation] {
		return domain.CategoryFuzz
	}
	if req != nil {
		text := strings.ToLower(req.Text)
		if containsAny(text, "performance", "latency", "throughput") && len(req.AcceptanceCriteria) > 0 {
			return domain.CategoryPerf
		}
		if containsAny(text, "auth", "security", "injection", "token") {
			return domain.CategorySecurity
		}
	}
	if riskSet[domain.RiskInputValidation] {
		return domain.CategoryProperty
	}
	return domain.CategoryUnit
}

func buildProposal(symName, category string) domain.Proposal {
	switch category {
	case domain.CategoryUnit:
		return domain.Proposal{
			Title:    "Table-driven unit test for " + symName,
			Approach: "table-driven",
			Assertions: []string{
				"Verify expected output for valid inputs",
				"Verify error handling for invalid inputs",
			},
		}
	case domain.CategoryIntegration:
		return domain.Proposal{
			Title:    "Integration test for " + symName,
			Approach: "subtests with setup/teardown",
			Assertions: []string{
				"Verify end-to-end behavior",
				"Verify error propagation",
			},
		}
	case domain.CategoryContract:
		return domain.Proposal{
			Title:    "Contract test for " + symName,
			Approach: "HTTP request/response validation against schema",
			Assertions: []string{
				"Verify response status codes",
				"Verify response body schema",
			},
		}
	case domain.CategoryConcurrency:
		return domain.Proposal{
			Title:    "Concurrency test for " + symName,
			Approach: "parallel execution with race detection",
			Assertions: []string{
				"Verify thread safety under concurrent access",
				"Run with -race flag",
			},
		}
	case domain.CategoryFuzz:
		return domain.Proposal{
			Title:    "Fuzz test for " + symName,
			Approach: "go test fuzz with corpus",
			Assertions: []string{
				"Verify no panics on random input",
				"Verify error returns for malformed input",
			},
		}
	case domain.CategorySecurity:
		return domain.Proposal{
			Title:    "Security test for " + symName,
			Approach: "boundary validation",
			Assertions: []string{
				"Verify authentication/authorization checks",
				"Verify input sanitization",
			},
		}
	case domain.CategoryProperty:
		return domain.Proposal{
			Title:    "Property test for " + symName,
			Approach: "invariant checking",
			Assertions: []string{
				"Verify input validation invariants hold",
			},
		}
	default:
		return domain.Proposal{
			Title:    "Test for " + symName,
			Approach: "table-driven",
		}
	}
}

func targetKind(sym domain.Symbol) string {
	switch sym.Kind {
	case "method":
		return domain.TargetMethod
	case "function":
		return domain.TargetFunction
	default:
		return domain.TargetFunction
	}
}

func buildRiskIndex(risks *domain.RiskSignals) map[string][]string {
	idx := make(map[string][]string)
	if risks == nil {
		return idx
	}
	for _, sig := range risks.Signals {
		idx[sig.Symbol] = sig.Risks
	}
	return idx
}

func buildSymIndex(idx *domain.SymbolIndex) map[string]domain.Symbol {
	m := make(map[string]domain.Symbol)
	if idx == nil {
		return m
	}
	for _, sym := range idx.Symbols {
		m[sym.Name] = sym
	}
	return m
}

func findRequirement(reqs *domain.RequirementSet, id string) *domain.Requirement {
	if reqs == nil {
		return nil
	}
	for i := range reqs.Requirements {
		if reqs.Requirements[i].ID == id {
			return &reqs.Requirements[i]
		}
	}
	return nil
}

func containsAny(text string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

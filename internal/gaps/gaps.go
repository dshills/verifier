package gaps

import (
	"strings"

	"github.com/dshills/verifier/internal/domain"
)

// AnnotateExistingTests checks recommendations against the test inventory
// and adds existing test references and gap annotations.
func AnnotateExistingTests(recs []domain.Recommendation, tests *domain.TestInventory) {
	testsByPkg := make(map[string][]domain.TestInfo)
	for _, t := range tests.Tests {
		testsByPkg[t.Package] = append(testsByPkg[t.Package], t)
	}

	for i := range recs {
		rec := &recs[i]
		targetName := rec.Target.Name

		// Find package from file path
		pkgTests := findPackageTests(rec.Target.File, testsByPkg)

		for _, t := range pkgTests {
			if isTestForSymbol(t.FuncName, targetName) {
				existing := domain.ExistingTest{
					File: t.File,
					Name: t.FuncName,
				}

				gap := detectGap(t.FuncName, pkgTests, rec)
				if gap != "" {
					existing.Gap = gap
				}

				rec.ExistingTests = append(rec.ExistingTests, existing)
			}
		}
	}
}

func isTestForSymbol(testName, symbolName string) bool {
	// TestCreateUser matches CreateUser
	if strings.Contains(testName, symbolName) {
		return true
	}
	// Test_CreateUser matches CreateUser
	cleaned := strings.ReplaceAll(testName, "_", "")
	return strings.Contains(cleaned, symbolName)
}

func detectGap(_ string, allTests []domain.TestInfo, rec *domain.Recommendation) string {
	hasNegative := false
	for _, t := range allTests {
		if !isTestForSymbol(t.FuncName, rec.Target.Name) {
			continue
		}
		name := strings.ToLower(t.FuncName)
		if strings.Contains(name, "error") || strings.Contains(name, "invalid") ||
			strings.Contains(name, "fail") || strings.Contains(name, "bad") {
			hasNegative = true
		}
	}

	risks := make(map[string]bool)
	for _, r := range rec.Covers.Risks {
		risks[r] = true
	}

	if !hasNegative {
		return "missing_negative_paths"
	}
	if risks[domain.RiskBoundary] || risks[domain.RiskHTTPHandler] || risks[domain.RiskDBQuery] {
		hasIntegration := false
		for _, t := range allTests {
			if strings.Contains(strings.ToLower(t.FuncName), "integration") {
				hasIntegration = true
			}
		}
		if !hasIntegration {
			return "missing_boundary_tests"
		}
	}
	if risks[domain.RiskConcurrency] {
		return "missing_race_tests"
	}
	return ""
}

func findPackageTests(file string, testsByPkg map[string][]domain.TestInfo) []domain.TestInfo {
	// Try each package's tests
	for _, tests := range testsByPkg {
		if len(tests) > 0 {
			// Check if test files are in same directory
			for _, t := range tests {
				if sameDir(t.File, file) {
					return tests
				}
			}
		}
	}
	return nil
}

func sameDir(a, b string) bool {
	aDir := dirOf(a)
	bDir := dirOf(b)
	return aDir == bDir
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}

// DegradedFindings generates recommendations from code signals only (no spec/plan).
func DegradedFindings(arts *domain.Artifacts) []domain.Recommendation {
	var recs []domain.Recommendation

	if arts.RepoGraph == nil {
		return recs
	}

	testedPkgs := make(map[string]bool)
	if arts.TestInventory != nil {
		for _, t := range arts.TestInventory.Tests {
			testedPkgs[t.Package] = true
		}
	}

	// Zero-test packages
	for _, pkg := range arts.RepoGraph.Packages {
		if len(pkg.GoFiles) > 0 && !pkg.HasTests {
			recs = append(recs, domain.Recommendation{
				Category: domain.CategoryUnit,
				Target: domain.Target{
					Kind: domain.TargetPackage,
					Name: pkg.Name,
					File: pkg.Dir,
				},
				Proposal: domain.Proposal{
					Title:    "Add tests for package " + pkg.Name,
					Approach: "Package has no test files",
				},
			})
		}
	}

	// Untested exported symbols (if we have symbol index)
	if arts.SymbolIndex != nil && arts.TestInventory != nil {
		testedSyms := make(map[string]bool)
		for _, t := range arts.TestInventory.Tests {
			testedSyms[t.FuncName] = true
		}

		for _, sym := range arts.SymbolIndex.Symbols {
			if !sym.Exported || sym.Kind == "type" || sym.Kind == "interface" {
				continue
			}
			testName := "Test" + sym.Name
			if !testedSyms[testName] {
				recs = append(recs, domain.Recommendation{
					Category: domain.CategoryUnit,
					Target: domain.Target{
						Kind:      targetKind(sym),
						Name:      sym.Name,
						File:      sym.File,
						LineStart: sym.LineStart,
						LineEnd:   sym.LineEnd,
					},
					Proposal: domain.Proposal{
						Title:    "Add test for " + sym.Name,
						Approach: "Exported symbol has no corresponding test",
					},
				})
			}
		}
	}

	return recs
}

func targetKind(sym domain.Symbol) string {
	switch sym.Kind {
	case "method":
		return domain.TargetMethod
	default:
		return domain.TargetFunction
	}
}

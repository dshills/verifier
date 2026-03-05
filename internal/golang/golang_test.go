package golang

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/dshills/verifier/internal/domain"
)

func testdataDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(file)
}

func sampleGraph() *domain.RepoGraph {
	dir := filepath.Join(testdataDir(), "testdata")
	return &domain.RepoGraph{
		Packages: []domain.PackageInfo{
			{
				Name:      "testdata",
				Dir:       dir,
				GoFiles:   []string{filepath.Join(dir, "sample.go")},
				TestFiles: []string{filepath.Join(dir, "sample_test.go")},
				HasTests:  true,
			},
		},
	}
}

func TestAnalyzePackages(t *testing.T) {
	graph := sampleGraph()
	cfg := &domain.Config{}
	idx, risks, err := AnalyzePackages(graph, cfg)
	if err != nil {
		t.Fatalf("AnalyzePackages: %v", err)
	}

	if len(idx.Symbols) == 0 {
		t.Fatal("expected symbols")
	}

	// Check for expected symbols
	symbolNames := make(map[string]bool)
	for _, s := range idx.Symbols {
		symbolNames[s.Name] = true
	}
	for _, name := range []string{"CreateUser", "HandleHealth", "QueryUsers", "ComplexFunc", "ValidateInput", "GetUser"} {
		if !symbolNames[name] {
			t.Errorf("missing symbol %q", name)
		}
	}

	// Check types
	if !symbolNames["MyInterface"] {
		t.Error("missing interface MyInterface")
	}
	if !symbolNames["UserService"] {
		t.Error("missing type UserService")
	}

	// Check risk signals
	if len(risks.Signals) == 0 {
		t.Fatal("expected risk signals")
	}

	riskMap := make(map[string][]string)
	for _, sig := range risks.Signals {
		riskMap[sig.Symbol] = sig.Risks
	}

	assertContains(t, riskMap["CreateUser"], domain.RiskErrorPath)
	assertContains(t, riskMap["HandleHealth"], domain.RiskHTTPHandler)
	assertContains(t, riskMap["QueryUsers"], domain.RiskDBQuery)
	assertContains(t, riskMap["ComplexFunc"], domain.RiskComplexity)
	assertContains(t, riskMap["ValidateInput"], domain.RiskInputValidation)
}

func TestEnrichTests(t *testing.T) {
	graph := sampleGraph()
	inv := &domain.TestInventory{}
	enrichTests(graph, inv)

	if len(inv.Tests) == 0 {
		t.Fatal("expected enriched tests")
	}

	hasSubtest := false
	for _, tt := range inv.Tests {
		if tt.IsSubtest {
			hasSubtest = true
		}
	}
	if !hasSubtest {
		t.Error("expected at least one subtest")
	}
}

func TestMethodDetection(t *testing.T) {
	graph := sampleGraph()
	cfg := &domain.Config{}
	idx, _, err := AnalyzePackages(graph, cfg)
	if err != nil {
		t.Fatal(err)
	}

	for _, sym := range idx.Symbols {
		if sym.Name == "GetUser" {
			if sym.Kind != "method" {
				t.Errorf("GetUser kind = %q, want method", sym.Kind)
			}
			if sym.ReceiverType != "*UserService" {
				t.Errorf("GetUser receiver = %q, want *UserService", sym.ReceiverType)
			}
			return
		}
	}
	t.Error("GetUser not found")
}

func assertContains(t *testing.T, risks []string, want string) {
	t.Helper()
	for _, r := range risks {
		if r == want {
			return
		}
	}
	t.Errorf("risks %v does not contain %q", risks, want)
}

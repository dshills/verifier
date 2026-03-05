package repo

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/dshills/verifier/internal/domain"
)

func testdataDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata", "sample")
}

func TestScanRepo(t *testing.T) {
	root := testdataDir()
	cfg := &domain.Config{Root: root}

	graph, err := ScanRepo(cfg)
	if err != nil {
		t.Fatalf("ScanRepo: %v", err)
	}

	if graph.ModulePath != "github.com/example/sample" {
		t.Errorf("module path = %q, want github.com/example/sample", graph.ModulePath)
	}

	if len(graph.Packages) == 0 {
		t.Fatal("expected packages, got 0")
	}

	// Check cmd package detected
	foundCmd := false
	foundHandler := false
	foundUtil := false
	for _, pkg := range graph.Packages {
		if pkg.IsCmd {
			foundCmd = true
		}
		if pkg.Name == "handler" {
			foundHandler = true
			if !pkg.HasTests {
				t.Error("handler package should have tests")
			}
		}
		if pkg.Name == "util" {
			foundUtil = true
		}
	}
	if !foundCmd {
		t.Error("expected cmd package to be detected")
	}
	if !foundHandler {
		t.Error("expected handler package")
	}
	if !foundUtil {
		t.Error("expected util package")
	}
}

func TestDetectBoundaries(t *testing.T) {
	root := testdataDir()
	cfg := &domain.Config{Root: root}
	graph, err := ScanRepo(cfg)
	if err != nil {
		t.Fatalf("ScanRepo: %v", err)
	}

	bm := DetectBoundaries(graph)

	if len(bm.HTTP) == 0 {
		t.Error("expected HTTP boundaries from handler package")
	}

	if len(bm.FS) == 0 {
		t.Error("expected FS boundaries from util package")
	}
}

func TestCollectTests(t *testing.T) {
	root := testdataDir()
	cfg := &domain.Config{Root: root}
	graph, err := ScanRepo(cfg)
	if err != nil {
		t.Fatalf("ScanRepo: %v", err)
	}

	inv := CollectTests(graph)

	if len(inv.Tests) == 0 {
		t.Fatal("expected at least 1 test")
	}

	found := false
	for _, tt := range inv.Tests {
		if tt.FuncName == "TestHealth" {
			found = true
		}
	}
	if !found {
		t.Error("expected TestHealth in test inventory")
	}
}

func TestExcludeGlob(t *testing.T) {
	root := testdataDir()
	cfg := &domain.Config{
		Root:    root,
		Exclude: []string{"**/cmd/**"},
	}
	graph, err := ScanRepo(cfg)
	if err != nil {
		t.Fatalf("ScanRepo: %v", err)
	}

	for _, pkg := range graph.Packages {
		if pkg.IsCmd {
			t.Error("cmd package should have been excluded")
		}
	}
}

func TestExcludeTakesPrecedence(t *testing.T) {
	root := testdataDir()
	cfg := &domain.Config{
		Root:    root,
		Include: []string{"**/*.go"},
		Exclude: []string{"**/util/**"},
	}
	graph, err := ScanRepo(cfg)
	if err != nil {
		t.Fatalf("ScanRepo: %v", err)
	}

	for _, pkg := range graph.Packages {
		if pkg.Name == "util" {
			t.Error("util package should have been excluded despite include")
		}
	}
}

package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dshills/verifier/internal/domain"
)

func TestPlanCriticalOnly(t *testing.T) {
	recs := []domain.Recommendation{
		{ID: "T-001", Severity: domain.SeverityCritical, Target: domain.Target{Name: "Foo", File: "/app/foo.go"}, Category: "unit"},
		{ID: "T-002", Severity: domain.SeverityLow, Target: domain.Target{Name: "Bar", File: "/app/bar.go"}, Category: "unit"},
	}

	actions, selected := Plan(recs, 5)
	if len(selected) != 1 {
		t.Errorf("selected = %d, want 1 (critical only)", len(selected))
	}
	if len(actions) == 0 {
		t.Fatal("expected actions")
	}
}

func TestPlanNoCritical(t *testing.T) {
	recs := []domain.Recommendation{
		{ID: "T-001", Severity: domain.SeverityLow, Target: domain.Target{Name: "Foo", File: "/app/foo.go"}},
	}

	actions, selected := Plan(recs, 5)
	if len(actions) != 0 {
		t.Error("expected no actions when no critical and limit > 0")
	}
	if len(selected) != 0 {
		t.Error("expected no selected")
	}
}

func TestPlanUnlimited(t *testing.T) {
	recs := []domain.Recommendation{
		{ID: "T-001", Severity: domain.SeverityLow, Target: domain.Target{Name: "Foo", File: "/app/foo.go"}, Category: "unit"},
		{ID: "T-002", Severity: domain.SeverityHigh, Target: domain.Target{Name: "Bar", File: "/app/bar.go"}, Category: "unit"},
	}

	_, selected := Plan(recs, 0)
	if len(selected) != 2 {
		t.Errorf("selected = %d, want 2 (unlimited)", len(selected))
	}
}

func TestGenerateUnitTest(t *testing.T) {
	rec := &domain.Recommendation{
		ID:       "TESTREC-ABCD1234",
		Category: domain.CategoryUnit,
		Target:   domain.Target{Name: "CreateUser"},
		Proposal: domain.Proposal{Title: "Unit test for CreateUser"},
	}

	content := GenerateTest(rec, "std")
	if !strings.Contains(content, "TESTREC-ABCD1234") {
		t.Error("expected TESTREC ID in output")
	}
	if !strings.Contains(content, "TestCreateUser") {
		t.Error("expected TestCreateUser function")
	}
	if !strings.Contains(content, "tests :=") {
		t.Error("expected table-driven structure")
	}
}

func TestGenerateFuzzTest(t *testing.T) {
	rec := &domain.Recommendation{
		ID:       "TESTREC-FUZZ0001",
		Category: domain.CategoryFuzz,
		Target:   domain.Target{Name: "ParseInput"},
		Proposal: domain.Proposal{Title: "Fuzz test for ParseInput"},
	}

	content := GenerateTest(rec, "std")
	if !strings.Contains(content, "FuzzParseInput") {
		t.Error("expected FuzzParseInput function")
	}
}

func TestExecuteDryRun(t *testing.T) {
	recs := []domain.Recommendation{
		{ID: "T-001", Category: domain.CategoryUnit, Target: domain.Target{Name: "Foo", File: "/tmp/foo.go"}},
	}
	actions := []Action{
		{Path: "/tmp/foo_test.go", Action: "create_file", TestrecID: "T-001"},
	}

	err := Execute(actions, recs, "std", true)
	if err != nil {
		t.Fatal(err)
	}
	if actions[0].Status != "dry_run" {
		t.Errorf("status = %q, want dry_run", actions[0].Status)
	}
}

func TestExecuteCreateFile(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "foo_test.go")

	recs := []domain.Recommendation{
		{ID: "T-001", Category: domain.CategoryUnit, Target: domain.Target{Name: "Foo", File: filepath.Join(dir, "foo.go")}},
	}
	actions := []Action{
		{Path: testFile, Action: "create_file", TestrecID: "T-001"},
	}

	err := Execute(actions, recs, "std", false)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "TestFoo") {
		t.Error("expected TestFoo in generated file")
	}
}

func TestExecuteAppendWithBackup(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "foo_test.go")
	original := "package foo\n\nimport \"testing\"\n\nfunc TestExisting(t *testing.T) {}\n"
	if err := os.WriteFile(testFile, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	recs := []domain.Recommendation{
		{ID: "T-001", Category: domain.CategoryUnit, Target: domain.Target{Name: "Bar", File: filepath.Join(dir, "foo.go")}},
	}
	actions := []Action{
		{Path: testFile, Action: "add_test_case", TestrecID: "T-001"},
	}

	err := Execute(actions, recs, "std", false)
	if err != nil {
		t.Fatal(err)
	}

	// Check backup exists
	bakPath := testFile + ".bak"
	bakData, err := os.ReadFile(bakPath)
	if err != nil {
		t.Fatal("backup file not created")
	}
	if string(bakData) != original {
		t.Error("backup should contain original content")
	}

	// Check appended content
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "TestBar") {
		t.Error("expected TestBar appended")
	}
	if !strings.Contains(string(data), "TestExisting") {
		t.Error("original content should be preserved")
	}
}

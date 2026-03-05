package scaffold

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/dshills/verifier/internal/domain"
)

// Action describes a planned scaffold change.
type Action struct {
	Path      string `json:"path"`
	Action    string `json:"action"` // add_test_case, create_file
	TestrecID string `json:"testrec_id"`
	Status    string `json:"status"` // written, skipped, dry_run
}

// Plan creates scaffold actions for the given recommendations.
func Plan(recs []domain.Recommendation, limit int) ([]Action, []domain.Recommendation) {
	// Filter to critical severity if limit > 0
	var selected []domain.Recommendation
	for _, rec := range recs {
		if limit > 0 && rec.Severity == domain.SeverityCritical {
			selected = append(selected, rec)
		} else if limit <= 0 {
			selected = append(selected, rec)
		}
	}

	if limit > 0 && len(selected) == 0 {
		slog.Info("no critical-severity recommendations to scaffold")
		return nil, nil
	}

	if limit > 0 && len(selected) > limit {
		selected = selected[:limit]
	}

	var actions []Action
	for _, rec := range selected {
		dir := filepath.Dir(rec.Target.File)
		if dir == "." || dir == "" {
			dir = rec.Target.File // package target
		}
		pkgName := filepath.Base(dir)

		testFile := findTestFile(dir, pkgName)
		if testFile == "" {
			// Create new test file
			testFile = filepath.Join(dir, pkgName+"_test.go")
			actions = append(actions, Action{
				Path:      testFile,
				Action:    "create_file",
				TestrecID: rec.ID,
			})
		} else {
			// Check if test already exists
			if hasTestForFunc(testFile, rec.Target.Name) {
				slog.Info("test already exists, skipping", "target", rec.Target.Name, "file", testFile)
				actions = append(actions, Action{
					Path:      testFile,
					Action:    "add_test_case",
					TestrecID: rec.ID,
					Status:    "skipped",
				})
				continue
			}
			actions = append(actions, Action{
				Path:      testFile,
				Action:    "add_test_case",
				TestrecID: rec.ID,
			})
		}
	}

	return actions, selected
}

// Execute writes scaffold changes to disk.
func Execute(actions []Action, recs []domain.Recommendation, style string, dryRun bool) error {
	recMap := make(map[string]*domain.Recommendation)
	for i := range recs {
		recMap[recs[i].ID] = &recs[i]
	}

	for i := range actions {
		action := &actions[i]
		if action.Status == "skipped" {
			continue
		}

		rec := recMap[action.TestrecID]
		if rec == nil {
			continue
		}

		content := GenerateTest(rec, style)

		if dryRun {
			action.Status = "dry_run"
			fmt.Printf("--- %s (%s) ---\n%s\n", action.Path, action.Action, content)
			continue
		}

		switch action.Action {
		case "create_file":
			pkgName := filepath.Base(filepath.Dir(action.Path))
			full := fmt.Sprintf("package %s\n\nimport \"testing\"\n\n%s", pkgName, content)
			if err := os.WriteFile(action.Path, []byte(full), 0644); err != nil {
				return fmt.Errorf("create %s: %w", action.Path, err)
			}
			action.Status = "written"

		case "add_test_case":
			// Create .bak backup
			bakPath := action.Path + ".bak"
			existing, err := os.ReadFile(action.Path)
			if err != nil {
				return fmt.Errorf("read %s: %w", action.Path, err)
			}
			if err := os.WriteFile(bakPath, existing, 0644); err != nil {
				return fmt.Errorf("create backup %s: %w", bakPath, err)
			}
			// Append
			f, err := os.OpenFile(action.Path, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				return fmt.Errorf("open %s: %w", action.Path, err)
			}
			_, err = f.WriteString("\n" + content)
			closeErr := f.Close()
			if err != nil {
				return fmt.Errorf("append to %s: %w", action.Path, err)
			}
			if closeErr != nil {
				return fmt.Errorf("close %s: %w", action.Path, closeErr)
			}
			action.Status = "written"
		}
	}

	return nil
}

func findTestFile(dir, _ string) string {
	// Look for existing *_test.go
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "_test.go") {
			return filepath.Join(dir, e.Name())
		}
	}
	return ""
}

func hasTestForFunc(testFile, funcName string) bool {
	data, err := os.ReadFile(testFile)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "Test"+funcName)
}

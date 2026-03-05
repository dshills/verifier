package scaffold

import (
	"fmt"
	"strings"

	"github.com/dshills/verifier/internal/domain"
)

// GenerateTest generates a test skeleton for the given recommendation.
func GenerateTest(rec *domain.Recommendation, style string) string {
	switch rec.Category {
	case domain.CategoryUnit:
		return generateUnit(rec, style)
	case domain.CategoryIntegration:
		return generateIntegration(rec, style)
	case domain.CategoryConcurrency:
		return generateConcurrency(rec, style)
	case domain.CategoryFuzz:
		return generateFuzz(rec)
	case domain.CategoryProperty:
		return generateProperty(rec, style)
	case domain.CategorySecurity:
		return generateSecurity(rec, style)
	case domain.CategoryContract:
		return generateIntegration(rec, style)
	default:
		return generateUnit(rec, style)
	}
}

func testFuncName(rec *domain.Recommendation) string {
	name := rec.Target.Name
	if name == "" {
		name = "Unknown"
	}
	return "Test" + name
}

func todoComment(rec *domain.Recommendation) string {
	return fmt.Sprintf("// TODO(%s): %s", rec.ID, rec.Proposal.Title)
}

func generateUnit(rec *domain.Recommendation, style string) string {
	var sb strings.Builder
	sb.WriteString(todoComment(rec) + "\n")
	sb.WriteString(fmt.Sprintf("func %s(t *testing.T) {\n", testFuncName(rec)))
	sb.WriteString("\ttests := []struct {\n")
	sb.WriteString("\t\tname string\n")
	sb.WriteString("\t}{\n")
	sb.WriteString("\t\t{name: \"valid input\"},\n")
	sb.WriteString("\t\t{name: \"invalid input\"},\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\tfor _, tt := range tests {\n")
	sb.WriteString("\t\tt.Run(tt.name, func(t *testing.T) {\n")
	if style == "go-testify" {
		sb.WriteString("\t\t\t// require.NotNil(t, result) // TODO: add testify assertions\n")
	} else {
		sb.WriteString("\t\t\t// TODO: implement test case\n")
	}
	sb.WriteString("\t\t})\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n")
	return sb.String()
}

func generateIntegration(rec *domain.Recommendation, style string) string {
	var sb strings.Builder
	sb.WriteString(todoComment(rec) + "\n")
	sb.WriteString(fmt.Sprintf("func %s(t *testing.T) {\n", testFuncName(rec)))
	sb.WriteString("\t// Setup\n")
	sb.WriteString("\t// TODO: initialize dependencies\n\n")
	sb.WriteString("\tt.Run(\"happy path\", func(t *testing.T) {\n")
	if style == "go-testify" {
		sb.WriteString("\t\t// assert.NoError(t, err) // TODO: add assertions\n")
	} else {
		sb.WriteString("\t\t// TODO: implement test\n")
	}
	sb.WriteString("\t})\n\n")
	sb.WriteString("\tt.Run(\"error path\", func(t *testing.T) {\n")
	sb.WriteString("\t\t// TODO: test error handling\n")
	sb.WriteString("\t})\n\n")
	sb.WriteString("\t// Teardown\n")
	sb.WriteString("\t// TODO: cleanup resources\n")
	sb.WriteString("}\n")
	return sb.String()
}

func generateConcurrency(rec *domain.Recommendation, _ string) string {
	var sb strings.Builder
	sb.WriteString(todoComment(rec) + "\n")
	sb.WriteString(fmt.Sprintf("func %s(t *testing.T) {\n", testFuncName(rec)))
	sb.WriteString("\tt.Parallel()\n\n")
	sb.WriteString("\t// TODO: launch concurrent goroutines\n")
	sb.WriteString("\t// TODO: verify thread safety\n")
	sb.WriteString("\t// Run with: go test -race\n")
	sb.WriteString("}\n")
	return sb.String()
}

func generateFuzz(rec *domain.Recommendation) string {
	var sb strings.Builder
	sb.WriteString(todoComment(rec) + "\n")
	funcName := rec.Target.Name
	if funcName == "" {
		funcName = "Unknown"
	}
	sb.WriteString(fmt.Sprintf("func Fuzz%s(f *testing.F) {\n", funcName))
	sb.WriteString("\tf.Add(\"seed input\")\n")
	sb.WriteString("\tf.Fuzz(func(t *testing.T, input string) {\n")
	sb.WriteString("\t\t// TODO: call function and verify no panic\n")
	sb.WriteString("\t})\n")
	sb.WriteString("}\n")
	return sb.String()
}

func generateProperty(rec *domain.Recommendation, _ string) string {
	var sb strings.Builder
	sb.WriteString(todoComment(rec) + "\n")
	sb.WriteString(fmt.Sprintf("func %s(t *testing.T) {\n", testFuncName(rec)))
	sb.WriteString("\t// Property: TODO define invariant\n")
	sb.WriteString("\t// For all valid inputs, the following should hold:\n")
	sb.WriteString("\t// TODO: implement property check\n")
	sb.WriteString("}\n")
	return sb.String()
}

func generateSecurity(rec *domain.Recommendation, _ string) string {
	var sb strings.Builder
	sb.WriteString(todoComment(rec) + "\n")
	sb.WriteString(fmt.Sprintf("func %s(t *testing.T) {\n", testFuncName(rec)))
	sb.WriteString("\t// Security: verify authentication and authorization\n")
	sb.WriteString("\tt.Run(\"unauthenticated request\", func(t *testing.T) {\n")
	sb.WriteString("\t\t// TODO: verify rejection\n")
	sb.WriteString("\t})\n\n")
	sb.WriteString("\tt.Run(\"input sanitization\", func(t *testing.T) {\n")
	sb.WriteString("\t\t// TODO: test with malicious input\n")
	sb.WriteString("\t})\n")
	sb.WriteString("}\n")
	return sb.String()
}

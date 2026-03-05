package golang

import (
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"strings"

	"github.com/dshills/verifier/internal/domain"
)

// enrichTests enriches the test inventory with AST-detected subtests and line numbers.
func enrichTests(graph *domain.RepoGraph, inv *domain.TestInventory) {
	var enriched []domain.TestInfo

	for _, pkg := range graph.Packages {
		for _, testFile := range pkg.TestFiles {
			tests := parseTestFile(testFile, pkg.Name)
			enriched = append(enriched, tests...)
		}
	}

	if len(enriched) > 0 {
		inv.Tests = enriched
	}
}

func parseTestFile(path, pkgName string) []domain.TestInfo {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		slog.Warn("failed to parse test file", "path", path, "err", err)
		return nil
	}

	var tests []domain.TestInfo

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if !strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}

		tests = append(tests, domain.TestInfo{
			File:     path,
			Package:  pkgName,
			FuncName: fn.Name.Name,
			Line:     fset.Position(fn.Pos()).Line,
		})

		// Detect subtests: t.Run("name", ...)
		if fn.Body != nil {
			subtests := detectSubtests(fset, fn.Body, path, pkgName)
			tests = append(tests, subtests...)
		}
	}

	return tests
}

func detectSubtests(fset *token.FileSet, body *ast.BlockStmt, path, pkgName string) []domain.TestInfo {
	var subtests []domain.TestInfo

	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Run" {
			return true
		}

		if len(call.Args) < 2 {
			return true
		}

		// Get subtest name from first arg (string literal)
		if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
			name := strings.Trim(lit.Value, `"`)
			subtests = append(subtests, domain.TestInfo{
				File:      path,
				Package:   pkgName,
				FuncName:  name,
				IsSubtest: true,
				Line:      fset.Position(call.Pos()).Line,
			})
		}

		return true
	})

	return subtests
}

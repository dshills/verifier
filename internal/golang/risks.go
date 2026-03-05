package golang

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/dshills/verifier/internal/domain"
)

func detectFuncRisks(_ *token.FileSet, fn *ast.FuncDecl, file *ast.File) []string {
	var risks []string

	if returnsError(fn) {
		risks = append(risks, domain.RiskErrorPath)
	}
	if hasConcurrency(fn) {
		risks = append(risks, domain.RiskConcurrency)
	}
	if isHTTPHandler(fn) {
		risks = append(risks, domain.RiskHTTPHandler)
	}
	if hasDBCalls(fn) {
		risks = append(risks, domain.RiskDBQuery)
	}
	if isHighComplexity(fn) {
		risks = append(risks, domain.RiskComplexity)
	}
	if hasInputValidation(fn) {
		risks = append(risks, domain.RiskInputValidation)
	}
	if isBoundaryFunc(fn, file) {
		risks = append(risks, domain.RiskBoundary)
	}

	return risks
}

func returnsError(fn *ast.FuncDecl) bool {
	if fn.Type.Results == nil {
		return false
	}
	for _, field := range fn.Type.Results.List {
		if ident, ok := field.Type.(*ast.Ident); ok && ident.Name == "error" {
			return true
		}
	}
	return false
}

func hasConcurrency(fn *ast.FuncDecl) bool {
	if fn.Body == nil {
		return false
	}
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if found {
			return false
		}
		switch n.(type) {
		case *ast.GoStmt:
			found = true
			return false
		case *ast.SendStmt:
			found = true
			return false
		}
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				name := sel.Sel.Name
				if name == "Lock" || name == "Unlock" || name == "RLock" || name == "RUnlock" ||
					name == "Wait" || name == "Add" || name == "Done" {
					if x, ok := sel.X.(*ast.Ident); ok {
						_ = x // sync types detected by method names
						found = true
						return false
					}
				}
			}
		}
		return true
	})
	return found
}

func isHTTPHandler(fn *ast.FuncDecl) bool {
	if fn.Type.Params == nil || len(fn.Type.Params.List) < 2 {
		return false
	}

	params := fn.Type.Params.List
	// Check for (http.ResponseWriter, *http.Request) pattern
	hasWriter := false
	hasRequest := false

	for _, p := range params {
		typStr := exprString(p.Type)
		if typStr == "http.ResponseWriter" {
			hasWriter = true
		}
		if typStr == "*http.Request" {
			hasRequest = true
		}
	}

	return hasWriter && hasRequest
}

func hasDBCalls(fn *ast.FuncDecl) bool {
	if fn.Body == nil {
		return false
	}
	dbMethods := map[string]bool{
		"Query": true, "QueryContext": true, "QueryRow": true,
		"QueryRowContext": true, "Exec": true, "ExecContext": true,
	}
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if found {
			return false
		}
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if dbMethods[sel.Sel.Name] {
					found = true
					return false
				}
			}
		}
		return true
	})
	return found
}

func isHighComplexity(fn *ast.FuncDecl) bool {
	if fn.Body == nil {
		return false
	}
	branches := 0
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt, *ast.CaseClause, *ast.CommClause:
			branches++
		}
		return true
	})
	return branches > 5
}

func hasInputValidation(fn *ast.FuncDecl) bool {
	if fn.Body == nil {
		return false
	}

	// Check function name
	name := strings.ToLower(fn.Name.Name)
	validationNames := []string{"validat", "check", "pars", "sanitiz", "decode", "unmarshal"}
	for _, v := range validationNames {
		if strings.Contains(name, v) {
			return true
		}
	}

	// Check for string comparisons and length checks in first 20 statements
	count := 0
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if found || count > 20 {
			return false
		}
		if _, ok := n.(ast.Stmt); ok {
			count++
		}
		if bin, ok := n.(*ast.BinaryExpr); ok {
			if bin.Op == token.EQL || bin.Op == token.NEQ {
				found = true
			}
		}
		return true
	})
	return found
}

func isBoundaryFunc(fn *ast.FuncDecl, file *ast.File) bool {
	if isHTTPHandler(fn) || hasDBCalls(fn) {
		return true
	}
	// Check if the file imports boundary packages
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if path == "net/http" || path == "database/sql" {
			return true
		}
	}
	return false
}

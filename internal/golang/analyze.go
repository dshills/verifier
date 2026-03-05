package golang

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"os"
	"strings"

	"github.com/dshills/verifier/internal/domain"
)

const maxFileSize = 1 << 20 // 1MB

// AnalyzePackages walks all packages and extracts symbols and risk signals.
func AnalyzePackages(graph *domain.RepoGraph, cfg *domain.Config) (*domain.SymbolIndex, *domain.RiskSignals, error) {
	idx := &domain.SymbolIndex{}
	risks := &domain.RiskSignals{}

	for _, pkg := range graph.Packages {
		for _, goFile := range pkg.GoFiles {
			info, err := os.Stat(goFile)
			if err != nil {
				continue
			}
			if info.Size() > maxFileSize {
				slog.Warn("skipping large file for AST", "path", goFile)
				continue
			}

			syms, sigs := analyzeFile(goFile, pkg.Name)
			idx.Symbols = append(idx.Symbols, syms...)
			risks.Signals = append(risks.Signals, sigs...)
		}
	}

	return idx, risks, nil
}

func analyzeFile(path, pkgName string) ([]domain.Symbol, []domain.RiskSignal) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		slog.Warn("failed to parse", "path", path, "err", err)
		return nil, nil
	}

	var symbols []domain.Symbol
	var signals []domain.RiskSignal

	ast.Inspect(f, func(n ast.Node) bool {
		switch decl := n.(type) {
		case *ast.FuncDecl:
			sym := extractFuncSymbol(fset, decl, path, pkgName)
			symbols = append(symbols, sym)

			riskList := detectFuncRisks(fset, decl, f)
			if len(riskList) > 0 {
				signals = append(signals, domain.RiskSignal{
					Symbol:  sym.Name,
					File:    path,
					Package: pkgName,
					Risks:   riskList,
				})
			}

		case *ast.GenDecl:
			if decl.Tok == token.TYPE {
				for _, spec := range decl.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					kind := "type"
					if _, isIface := ts.Type.(*ast.InterfaceType); isIface {
						kind = "interface"
					}
					symbols = append(symbols, domain.Symbol{
						Name:      ts.Name.Name,
						Package:   pkgName,
						File:      path,
						LineStart: fset.Position(ts.Pos()).Line,
						LineEnd:   fset.Position(ts.End()).Line,
						Kind:      kind,
						Exported:  ts.Name.IsExported(),
					})
				}
			}
		}
		return true
	})

	return symbols, signals
}

func extractFuncSymbol(fset *token.FileSet, fn *ast.FuncDecl, path, pkgName string) domain.Symbol {
	sym := domain.Symbol{
		Name:      fn.Name.Name,
		Package:   pkgName,
		File:      path,
		LineStart: fset.Position(fn.Pos()).Line,
		LineEnd:   fset.Position(fn.End()).Line,
		Kind:      "function",
		Exported:  fn.Name.IsExported(),
		Signature: buildSignature(fn),
	}

	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		sym.Kind = "method"
		sym.ReceiverType = exprString(fn.Recv.List[0].Type)
	}

	return sym
}

func buildSignature(fn *ast.FuncDecl) string {
	var sb strings.Builder
	sb.WriteString("func ")
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		sb.WriteString("(")
		sb.WriteString(exprString(fn.Recv.List[0].Type))
		sb.WriteString(") ")
	}
	sb.WriteString(fn.Name.Name)
	sb.WriteString("(")
	if fn.Type.Params != nil {
		writeFieldList(&sb, fn.Type.Params.List)
	}
	sb.WriteString(")")
	if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
		sb.WriteString(" ")
		if len(fn.Type.Results.List) > 1 {
			sb.WriteString("(")
		}
		writeFieldList(&sb, fn.Type.Results.List)
		if len(fn.Type.Results.List) > 1 {
			sb.WriteString(")")
		}
	}
	return sb.String()
}

func writeFieldList(sb *strings.Builder, fields []*ast.Field) {
	for i, field := range fields {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(exprString(field.Type))
	}
}

func exprString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return exprString(e.X) + "." + e.Sel.Name
	case *ast.StarExpr:
		return "*" + exprString(e.X)
	case *ast.ArrayType:
		return "[]" + exprString(e.Elt)
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", exprString(e.Key), exprString(e.Value))
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.Ellipsis:
		return "..." + exprString(e.Elt)
	case *ast.FuncType:
		return "func(...)"
	case *ast.ChanType:
		return "chan " + exprString(e.Value)
	default:
		return "?"
	}
}

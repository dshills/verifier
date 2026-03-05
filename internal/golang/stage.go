package golang

import (
	"context"

	"github.com/dshills/verifier/internal/domain"
)

// Stage implements Stage C: Go AST Analysis.
type Stage struct{}

func (Stage) Name() string { return "go-ast-analysis" }

func (Stage) Execute(_ context.Context, cfg *domain.Config, arts *domain.Artifacts) error {
	if arts.RepoGraph == nil {
		return nil
	}

	symbols, risks, err := AnalyzePackages(arts.RepoGraph, cfg)
	if err != nil {
		return err
	}

	arts.SymbolIndex = symbols
	arts.RiskSignals = risks

	// Enrich test inventory with AST data
	if arts.TestInventory == nil {
		arts.TestInventory = &domain.TestInventory{}
	}
	enrichTests(arts.RepoGraph, arts.TestInventory)

	return nil
}

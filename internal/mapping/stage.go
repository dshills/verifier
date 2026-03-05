package mapping

import (
	"context"

	"github.com/dshills/verifier/internal/domain"
)

// Stage implements Stage D: Mapping.
type Stage struct{}

func (Stage) Name() string { return "mapping" }

func (Stage) Execute(_ context.Context, _ *domain.Config, arts *domain.Artifacts) error {
	if arts.RequirementSet == nil || arts.SymbolIndex == nil {
		arts.CoverageMap = &domain.CoverageMap{}
		arts.UnmappedRequirements = &domain.UnmappedRequirements{}
		arts.UntestedIntents = &domain.UntestedIntents{}
		return nil
	}

	cm, unmapped := MapRequirements(arts.RequirementSet, arts.SymbolIndex)
	arts.CoverageMap = cm
	arts.UnmappedRequirements = unmapped

	if arts.PlanIntentSet != nil && arts.TestInventory != nil {
		arts.UntestedIntents = FindUntestedIntents(arts.PlanIntentSet, arts.TestInventory, arts.RepoGraph)
	} else {
		arts.UntestedIntents = &domain.UntestedIntents{}
	}

	return nil
}

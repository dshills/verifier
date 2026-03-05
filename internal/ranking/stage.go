package ranking

import (
	"context"

	"github.com/dshills/verifier/internal/domain"
)

// Stage implements Stage G: Ranking and Reporting.
type Stage struct{}

func (Stage) Name() string { return "ranking" }

func (Stage) Execute(_ context.Context, cfg *domain.Config, arts *domain.Artifacts) error {
	// Assign severity and TESTREC IDs
	AssignSeverity(arts.Recommendations, arts.RequirementSet)
	AssignIDs(arts.Recommendations)

	// Sort: severity desc → confidence desc → ID asc
	SortRecommendations(arts.Recommendations)

	return nil
}

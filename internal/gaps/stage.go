package gaps

import (
	"context"

	"github.com/dshills/verifier/internal/domain"
)

// Stage implements Stage F: Gap Detection.
type Stage struct{}

func (Stage) Name() string { return "gap-detection" }

func (Stage) Execute(_ context.Context, _ *domain.Config, arts *domain.Artifacts) error {
	if arts.Recommendations == nil && arts.RepoGraph != nil {
		// Degraded mode: generate findings from code signals only
		arts.Recommendations = DegradedFindings(arts)
	}

	if len(arts.Recommendations) > 0 && arts.TestInventory != nil {
		AnnotateExistingTests(arts.Recommendations, arts.TestInventory)
	}

	return nil
}

package strategy

import (
	"context"

	"github.com/dshills/verifier/internal/domain"
)

// Stage implements Stage E: Test Strategy.
type Stage struct{}

func (Stage) Name() string { return "test-strategy" }

func (Stage) Execute(_ context.Context, _ *domain.Config, arts *domain.Artifacts) error {
	if arts.CoverageMap == nil {
		return nil
	}

	recs := AssignStrategies(arts)
	arts.Recommendations = recs
	return nil
}

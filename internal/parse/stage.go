package parse

import (
	"context"
	"fmt"
	"os"

	"github.com/dshills/verifier/internal/domain"
)

// Stage implements Stage B: Spec & Plan Extraction.
type Stage struct{}

func (Stage) Name() string { return "spec-plan-extraction" }

func (Stage) Execute(_ context.Context, cfg *domain.Config, arts *domain.Artifacts) error {
	reqSet := &domain.RequirementSet{}
	planSet := &domain.PlanIntentSet{}
	idTracker := newIDTracker()

	for _, path := range cfg.SpecPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("read spec %s: %w", path, err)
		}
		reqs := ExtractRequirements(string(data), path, idTracker)
		reqSet.Requirements = append(reqSet.Requirements, reqs...)
	}

	for _, path := range cfg.PlanPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("read plan %s: %w", path, err)
		}
		intents := ExtractPlanIntents(string(data), path, idTracker)
		planSet.Intents = append(planSet.Intents, intents...)
	}

	// Check for ID conflicts
	if err := idTracker.validate(); err != nil {
		return err
	}

	arts.RequirementSet = reqSet
	arts.PlanIntentSet = planSet
	return nil
}

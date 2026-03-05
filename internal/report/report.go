package report

import (
	"time"

	"github.com/dshills/verifier/internal/domain"
	"github.com/dshills/verifier/internal/ranking"
)

// Build creates a Report from the pipeline artifacts and config.
func Build(arts *domain.Artifacts, cfg *domain.Config, version string) *domain.Report {
	recs := arts.Recommendations

	// Compute risk score from full set (before truncation)
	riskScore := ranking.ComputeRiskScore(recs)
	totalFindings := len(recs)

	// Truncate
	var truncated bool
	recs, truncated = ranking.Truncate(recs, cfg.MaxFindings)

	// Gather requirements
	var reqs []domain.Requirement
	if arts.RequirementSet != nil {
		reqs = arts.RequirementSet.Requirements
	}

	// Count unverifiable
	unverifiable := 0
	for _, req := range reqs {
		if req.Verifiability == domain.VerifiabilityLow {
			unverifiable++
		}
	}

	// Count missing (unmapped)
	missing := 0
	if arts.UnmappedRequirements != nil {
		missing = len(arts.UnmappedRequirements.IDs)
	}

	// Collect input files
	specFiles := append([]string{}, cfg.SpecPaths...)
	planFiles := append([]string{}, cfg.PlanPaths...)

	report := &domain.Report{
		Meta: domain.Meta{
			Tool:      "verifier",
			Version:   version,
			RepoRoot:  cfg.Root,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Seed:      cfg.Seed,
			Mode:      cfg.Mode,
			Inputs: domain.InputFiles{
				SpecFiles: specFiles,
				PlanFiles: planFiles,
			},
		},
		Summary: domain.Summary{
			RiskScore:                riskScore,
			TotalFindings:            totalFindings,
			Truncated:                truncated,
			MissingRecommendations:   missing,
			UnverifiableRequirements: unverifiable,
		},
		Requirements:    reqs,
		Recommendations: recs,
	}

	return report
}

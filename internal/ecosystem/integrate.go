package ecosystem

import (
	"fmt"
	"regexp"

	"github.com/dshills/verifier/internal/domain"
)

var reqIDPattern = regexp.MustCompile(`REQ-\d+`)

// ApplySpecCritic boosts severity of recommendations linked to SpecCritic issues.
func ApplySpecCritic(recs *[]domain.Recommendation, sc *SpecCriticReport) {
	if sc == nil {
		return
	}
	for _, issue := range sc.Issues {
		// Extract requirement ID from anchor
		reqID := reqIDPattern.FindString(issue.Anchor)
		if reqID == "" {
			continue
		}
		for i := range *recs {
			for _, coveredReq := range (*recs)[i].Covers.Requirements {
				if coveredReq == reqID {
					(*recs)[i].Severity = boostSeverity((*recs)[i].Severity)
					break
				}
			}
		}
	}
}

// ApplyPlanCritic generates additional recommendations for flagged components.
func ApplyPlanCritic(recs *[]domain.Recommendation, pc *PlanCriticReport) {
	if pc == nil {
		return
	}
	for _, issue := range pc.Issues {
		rec := domain.Recommendation{
			Severity: issue.Severity,
			Category: domain.CategoryUnit,
			Target: domain.Target{
				Kind: domain.TargetComponent,
				Name: issue.Component,
			},
			Covers: domain.Covers{
				PlanItems: []string{issue.ID},
			},
			Proposal: domain.Proposal{
				Title:    "Test plan risk: " + issue.Title,
				Approach: "Address plan-identified risk in " + issue.Component,
			},
		}
		*recs = append(*recs, rec)
	}
}

// ApplyRealityCheck generates regression test recommendations for each delta.
func ApplyRealityCheck(recs *[]domain.Recommendation, rc *RealityCheckReport) {
	if rc == nil {
		return
	}
	for _, delta := range rc.Deltas {
		targetSymbol := delta.CodeRef
		if targetSymbol == "" {
			targetSymbol = "unknown"
		}
		rec := domain.Recommendation{
			Severity: domain.SeverityHigh,
			Category: domain.CategoryIntegration,
			Target: domain.Target{
				Kind: domain.TargetComponent,
				Name: targetSymbol,
			},
			Covers: domain.Covers{
				Requirements: []string{fmt.Sprintf("RC-%s", delta.ID)},
			},
			Proposal: domain.Proposal{
				Title:    "Regression test for delta: " + delta.Description,
				Approach: "Verify behavior after " + delta.Kind + " change",
			},
		}
		*recs = append(*recs, rec)
	}
}

// ApplyPrism promotes flagged code sections to test recommendations.
func ApplyPrism(recs *[]domain.Recommendation, pr *PrismReport) {
	if pr == nil {
		return
	}
	for _, finding := range pr.Findings {
		targetName := finding.File
		if finding.LineStart > 0 {
			targetName = fmt.Sprintf("%s:%d", finding.File, finding.LineStart)
		}
		rec := domain.Recommendation{
			Severity: finding.Severity,
			Category: domain.CategoryUnit,
			Target: domain.Target{
				Kind:      domain.TargetFunction,
				Name:      targetName,
				File:      finding.File,
				LineStart: finding.LineStart,
				LineEnd:   finding.LineEnd,
			},
			Covers: domain.Covers{
				Requirements: []string{fmt.Sprintf("PRISM-%s", finding.ID)},
			},
			Proposal: domain.Proposal{
				Title:    "Test for prism finding: " + finding.Message,
				Approach: "Address code review finding",
			},
		}
		*recs = append(*recs, rec)
	}
}

func boostSeverity(sev string) string {
	switch sev {
	case domain.SeverityLow:
		return domain.SeverityMedium
	case domain.SeverityMedium:
		return domain.SeverityHigh
	case domain.SeverityHigh:
		return domain.SeverityCritical
	default:
		return sev
	}
}

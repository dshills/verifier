package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/dshills/verifier/internal/domain"
)

// WriteText writes the report as plain text.
func WriteText(w io.Writer, rpt *domain.Report) error {
	p := func(format string, args ...any) { _, _ = fmt.Fprintf(w, format, args...) }

	p("Verifier Report\n")
	p("===============\n\n")
	p("Tool:       %s v%s\n", rpt.Meta.Tool, rpt.Meta.Version)
	p("Repository: %s\n", rpt.Meta.RepoRoot)
	p("Mode:       %s\n", rpt.Meta.Mode)
	p("Timestamp:  %s\n\n", rpt.Meta.Timestamp)

	p("Summary\n")
	p("-------\n")
	p("Risk Score:         %d\n", rpt.Summary.RiskScore)
	p("Total Findings:     %d\n", rpt.Summary.TotalFindings)
	p("Truncated:          %v\n", rpt.Summary.Truncated)
	p("Missing Recs:       %d\n", rpt.Summary.MissingRecommendations)
	p("Unverifiable Reqs:  %d\n\n", rpt.Summary.UnverifiableRequirements)

	for _, sev := range []string{"critical", "high", "medium", "low"} {
		var filtered []domain.Recommendation
		for _, rec := range rpt.Recommendations {
			if rec.Severity == sev {
				filtered = append(filtered, rec)
			}
		}
		if len(filtered) == 0 {
			continue
		}

		title := strings.ToUpper(sev[:1]) + sev[1:]
		p("%s Gaps\n", title)
		p("%s\n\n", strings.Repeat("-", len(title)+5))

		for _, rec := range filtered {
			p("  %s: %s\n", rec.ID, rec.Proposal.Title)
			p("    Category:   %s\n", rec.Category)
			p("    Confidence: %.1f\n", rec.Confidence)
			p("    Target:     %s %s (%s)\n", rec.Target.Kind, rec.Target.Name, rec.Target.File)
			p("    Approach:   %s\n", rec.Proposal.Approach)
			if len(rec.ExistingTests) > 0 {
				p("    Existing:   ")
				for i, et := range rec.ExistingTests {
					if i > 0 {
						p(", ")
					}
					p("%s", et.Name)
					if et.Gap != "" {
						p(" (gap: %s)", et.Gap)
					}
				}
				p("\n")
			}
			p("\n")
		}
	}

	return nil
}

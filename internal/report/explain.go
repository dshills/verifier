package report

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/dshills/verifier/internal/domain"
)

// LoadJSON reads a report from a JSON file.
func LoadJSON(path string) (*domain.Report, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open report: %w", err)
	}
	defer func() { _ = f.Close() }()
	return ReadJSON(f)
}

// ReadJSON reads a report from a reader.
func ReadJSON(r io.Reader) (*domain.Report, error) {
	var rpt domain.Report
	if err := json.NewDecoder(r).Decode(&rpt); err != nil {
		return nil, fmt.Errorf("decode report: %w", err)
	}
	return &rpt, nil
}

// ExplainRecommendation prints a detailed explanation of a recommendation.
func ExplainRecommendation(w io.Writer, rec *domain.Recommendation) {
	p := func(format string, args ...any) { _, _ = fmt.Fprintf(w, format, args...) }

	p("ID:         %s\n", rec.ID)
	p("Severity:   %s\n", rec.Severity)
	p("Category:   %s\n", rec.Category)
	p("Confidence: %.2f\n\n", rec.Confidence)

	p("Target:\n")
	p("  Kind: %s\n", rec.Target.Kind)
	p("  Name: %s\n", rec.Target.Name)
	p("  File: %s\n", rec.Target.File)
	if rec.Target.LineStart > 0 {
		p("  Lines: %d-%d\n", rec.Target.LineStart, rec.Target.LineEnd)
	}

	p("\nProposal:\n")
	p("  Title:    %s\n", rec.Proposal.Title)
	p("  Approach: %s\n", rec.Proposal.Approach)
	if len(rec.Proposal.Assertions) > 0 {
		p("  Assertions:\n")
		for _, a := range rec.Proposal.Assertions {
			p("    - %s\n", a)
		}
	}

	if len(rec.Covers.Requirements) > 0 {
		p("\nCovers Requirements: %v\n", rec.Covers.Requirements)
	}
	if len(rec.Covers.Risks) > 0 {
		p("Covers Risks: %v\n", rec.Covers.Risks)
	}

	if len(rec.ExistingTests) > 0 {
		p("\nExisting Tests:\n")
		for _, et := range rec.ExistingTests {
			p("  - %s (%s)", et.Name, et.File)
			if et.Gap != "" {
				p(" — gap: %s", et.Gap)
			}
			p("\n")
		}
	}

	if len(rec.Evidence) > 0 {
		p("\nEvidence:\n")
		for _, ev := range rec.Evidence {
			p("  - [%s] %s", ev.Kind, ev.File)
			if ev.Symbol != "" {
				p(" (%s)", ev.Symbol)
			}
			p("\n")
		}
	}
}

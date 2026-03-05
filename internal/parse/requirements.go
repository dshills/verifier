package parse

import (
	"regexp"
	"strings"

	"github.com/dshills/verifier/internal/domain"
)

var requirementKeywords = []string{
	"requirement", "must", "shall", "feature",
}

var explicitIDPattern = regexp.MustCompile(`^((?:REQ|FR|NFR|CR|SR|IR)-\d+)`)

var acceptanceCriteriaKeywords = []string{
	"accept", "criteria", "measurable", "verify", "assert",
}

// ExtractRequirements parses markdown text and returns requirements.
func ExtractRequirements(text, source string, tracker *idTracker) []domain.Requirement {
	sections := ParseSections(text)
	var reqs []domain.Requirement

	for _, sec := range sections {
		if !isRequirementHeading(sec.Heading.Text) {
			continue
		}

		items := ExtractListItems(sec.Body)
		for i, item := range items {
			id := extractExplicitID(item)
			if id == "" {
				id = generateSyntheticID(sec.Heading.Text, i, tracker)
			} else {
				tracker.recordExplicit(id)
			}

			ac := extractAcceptanceCriteria(sec.Body, i)
			verifiability := domain.VerifiabilityMedium
			if len(ac) == 0 {
				verifiability = domain.VerifiabilityLow
			}

			reqs = append(reqs, domain.Requirement{
				ID:                 id,
				Text:               item,
				Verifiability:      verifiability,
				AcceptanceCriteria: ac,
				Source:             source,
				HeadingContext:     sec.Heading.Text,
			})
		}
	}

	return reqs
}

func isRequirementHeading(text string) bool {
	lower := strings.ToLower(text)
	for _, kw := range requirementKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func extractExplicitID(item string) string {
	matches := explicitIDPattern.FindStringSubmatch(item)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func extractAcceptanceCriteria(bodyLines []string, itemIndex int) []string {
	// Look for sub-items after the target item that contain AC keywords
	var criteria []string
	currentItem := -1
	for _, line := range bodyLines {
		trimmed := strings.TrimSpace(line)
		if _, ok := parseListItem(trimmed); ok {
			currentItem++
			if currentItem > itemIndex {
				// Check if this is a sub-item with AC keywords
				if hasAcceptanceCriteriaKeyword(trimmed) {
					criteria = append(criteria, trimmed)
				}
			}
		}
	}
	return criteria
}

func hasAcceptanceCriteriaKeyword(text string) bool {
	lower := strings.ToLower(text)
	for _, kw := range acceptanceCriteriaKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

package parse

import (
	"strings"

	"github.com/dshills/verifier/internal/domain"
)

var planKeywords = []string{
	"component", "module", "architecture", "design", "plan", "integration",
}

// ExtractPlanIntents parses markdown text and returns plan intents.
func ExtractPlanIntents(text, source string, tracker *idTracker) []domain.PlanIntent {
	sections := ParseSections(text)
	var intents []domain.PlanIntent

	for _, sec := range sections {
		if !isPlanHeading(sec.Heading.Text) {
			continue
		}

		items := ExtractListItems(sec.Body)
		for i, item := range items {
			id := generatePlanID(sec.Heading.Text, i, tracker)

			intent := domain.PlanIntent{
				ID:             id,
				Component:      extractComponent(sec.Heading.Text),
				Responsibility: item,
				Description:    item,
				Source:         source,
			}

			// Extract integration points from sub-items
			intent.IntegrationPoints = extractIntegrationPoints(sec.Body, i)

			intents = append(intents, intent)
		}
	}

	return intents
}

func isPlanHeading(text string) bool {
	lower := strings.ToLower(text)
	for _, kw := range planKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func extractComponent(heading string) string {
	// Use the heading text as the component name, cleaned up
	return strings.TrimSpace(heading)
}

func extractIntegrationPoints(bodyLines []string, itemIndex int) []string {
	var points []string
	currentItem := -1
	inSubItems := false

	for _, line := range bodyLines {
		trimmed := strings.TrimSpace(line)
		if _, ok := parseListItem(trimmed); ok {
			if strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t") {
				// Sub-item
				if inSubItems {
					lower := strings.ToLower(trimmed)
					if strings.Contains(lower, "integrat") || strings.Contains(lower, "connect") ||
						strings.Contains(lower, "depend") || strings.Contains(lower, "interact") {
						if item, ok := parseListItem(trimmed); ok {
							points = append(points, item)
						}
					}
				}
			} else {
				currentItem++
				inSubItems = currentItem == itemIndex
			}
		}
	}
	return points
}

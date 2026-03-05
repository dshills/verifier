package parse

import (
	"fmt"
	"regexp"
	"strings"
)

var nonAlphanumeric = regexp.MustCompile(`[^A-Z0-9]`)

type idTracker struct {
	seen     map[string]int
	explicit map[string]bool
}

func newIDTracker() *idTracker {
	return &idTracker{
		seen:     make(map[string]int),
		explicit: make(map[string]bool),
	}
}

func (t *idTracker) recordExplicit(id string) {
	t.explicit[id] = true
	t.seen[id]++
}

func (t *idTracker) validate() error {
	for id, count := range t.seen {
		if t.explicit[id] && count > 1 {
			return fmt.Errorf("duplicate requirement ID %q found across spec files", id)
		}
	}
	return nil
}

func (t *idTracker) uniqueID(base string) string {
	t.seen[base]++
	count := t.seen[base]
	if count == 1 {
		return base
	}
	suffixed := fmt.Sprintf("%s-%d", base, count)
	t.seen[suffixed]++
	return suffixed
}

func normalizeHeading(heading string) string {
	upper := strings.ToUpper(heading)
	cleaned := nonAlphanumeric.ReplaceAllString(upper, "")
	if len(cleaned) > 10 {
		cleaned = cleaned[:10]
	}
	return cleaned
}

func generateSyntheticID(heading string, index int, tracker *idTracker) string {
	norm := normalizeHeading(heading)
	base := fmt.Sprintf("SYN-%s-%03d", norm, index+1)
	return tracker.uniqueID(base)
}

func generatePlanID(heading string, index int, tracker *idTracker) string {
	norm := normalizeHeading(heading)
	base := fmt.Sprintf("PLAN-%s-%03d", norm, index+1)
	return tracker.uniqueID(base)
}

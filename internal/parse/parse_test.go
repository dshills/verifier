package parse

import (
	"strings"
	"testing"
)

func TestParseSectionsATX(t *testing.T) {
	md := `# Top Level
some body

## Sub Section
- item 1
- item 2

### Deep
content here
`
	sections := ParseSections(md)
	if len(sections) != 3 {
		t.Fatalf("sections = %d, want 3", len(sections))
	}
	if sections[0].Heading.Level != 1 {
		t.Errorf("first heading level = %d, want 1", sections[0].Heading.Level)
	}
	if sections[1].Heading.Level != 2 {
		t.Errorf("second heading level = %d, want 2", sections[1].Heading.Level)
	}
}

func TestExtractListItems(t *testing.T) {
	lines := []string{
		"- bullet one",
		"* bullet two",
		"1. numbered one",
		"2) numbered two",
		"not a list item",
		"  - indented bullet",
	}
	items := ExtractListItems(lines)
	if len(items) != 5 {
		t.Fatalf("items = %d, want 5", len(items))
	}
}

func TestExtractRequirements(t *testing.T) {
	md := `# Requirements

- REQ-001 The system must authenticate users
- The system shall log all access attempts
- FR-002 Feature: support OAuth2

# Other Information

- This should not be extracted
`
	tracker := newIDTracker()
	reqs := ExtractRequirements(md, "test.md", tracker)

	if len(reqs) != 3 {
		t.Fatalf("reqs = %d, want 3", len(reqs))
	}

	// First should have explicit ID
	if reqs[0].ID != "REQ-001" {
		t.Errorf("first req ID = %q, want REQ-001", reqs[0].ID)
	}

	// Second should get synthetic ID
	if !strings.HasPrefix(reqs[1].ID, "SYN-") {
		t.Errorf("second req ID = %q, want SYN- prefix", reqs[1].ID)
	}

	// Third should have explicit ID
	if reqs[2].ID != "FR-002" {
		t.Errorf("third req ID = %q, want FR-002", reqs[2].ID)
	}
}

func TestSyntheticIDCollision(t *testing.T) {
	tracker := newIDTracker()

	id1 := generateSyntheticID("Requirements", 0, tracker)
	id2 := generateSyntheticID("Requirements", 0, tracker)

	if id1 == id2 {
		t.Errorf("collision not handled: both IDs = %q", id1)
	}
	if id1 != "SYN-REQUIREMEN-001" {
		t.Errorf("first ID = %q, want SYN-REQUIREMEN-001", id1)
	}
}

func TestNormalizeHeading(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Requirements", "REQUIREMEN"},
		{"Short", "SHORT"},
		{"Has Spaces & Symbols!", "HASSPACESS"},
		{"", ""},
	}
	for _, tt := range tests {
		got := normalizeHeading(tt.input)
		if got != tt.want {
			t.Errorf("normalizeHeading(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestVerifiabilityLowWithoutAC(t *testing.T) {
	md := `# Requirements
- The system must do something
`
	tracker := newIDTracker()
	reqs := ExtractRequirements(md, "test.md", tracker)
	if len(reqs) != 1 {
		t.Fatalf("reqs = %d, want 1", len(reqs))
	}
	if reqs[0].Verifiability != "low" {
		t.Errorf("verifiability = %q, want low", reqs[0].Verifiability)
	}
}

func TestExtractPlanIntents(t *testing.T) {
	md := `# Component Design

- Handle user authentication
- Manage session tokens

# Architecture Overview

- Frontend serves React app
- Backend provides REST API
`
	tracker := newIDTracker()
	intents := ExtractPlanIntents(md, "plan.md", tracker)

	if len(intents) != 4 {
		t.Fatalf("intents = %d, want 4", len(intents))
	}

	if !strings.HasPrefix(intents[0].ID, "PLAN-") {
		t.Errorf("intent ID = %q, want PLAN- prefix", intents[0].ID)
	}
}

func TestPlanIDGeneration(t *testing.T) {
	tracker := newIDTracker()
	id := generatePlanID("Component Design", 0, tracker)
	if !strings.HasPrefix(id, "PLAN-COMPONENTD-001") {
		t.Errorf("plan ID = %q, want PLAN-COMPONENTD-001", id)
	}
}

func TestDuplicateExplicitID(t *testing.T) {
	tracker := newIDTracker()
	tracker.recordExplicit("REQ-001")
	tracker.recordExplicit("REQ-001")
	err := tracker.validate()
	if err == nil {
		t.Error("expected error for duplicate explicit ID")
	}
}

func TestEmptySpec(t *testing.T) {
	tracker := newIDTracker()
	reqs := ExtractRequirements("", "empty.md", tracker)
	if len(reqs) != 0 {
		t.Errorf("expected 0 reqs from empty spec, got %d", len(reqs))
	}
}

func TestNoRequirementHeadings(t *testing.T) {
	md := `# Introduction
This is a regular document.

# Usage
How to use the system.
`
	tracker := newIDTracker()
	reqs := ExtractRequirements(md, "test.md", tracker)
	if len(reqs) != 0 {
		t.Errorf("expected 0 reqs from non-requirement doc, got %d", len(reqs))
	}
}

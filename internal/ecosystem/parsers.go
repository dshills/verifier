package ecosystem

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/dshills/verifier/internal/domain"
)

// SpecCriticIssue represents a finding from SpecCritic.
type SpecCriticIssue struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Anchor   string `json:"anchor"`
}

// SpecCriticReport is the root structure of SpecCritic JSON.
type SpecCriticReport struct {
	Issues []SpecCriticIssue `json:"issues"`
}

// PlanCriticIssue represents a finding from PlanCritic.
type PlanCriticIssue struct {
	ID        string `json:"id"`
	Severity  string `json:"severity"`
	Title     string `json:"title"`
	Component string `json:"component"`
}

// PlanCriticReport is the root structure of PlanCritic JSON.
type PlanCriticReport struct {
	Issues []PlanCriticIssue `json:"issues"`
}

// RealityCheckDelta represents a delta from RealityCheck.
type RealityCheckDelta struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	SpecRef     string `json:"spec_ref"`
	CodeRef     string `json:"code_ref"`
}

// RealityCheckReport is the root structure of RealityCheck JSON.
type RealityCheckReport struct {
	Deltas []RealityCheckDelta `json:"deltas"`
}

// PrismFinding represents a finding from Prism.
type PrismFinding struct {
	ID        string `json:"id"`
	Severity  string `json:"severity"`
	File      string `json:"file"`
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
	Message   string `json:"message"`
}

// PrismReport is the root structure of Prism JSON.
type PrismReport struct {
	Findings []PrismFinding `json:"findings"`
}

// LoadSpecCritic parses SpecCritic JSON from a file.
func LoadSpecCritic(path string) (*SpecCriticReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read speccritic: %w", err)
	}
	var report SpecCriticReport
	if err := json.Unmarshal(data, &report); err != nil {
		slog.Warn("invalid speccritic JSON, skipping", "path", path, "err", err)
		return nil, nil
	}
	validateSpecCriticIssues(report.Issues)
	return &report, nil
}

// LoadPlanCritic parses PlanCritic JSON from a file.
func LoadPlanCritic(path string) (*PlanCriticReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read plancritic: %w", err)
	}
	var report PlanCriticReport
	if err := json.Unmarshal(data, &report); err != nil {
		slog.Warn("invalid plancritic JSON, skipping", "path", path, "err", err)
		return nil, nil
	}
	validatePlanCriticSeverities(report.Issues)
	return &report, nil
}

// LoadRealityCheck parses RealityCheck JSON from a file.
func LoadRealityCheck(path string) (*RealityCheckReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read realitycheck: %w", err)
	}
	var report RealityCheckReport
	if err := json.Unmarshal(data, &report); err != nil {
		slog.Warn("invalid realitycheck JSON, skipping", "path", path, "err", err)
		return nil, nil
	}
	validateRealityCheckDeltas(report.Deltas)
	return &report, nil
}

// LoadPrism parses Prism JSON from a file.
func LoadPrism(path string) (*PrismReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read prism: %w", err)
	}
	var report PrismReport
	if err := json.Unmarshal(data, &report); err != nil {
		slog.Warn("invalid prism JSON, skipping", "path", path, "err", err)
		return nil, nil
	}
	validatePrismFindings(report.Findings)
	return &report, nil
}

func validateSpecCriticIssues(issues []SpecCriticIssue) {
	for i := range issues {
		issues[i].Severity = normalizeSeverity(issues[i].Severity)
	}
}

func validatePlanCriticSeverities(issues []PlanCriticIssue) {
	for i := range issues {
		issues[i].Severity = normalizeSeverity(issues[i].Severity)
	}
}

func validateRealityCheckDeltas(deltas []RealityCheckDelta) {
	for i := range deltas {
		switch deltas[i].Kind {
		case "added", "removed", "changed", "missing":
			// valid
		default:
			slog.Warn("unknown realitycheck kind, mapping to 'changed'", "kind", deltas[i].Kind)
			deltas[i].Kind = "changed"
		}
	}
}

func validatePrismFindings(findings []PrismFinding) {
	for i := range findings {
		findings[i].Severity = normalizeSeverity(findings[i].Severity)
	}
}

func normalizeSeverity(sev string) string {
	switch sev {
	case domain.SeverityCritical, domain.SeverityHigh, domain.SeverityMedium, domain.SeverityLow:
		return sev
	default:
		if sev != "" {
			slog.Warn("unknown severity, mapping to 'low'", "severity", sev)
		}
		return domain.SeverityLow
	}
}

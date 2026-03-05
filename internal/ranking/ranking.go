package ranking

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	"github.com/dshills/verifier/internal/domain"
)

// AssignSeverity sets severity for each recommendation based on risk signals and requirement text.
func AssignSeverity(recs []domain.Recommendation, reqs *domain.RequirementSet) {
	reqMap := make(map[string]*domain.Requirement)
	if reqs != nil {
		for i := range reqs.Requirements {
			reqMap[reqs.Requirements[i].ID] = &reqs.Requirements[i]
		}
	}

	for i := range recs {
		recs[i].Severity = computeSeverity(&recs[i], reqMap)
	}
}

func computeSeverity(rec *domain.Recommendation, reqMap map[string]*domain.Requirement) string {
	risks := make(map[string]bool)
	for _, r := range rec.Covers.Risks {
		risks[r] = true
	}

	// Check requirement text for security/critical keywords
	for _, reqID := range rec.Covers.Requirements {
		if req, ok := reqMap[reqID]; ok {
			text := strings.ToLower(req.Text)
			if containsAny(text, "auth", "security", "injection", "token", "payment", "phi", "password", "credential") {
				return domain.SeverityCritical
			}
		}
	}

	// Concurrency hazards → critical
	if risks[domain.RiskConcurrency] {
		return domain.SeverityCritical
	}

	// Core functional gaps, boundary integrations → high
	if risks[domain.RiskHTTPHandler] || risks[domain.RiskDBQuery] || risks[domain.RiskBoundary] {
		return domain.SeverityHigh
	}

	// Error path and validation gaps → medium
	if risks[domain.RiskErrorPath] || risks[domain.RiskInputValidation] {
		return domain.SeverityMedium
	}

	// Complexity → medium
	if risks[domain.RiskComplexity] {
		return domain.SeverityMedium
	}

	return domain.SeverityLow
}

// AssignIDs generates TESTREC IDs for all recommendations.
func AssignIDs(recs []domain.Recommendation) {
	used := make(map[string]int)

	for i := range recs {
		rec := &recs[i]
		reqID := ""
		if len(rec.Covers.Requirements) > 0 {
			reqID = rec.Covers.Requirements[0]
		}

		hash := computeTESTRECHash(reqID, rec.Target.Name, rec.Category)
		id := fmt.Sprintf("TESTREC-%s", hash)

		used[id]++
		if used[id] > 1 {
			id = fmt.Sprintf("%s-%d", id, used[id])
		}

		rec.ID = id
	}
}

func computeTESTRECHash(reqID, symbol, category string) string {
	input := reqID + "\x00" + symbol + "\x00" + category
	h := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%X", h[:4])
}

// SortRecommendations sorts by severity desc, confidence desc, ID asc.
func SortRecommendations(recs []domain.Recommendation) {
	sort.SliceStable(recs, func(i, j int) bool {
		si := domain.SeverityOrder(recs[i].Severity)
		sj := domain.SeverityOrder(recs[j].Severity)
		if si != sj {
			return si > sj
		}
		if recs[i].Confidence != recs[j].Confidence {
			return recs[i].Confidence > recs[j].Confidence
		}
		return recs[i].ID < recs[j].ID
	})
}

// ComputeRiskScore computes min(100, critical*10 + high*5 + medium*2 + low*1).
func ComputeRiskScore(recs []domain.Recommendation) int {
	score := 0
	for _, rec := range recs {
		score += domain.SeverityWeight(rec.Severity)
	}
	if score > 100 {
		return 100
	}
	return score
}

// Truncate applies max-findings limit and returns whether truncation occurred.
func Truncate(recs []domain.Recommendation, maxFindings int) ([]domain.Recommendation, bool) {
	if maxFindings <= 0 || len(recs) <= maxFindings {
		return recs, false
	}
	return recs[:maxFindings], true
}

// CheckFailOn returns true if any recommendation exceeds the severity threshold.
func CheckFailOn(recs []domain.Recommendation, failOn string) bool {
	if failOn == "" || failOn == "none" {
		return false
	}
	threshold := domain.SeverityOrder(failOn)
	for _, rec := range recs {
		if domain.SeverityOrder(rec.Severity) >= threshold {
			return true
		}
	}
	return false
}

func containsAny(text string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

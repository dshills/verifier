package mapping

import (
	"strings"

	"github.com/dshills/verifier/internal/domain"
)

var stopWords = map[string]bool{
	"the": true, "a": true, "an": true, "is": true, "are": true,
	"for": true, "to": true, "of": true, "in": true, "with": true,
	"and": true, "or": true, "be": true, "it": true, "that": true,
	"this": true, "should": true, "must": true, "shall": true,
	"will": true, "can": true,
}

// MapRequirements maps requirements to symbols using name-based matching.
func MapRequirements(reqs *domain.RequirementSet, idx *domain.SymbolIndex) (*domain.CoverageMap, *domain.UnmappedRequirements) {
	cm := &domain.CoverageMap{}
	unmapped := &domain.UnmappedRequirements{}

	for _, req := range reqs.Requirements {
		reqTokens := tokenize(req.Text)
		var matches []symbolMatch

		for _, sym := range idx.Symbols {
			if !sym.Exported {
				continue
			}
			symTokens := tokenizeSymbol(sym.Name)
			jaccard := jaccardSimilarity(reqTokens, symTokens)

			if jaccard < 0.1 {
				// Check package proximity
				if mentionsPackage(req.Text, sym.Package) {
					matches = append(matches, symbolMatch{
						symbol:     sym.Name,
						confidence: 0.2,
					})
				}
				continue
			}

			conf := computeConfidence(jaccard, req, sym)
			matches = append(matches, symbolMatch{
				symbol:     sym.Name,
				confidence: conf,
			})
		}

		if len(matches) == 0 {
			unmapped.IDs = append(unmapped.IDs, req.ID)
			continue
		}

		var syms []string
		var bestConf float64
		for _, m := range matches {
			syms = append(syms, m.symbol)
			if m.confidence > bestConf {
				bestConf = m.confidence
			}
		}

		cm.Mappings = append(cm.Mappings, domain.CoverageMapping{
			RequirementID: req.ID,
			Symbols:       syms,
			Confidence:    bestConf,
		})
	}

	return cm, unmapped
}

type symbolMatch struct {
	symbol     string
	confidence float64
}

func computeConfidence(jaccard float64, req domain.Requirement, sym domain.Symbol) float64 {
	// Check if requirement text mentions the symbol's package
	samePackage := mentionsPackage(req.Text, sym.Package) ||
		mentionsPackage(req.HeadingContext, sym.Package)

	if jaccard >= 0.8 && samePackage {
		return 1.0
	}
	if jaccard >= 0.3 && samePackage {
		return 0.7
	}
	if jaccard >= 0.3 {
		return 0.5
	}
	if samePackage {
		return 0.2
	}
	return 0.1
}

func tokenize(text string) map[string]bool {
	tokens := make(map[string]bool)
	words := splitWords(strings.ToLower(text))
	for _, w := range words {
		if !stopWords[w] && len(w) > 1 {
			tokens[w] = true
		}
	}
	return tokens
}

func tokenizeSymbol(name string) map[string]bool {
	tokens := make(map[string]bool)
	words := splitCamelCase(name)
	for _, w := range words {
		lower := strings.ToLower(w)
		if len(lower) > 1 {
			tokens[lower] = true
		}
	}
	return tokens
}

func splitWords(text string) []string {
	var words []string
	var current strings.Builder
	for _, ch := range text {
		if isWordChar(ch) {
			current.WriteRune(ch)
		} else {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		}
	}
	if current.Len() > 0 {
		words = append(words, current.String())
	}
	return words
}

func splitCamelCase(name string) []string {
	// Also handle snake_case and kebab-case
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")

	var words []string
	var current strings.Builder

	for i, ch := range name {
		if ch == ' ' {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
			continue
		}
		if i > 0 && isUpper(ch) && current.Len() > 0 {
			words = append(words, current.String())
			current.Reset()
		}
		current.WriteRune(ch)
	}
	if current.Len() > 0 {
		words = append(words, current.String())
	}
	return words
}

func isWordChar(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') || ch == '_' || ch == '-'
}

func isUpper(ch rune) bool {
	return ch >= 'A' && ch <= 'Z'
}

func jaccardSimilarity(a, b map[string]bool) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}

	intersection := 0
	for k := range a {
		if b[k] {
			intersection++
		}
	}

	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

func mentionsPackage(text, pkgName string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(pkgName))
}

// FindUntestedIntents finds plan intents with components that have no tests.
func FindUntestedIntents(intents *domain.PlanIntentSet, tests *domain.TestInventory, graph *domain.RepoGraph) *domain.UntestedIntents {
	result := &domain.UntestedIntents{}

	testedPackages := make(map[string]bool)
	for _, t := range tests.Tests {
		testedPackages[t.Package] = true
	}

	for _, intent := range intents.Intents {
		component := strings.ToLower(intent.Component)
		hasTested := false
		if graph != nil {
			for _, pkg := range graph.Packages {
				if strings.Contains(strings.ToLower(pkg.Name), component) && testedPackages[pkg.Name] {
					hasTested = true
					break
				}
			}
		}
		if !hasTested {
			result.Intents = append(result.Intents, intent.ID)
		}
	}

	return result
}

package domain

import "time"

// Severity levels for recommendations.
const (
	SeverityCritical = "critical"
	SeverityHigh     = "high"
	SeverityMedium   = "medium"
	SeverityLow      = "low"
)

// Verifiability levels for requirements.
const (
	VerifiabilityHigh   = "high"
	VerifiabilityMedium = "medium"
	VerifiabilityLow    = "low"
)

// Test categories.
const (
	CategoryUnit        = "unit"
	CategoryIntegration = "integration"
	CategoryContract    = "contract"
	CategoryProperty    = "property"
	CategoryFuzz        = "fuzz"
	CategoryConcurrency = "concurrency"
	CategoryPerf        = "perf"
	CategorySecurity    = "security"
)

// Target kinds.
const (
	TargetFunction     = "function"
	TargetMethod       = "method"
	TargetPackage      = "package"
	TargetHTTPEndpoint = "http_endpoint"
	TargetDBQuery      = "db_query"
	TargetComponent    = "component"
)

// Risk signal types.
const (
	RiskErrorPath       = "error_path"
	RiskConcurrency     = "concurrency"
	RiskBoundary        = "boundary"
	RiskComplexity      = "complexity"
	RiskHTTPHandler     = "http_handler"
	RiskDBQuery         = "db_query"
	RiskInputValidation = "input_validation"
)

// SeverityOrder returns a numeric value for sorting (higher = more severe).
func SeverityOrder(s string) int {
	switch s {
	case SeverityCritical:
		return 4
	case SeverityHigh:
		return 3
	case SeverityMedium:
		return 2
	case SeverityLow:
		return 1
	default:
		return 0
	}
}

// SeverityWeight returns the weight for risk_score computation.
func SeverityWeight(s string) int {
	switch s {
	case SeverityCritical:
		return 10
	case SeverityHigh:
		return 5
	case SeverityMedium:
		return 2
	case SeverityLow:
		return 1
	default:
		return 0
	}
}

// PackageInfo describes a Go package in the repository.
type PackageInfo struct {
	Path      string   `json:"path"`
	Name      string   `json:"name"`
	Dir       string   `json:"dir"`
	GoFiles   []string `json:"go_files"`
	TestFiles []string `json:"test_files"`
	IsCmd     bool     `json:"is_cmd"`
	HasTests  bool     `json:"has_tests"`
}

// BoundaryEntry describes an external boundary detected in code.
type BoundaryEntry struct {
	File   string `json:"file"`
	Symbol string `json:"symbol"`
	Line   int    `json:"line"`
	Kind   string `json:"kind"` // http, db, fs, external, mq
}

// BoundaryMap contains detected external boundaries.
type BoundaryMap struct {
	HTTP     []BoundaryEntry `json:"http"`
	DB       []BoundaryEntry `json:"db"`
	FS       []BoundaryEntry `json:"fs"`
	External []BoundaryEntry `json:"external"`
	MQ       []BoundaryEntry `json:"mq"`
}

// RepoGraph describes the repository structure.
type RepoGraph struct {
	ModulePath string        `json:"module_path"`
	ModuleRoot string        `json:"module_root"`
	Packages   []PackageInfo `json:"packages"`
}

// TestInfo describes a single test function.
type TestInfo struct {
	File      string `json:"file"`
	Package   string `json:"package"`
	FuncName  string `json:"func_name"`
	IsSubtest bool   `json:"is_subtest"`
	Line      int    `json:"line"`
}

// TestInventory lists all tests found in the repository.
type TestInventory struct {
	Tests []TestInfo `json:"tests"`
}

// Requirement is a testable claim extracted from a spec.
type Requirement struct {
	ID                 string     `json:"id"`
	Text               string     `json:"text"`
	Verifiability      string     `json:"verifiability"`
	Issues             []string   `json:"issues"`
	Evidence           []Evidence `json:"evidence"`
	AcceptanceCriteria []string   `json:"acceptance_criteria,omitempty"`
	Source             string     `json:"source"`
	HeadingContext     string     `json:"heading_context"`
}

// RequirementSet is the collection of requirements from spec files.
type RequirementSet struct {
	Requirements []Requirement `json:"requirements"`
}

// PlanIntent is a structured item extracted from a plan document.
type PlanIntent struct {
	ID                string   `json:"id"`
	Component         string   `json:"component"`
	Responsibility    string   `json:"responsibility"`
	Description       string   `json:"description"`
	IntegrationPoints []string `json:"integration_points,omitempty"`
	Source            string   `json:"source"`
}

// PlanIntentSet is the collection of plan intents.
type PlanIntentSet struct {
	Intents []PlanIntent `json:"intents"`
}

// Symbol describes a Go symbol extracted from the AST.
type Symbol struct {
	Name         string `json:"name"`
	Package      string `json:"package"`
	File         string `json:"file"`
	LineStart    int    `json:"line_start"`
	LineEnd      int    `json:"line_end"`
	Kind         string `json:"kind"` // function, method, type, interface
	Exported     bool   `json:"exported"`
	Signature    string `json:"signature"`
	ReceiverType string `json:"receiver_type,omitempty"`
}

// SymbolIndex contains all symbols extracted from the codebase.
type SymbolIndex struct {
	Symbols []Symbol `json:"symbols"`
}

// RiskSignal describes a risk detected for a symbol.
type RiskSignal struct {
	Symbol  string   `json:"symbol"`
	File    string   `json:"file"`
	Package string   `json:"package"`
	Risks   []string `json:"risks"`
}

// RiskSignals contains all risk signals detected in the codebase.
type RiskSignals struct {
	Signals []RiskSignal `json:"signals"`
}

// CoverageMapping maps a requirement to code symbols.
type CoverageMapping struct {
	RequirementID string   `json:"requirement_id"`
	Symbols       []string `json:"symbols"`
	Confidence    float64  `json:"confidence"`
}

// CoverageMap is the mapping of requirements to code.
type CoverageMap struct {
	Mappings []CoverageMapping `json:"mappings"`
}

// UnmappedRequirements lists requirement IDs with no code match.
type UnmappedRequirements struct {
	IDs []string `json:"ids"`
}

// UntestedIntents lists plan intents with no corresponding tests.
type UntestedIntents struct {
	Intents []string `json:"intents"`
}

// Evidence supports a recommendation with traceable references.
type Evidence struct {
	Kind             string `json:"kind"` // spec, code, plan, test
	File             string `json:"file"`
	Anchor           string `json:"anchor,omitempty"`
	Symbol           string `json:"symbol,omitempty"`
	TruncatedContext bool   `json:"truncated_context,omitempty"`
}

// Target describes what code a test recommendation targets.
type Target struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	File      string `json:"file"`
	LineStart int    `json:"line_start,omitempty"`
	LineEnd   int    `json:"line_end,omitempty"`
}

// Covers describes what a recommendation covers.
type Covers struct {
	Requirements []string `json:"requirements,omitempty"`
	PlanItems    []string `json:"plan_items,omitempty"`
	Risks        []string `json:"risks,omitempty"`
}

// Proposal describes the recommended test approach.
type Proposal struct {
	Title      string   `json:"title"`
	Approach   string   `json:"approach"`
	Assertions []string `json:"assertions,omitempty"`
	Fixtures   []string `json:"fixtures,omitempty"`
}

// ExistingTest references a test that already exists.
type ExistingTest struct {
	File string `json:"file"`
	Name string `json:"name"`
	Gap  string `json:"gap,omitempty"`
}

// Recommendation is a proposed test or test set.
type Recommendation struct {
	ID            string         `json:"id"`
	Severity      string         `json:"severity"`
	Confidence    float64        `json:"confidence"`
	Category      string         `json:"category"`
	Target        Target         `json:"target"`
	Covers        Covers         `json:"covers"`
	Proposal      Proposal       `json:"proposal"`
	Evidence      []Evidence     `json:"evidence"`
	ExistingTests []ExistingTest `json:"existing_tests,omitempty"`
}

// PromptHash records a prompt hash for traceability.
type PromptHash struct {
	Stage string `json:"stage"`
	Hash  string `json:"hash"`
}

// InputFiles lists the input files used for analysis.
type InputFiles struct {
	SpecFiles []string `json:"spec_files"`
	PlanFiles []string `json:"plan_files"`
}

// Meta contains metadata about the analysis run.
type Meta struct {
	Tool              string       `json:"tool"`
	Version           string       `json:"version"`
	RepoRoot          string       `json:"repo_root"`
	Timestamp         string       `json:"timestamp"`
	Seed              *int         `json:"seed"`
	Mode              string       `json:"mode"`
	LLMFallbacks      []string     `json:"llm_fallbacks"`
	PromptHashes      []PromptHash `json:"prompt_hashes"`
	TruncatedPackages []string     `json:"truncated_packages"`
	Inputs            InputFiles   `json:"inputs"`
}

// CoverageSignal provides optional coverage information.
type CoverageSignal struct {
	HasCoverageData bool   `json:"has_coverage_data"`
	Statement       string `json:"statement"`
}

// Summary provides aggregate statistics about the analysis.
type Summary struct {
	RiskScore                int            `json:"risk_score"`
	TotalFindings            int            `json:"total_findings"`
	Truncated                bool           `json:"truncated"`
	MissingRecommendations   int            `json:"missing_recommendations"`
	UnverifiableRequirements int            `json:"unverifiable_requirements"`
	CoverageSignal           CoverageSignal `json:"coverage_signal"`
}

// Scaffold describes a scaffold action (placeholder for Phase 3).
type Scaffold struct {
	Path    string   `json:"path"`
	Actions []string `json:"actions"`
	Note    string   `json:"note,omitempty"`
}

// Report is the top-level output structure.
type Report struct {
	Meta            Meta             `json:"meta"`
	Summary         Summary          `json:"summary"`
	Requirements    []Requirement    `json:"requirements"`
	Recommendations []Recommendation `json:"recommendations"`
	Scaffolds       []Scaffold       `json:"scaffolds"`
}

// Artifacts is the container passed between pipeline stages.
type Artifacts struct {
	RepoGraph            *RepoGraph
	BoundaryMap          *BoundaryMap
	TestInventory        *TestInventory
	RequirementSet       *RequirementSet
	PlanIntentSet        *PlanIntentSet
	SymbolIndex          *SymbolIndex
	RiskSignals          *RiskSignals
	CoverageMap          *CoverageMap
	UnmappedRequirements *UnmappedRequirements
	UntestedIntents      *UntestedIntents
	Recommendations      []Recommendation
	HasOpenAPI           bool
}

// Config holds the merged configuration from file + CLI flags.
type Config struct {
	Mode        string        `yaml:"mode"`
	Format      string        `yaml:"format"`
	Root        string        `yaml:"-"`
	SpecPaths   []string      `yaml:"spec"`
	PlanPaths   []string      `yaml:"plan"`
	Exclude     []string      `yaml:"exclude"`
	Include     []string      `yaml:"include"`
	FailOn      string        `yaml:"-"`
	Seed        *int          `yaml:"-"`
	MaxFindings int           `yaml:"-"`
	Timeout     time.Duration `yaml:"-"`
	ConfigPath  string        `yaml:"-"`
	LLM         LLMConfig     `yaml:"llm"`
	CI          CIConfig      `yaml:"ci"`

	// External tool inputs (Phase 4)
	SpecCriticPath   string `yaml:"-"`
	PlanCriticPath   string `yaml:"-"`
	RealityCheckPath string `yaml:"-"`
	PrismPath        string `yaml:"-"`
}

// CIConfig holds CI-specific configuration.
type CIConfig struct {
	FailOn string `yaml:"fail_on"`
}

// LLMConfig holds LLM provider configuration.
type LLMConfig struct {
	Provider    string  `yaml:"provider"`
	Model       string  `yaml:"model"`
	Temperature float64 `yaml:"temperature"`
	MaxTokens   int     `yaml:"max_tokens"`
}

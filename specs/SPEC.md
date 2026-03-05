Verifier — SPEC.md

1. Overview

Verifier is a Go CLI tool that analyzes a repository and answers:
	•	What tests should exist to verify the specification and implementation plan?
	•	Which tests are missing, weak, or misaligned with intent?
	•	Which requirements are not verifiable (vague, untestable, or missing acceptance criteria)?
	•	What test strategy is appropriate (unit, integration, contract, property, fuzz, concurrency, perf, security)?

Verifier is designed to be composable in an agentic workflow with:
	•	SpecCritic (spec quality)
	•	PlanCritic (plan sanity)
	•	RealityCheck (intent enforcement)
	•	Prism (multi-LLM code review)

Verifier focuses on test design and gaps, not generating perfect tests (it may scaffold, but analysis is core).

⸻

2. Goals

2.1 Primary Goals
	1.	Produce a structured test gap report grounded in:
	•	SPEC.md requirements
	•	PLAN.md architectural intent
	•	source code realities (APIs, error paths, concurrency, boundaries)
	•	existing tests (presence/quality/coverage hints)
	2.	Recommend test types and test targets, including:
	•	unit
	•	integration
	•	contract/API
	•	property-based
	•	fuzzing
	•	concurrency/race
	•	security checks
	•	performance tests (only if spec defines measurable criteria)
	3.	Output results as:
	•	human-readable report
	•	machine-readable JSON (for agents/CI)
	4.	Provide deterministic operation:
	•	In offline mode: output is fully deterministic for identical inputs.
	•	In LLM mode: output ordering is deterministic; content may vary unless the provider supports seed-based sampling and --seed is set.
	•	explicit IDs for requirements and test recommendations
	•	reproducible runs with --seed and consistent formatting

2.2 Secondary Goals
	•	Optional: generate test scaffolds (skeleton files with TODOs).
	•	Optional: produce a “test plan” document (markdown) for the repo.

⸻

3. Non-Goals
	•	Not a replacement for go test, coverage tooling, or mutation testing.
	•	Not a full security scanner.
	•	Not a full architecture reverse-engineering system.
	•	Not guaranteed to infer perfect acceptance criteria from vague specs. Instead it flags vagueness.

⸻

4. Key Concepts

4.1 Requirements

Verifier treats specs as testable claims. A requirement must have:
	•	unique ID (preferred: REQ-###, FR-###, etc.)
	•	statement
	•	acceptance criteria (explicitly measurable/observable)

If acceptance criteria are missing, Verifier flags the requirement with verifiability: "low" and suggests clarifications.

4.2 Evidence

Verifier attaches evidence to every recommendation:
	•	spec excerpts / requirement IDs
	•	plan sections / component references
	•	code evidence (file path + symbol + line range when possible)
	•	existing tests evidence (test name + file)

4.3 Test Recommendation

A recommendation is a proposed test or test set to cover a requirement/plan intent/risky code path.

⸻

5. Inputs

5.1 Repository
	•	root directory (default .)
	•	source language: Go only. Non-Go source files are ignored for AST analysis but noted in the repo inventory.

5.2 Spec & Plan Files

Defaults:
	•	SPEC.md (or specs/SPEC.md)
	•	PLAN.md (or plans/PLAN.md)

Can be overridden:
	•	--spec path1,path2,...
	•	--plan path1,path2,...

Multi-file handling:
	•	Deduplication occurs before path resolution: duplicate paths in the comma-separated list are deduplicated silently, then each unique path is validated.
	•	If any path (whether default or override) resolves to a directory rather than a file, or if a file exists but cannot be read (e.g., permission denied), Verifier exits immediately with code 1 and an error message before any analysis begins. This takes precedence over the missing-file warning behavior.
	•	If multiple spec files define the same requirement ID, Verifier exits with code 1 and reports the conflict. Requirement IDs must be unique across all input spec files (both explicit and synthetic IDs).

File resolution behavior:
	•	If no spec file is found at the default or override path, Verifier emits a warning to stderr and continues in degraded offline mode with code-only analysis (Stages A, C, F, G only). Requirement-based analysis (Stages B, D, E) is skipped for spec. In degraded offline mode, Stage F operates only on code-heuristic signals from Stage C (zero-test packages, untested exported symbols, error paths without tests, concurrency without race tests) and does not perform requirement-based gap analysis. Stage G ranks only the findings produced by degraded Stage F.
	•	If no plan file is found, Verifier emits a warning and skips plan-based analysis. This is not an error.
	•	If both --spec and --plan resolve to missing files, Verifier still runs in degraded offline mode as described above.
	•	Non-Go source files are ignored for AST analysis. Their presence is noted in the repo inventory but does not affect recommendations.

5.3 Optional Artifacts
	•	README.md for context
	•	OpenAPI / Swagger files (openapi.yaml, swagger.json)
	•	ADRs (adr/*.md)
	•	Architecture docs (docs/architecture.md)
	•	CI config (.github/workflows/*.yml) for test expectations

5.4 Model Configuration (Optional)

Verifier can run in:
	•	offline mode: static heuristics + Go AST analysis
	•	LLM mode: uses one or more model providers

LLM is recommended for:
	•	extracting requirements and mapping to code semantics
	•	proposing test designs beyond naive heuristics

LLM failure behavior:
	•	If --mode llm is set but no provider or model is configured (via config file or CLI flags), Verifier exits with code 1 and a descriptive error message. This is a configuration error, not a provider error.
	•	If an LLM stage fails at runtime (timeout, malformed response, provider error), Verifier falls back to offline heuristics for that stage and logs a warning to stderr.
	•	If the LLM returns malformed JSON for an internal stage, Verifier retries once. If the retry also fails, it falls back to offline mode for that stage.
	•	Partial LLM results are valid: some stages may use LLM output while others fall back to offline. The JSON meta.mode field reports "llm" if any stage used the LLM, and a new meta.llm_fallbacks array lists stages that fell back to offline.
	•	If --seed is set, it is passed to the LLM provider if the provider supports it. If the provider does not support seeds, the seed is used only for deterministic tie-breaking in output ordering. This is logged as a warning.

⸻

6. Outputs

6.1 Human Report (Default)
	•	Format is controlled by --format (text|md|json); default is md.
	•	grouped by severity and test category
	•	actionable bullets
	•	includes requirement mapping + evidence pointers

6.2 JSON Output (Required for CI/agents)

--format json outputs a single JSON object with stable schema.

JSON Schema (Conceptual)

{
  "meta": {
    "tool": "verifier",
    "version": "0.1.0",  // injected at build time; reflects the binary's release version
    "repo_root": ".",
    "timestamp": "RFC3339",
    "seed": 123,
    "mode": "offline|llm",
    "llm_fallbacks": [],
    "prompt_hashes": [],
    "truncated_packages": [],
    "inputs": {
      "spec_files": ["SPEC.md"],
      "plan_files": ["PLAN.md"]
    }
  },
  "summary": {
    "risk_score": 0,
    "total_findings": 0,
    "truncated": false,
    "missing_recommendations": 0,
    "unverifiable_requirements": 0,
    "coverage_signal": {
      "has_coverage_data": false,
      "statement": "optional hint only"
    }
  },
  "requirements": [
    {
      "id": "REQ-12",
      "text": "API returns within 200ms P99",
      "verifiability": "high|medium|low",
      "issues": ["missing_acceptance_criteria"],
      "evidence": [{ "kind": "spec", "file": "SPEC.md", "anchor": "..." }]
    }
  ],
  "recommendations": [
    {
      "id": "TESTREC-A1B2C3D4",
      "severity": "critical|high|medium|low",
      "confidence": 0.0,
      "category": "unit|integration|contract|property|fuzz|concurrency|perf|security",
      "target": {
        "kind": "function|method|package|http_endpoint|db_query|component",
        "name": "UserService.CreateUser",
        "file": "pkg/user/service.go",
        "line_start": 120,
        "line_end": 240
      },
      "covers": {
        "requirements": ["REQ-4"],
        "plan_items": ["PLAN-3.2"],
        "risks": ["error_path", "boundary_condition"]
      },
      "proposal": {
        "title": "CreateUser rejects invalid email",
        "approach": "table-driven unit test with invalid inputs",
        "assertions": ["returns validation error", "does not write to DB"],
        "fixtures": ["fake dao", "in-memory repo"]
      },
      "evidence": [
        { "kind": "spec", "file": "SPEC.md", "anchor": "REQ-4" },
        { "kind": "code", "file": "pkg/user/service.go", "symbol": "CreateUser" }
      ],
      "existing_tests": [
        { "file": "pkg/user/service_test.go", "name": "TestCreateUser_HappyPath", "gap": "missing_invalid_inputs" }
      ]
    }
  ],
  "scaffolds": []
}

Notes on JSON schema fields:
	•	risk_score: integer 0–100. Computed as: min(100, critical_count*10 + high_count*5 + medium_count*2 + low_count*1), where counts are derived from the full pre-truncation set of recommendations grouped by their severity field (not the truncated array). A score of 0 means no recommendations.
	•	confidence: float 0.0–1.0 per recommendation. Measures strength of mapping evidence. In offline mode: 1.0 = exact name match in same package, 0.7 = partial name match in same package, 0.5 = partial name match in different package, 0.2 = package proximity only, 0.1 = heuristic guess with no direct match. In LLM mode, the model is prompted to include a "confidence" float field (0.0–1.0) in its JSON response for each recommendation. If the model does not provide it, confidence defaults to 0.5.
	•	total_findings and truncated: when --max-findings is reached, findings are sorted by severity desc and truncated. total_findings reports the pre-truncation count and truncated is set to true.
	•	scaffolds: always an empty array in verifier analyze output. Scaffold data is only produced by verifier scaffold --format json.


⸻

7. CLI Interface

7.1 Commands

verifier analyze
Primary command.

Examples:
	•	verifier analyze
	•	verifier analyze --spec SPEC.md --plan PLAN.md --format json
	•	verifier analyze --mode llm --provider openai --model gpt-5 --format md

Key flags:
	•	--root <dir> (default .)
	•	--spec <paths> comma-separated
	•	--plan <paths> comma-separated
	•	--format text|md|json (default md)
	•	--mode offline|llm (default offline)
	•	--config <file> default .verifier.yaml
	•	--fail-on <severity> (none|low|medium|high|critical; default: none) for CI gating. When "none" (the default), exit code 2 is never returned.
	•	--seed <int> Valid range: 0 to 2^31-1 (for cross-platform compatibility). Values outside this range cause exit code 1. In offline mode, used as PRNG seed for deterministic tie-breaking in output ordering. In LLM mode, passed to the provider if supported; otherwise used only for ordering (with a warning logged).
	•	--max-findings <int> default 200. Values <= 0 mean unlimited (no truncation). When the limit is reached, findings are sorted using the full sort key from Section 12.4 (severity desc, confidence desc, stable ID asc) then truncated; the JSON summary includes "truncated": true and "total_findings" with the pre-truncation count; a warning is emitted to stderr.
	•	--include <glob> / --exclude <glob>. When a file matches both, --exclude takes precedence. Syntactically invalid glob patterns cause exit code 1 with an error message identifying the invalid pattern.
	•	--timeout <duration> for LLM mode. Accepts Go time.ParseDuration format (e.g., "30s", "5m"). Default: 2m. Values <= 0 are treated as an error (exit code 1). When the timeout fires on any LLM stage, that stage falls back to offline heuristics (same as provider error behavior).

verifier scaffold
Optional. Writes skeleton tests and/or TODO blocks.
	•	--dry-run (default true unless --write)
	•	--write actually modifies files
	•	--style go-testify|std (default std, keep dep-free by default)
	•	--package-layout flat|standard (default standard). "flat" places all scaffold files in the package directory. "standard" respects internal/pkg/cmd directory conventions and creates test files adjacent to source files within those directories.
	•	--limit <n> only scaffold top N critical gaps. If N exceeds the number of available critical gaps, Verifier scaffolds all available critical gaps with no error. If zero critical gaps exist, Verifier emits a notice to stderr and exits with code 0 without writing any files.

verifier init
Generates .verifier.yaml with the defaults shown in Section 8. If .verifier.yaml already exists, Verifier exits with code 1 without overwriting. Use --force to overwrite an existing config file.

verifier explain <TESTREC-ID>
Prints deep explanation for a specific recommendation. Data source: reads from a JSON file specified by --input <file>, or from stdin if piped. If both --input and stdin are provided, --input takes precedence and stdin is ignored; a notice is logged to stderr. If neither is provided, Verifier re-runs analysis using current config/flags to locate the recommendation. If the re-run analysis itself fails (config error, IO error, etc.), Verifier exits with code 1 and prints the underlying error to stderr. TESTREC-ID lookup is exact-match only. If the TESTREC-ID is not found in the results, Verifier prints an error message to stderr suggesting that the underlying requirement/target/category may have changed, and exits with code 1.

7.2 Exit Codes
	•	0: success; no gating triggered
	•	2: analysis success, but --fail-on threshold exceeded. Exit code 2 is returned if any recommendation exists with severity >= the specified --fail-on level. Severity ordering: low < medium < high < critical.
	•	1: runtime error (bad config, parse error, IO). Note: a provider error that successfully falls back to offline mode does NOT produce exit code 1. Exit code 1 is only for errors that prevent any analysis from completing.

⸻

8. Configuration File

.verifier.yaml example:

mode: offline
format: md
spec:
  - SPEC.md
plan:
  - PLAN.md
exclude:
  - "**/vendor/**"
  - "**/node_modules/**"
  - "**/*.gen.go"
ci:
  fail_on: high
llm:
  provider: openai
  model: gpt-5
  temperature: 0.2
  max_tokens: 8000

Notes:
	•	Keep config minimal.
	•	CLI flags override config. For list-type values (spec, plan, exclude), CLI flags replace (not merge with) the corresponding config file values.
	•	If the config file exists but contains invalid YAML (parse error), Verifier exits with code 1 and an error message identifying the parse error.
	•	If the config file does not exist at the default path (.verifier.yaml), Verifier continues with built-in defaults (no error). If --config explicitly specifies a file that does not exist, Verifier exits with code 1.
	•	All validation rules for fields (e.g., temperature range 0.0–2.0) apply equally to values sourced from the config file.

⸻

9. Analysis Pipeline

Verifier has a multi-stage pipeline. Each stage produces artifacts used downstream. Stages execute sequentially in order: A → B → C → D → E → F → G. In degraded offline mode (no spec/plan), the order is A → C → F → G, where Stage F consumes only Stage A (TestInventory) and Stage C (SymbolIndex, RiskSignals) artifacts. If Stage C is also skipped (no go.mod found), Stage F operates solely on Stage A (TestInventory) artifacts and produces only package-level findings (zero-test packages); it emits a warning that code-semantic signals are unavailable.

9.1 Stage A — Repository Inventory
	•	Detect Go module root (go.mod). If no go.mod is found, Verifier emits a warning and skips Stage C (Go AST analysis). Note: if go.mod is absent AND spec/plan files are also absent, no meaningful analysis can be performed. In this case, Verifier exits with code 0 and emits a warning that no analyzable inputs were found. This takes precedence over the degraded offline mode behavior described in Section 5.2.
	•	List packages, commands, internal/pkg structure
	•	Identify test files (*_test.go)
	•	Identify entrypoints (cmd/*, main.go)
	•	Identify external boundaries:
	•	HTTP routes (common routers)
	•	DB drivers (sql.DB usage)
	•	message queues
	•	file IO
	•	third-party APIs

Artifacts:
	•	RepoGraph (packages, imports, exported APIs)
	•	BoundaryMap (http/db/fs/external)
	•	TestInventory

9.2 Stage B — Spec & Plan Extraction

Goal: Extract structured items.

Spec extraction
	•	Requirements list with IDs, text, acceptance criteria
	•	Non-functional requirements (NFRs)
	•	invariants
	•	error handling policies
	•	security/privacy claims

Plan extraction
	•	components/modules
	•	responsibilities
	•	integration points
	•	data flows
	•	“done” criteria if present

Artifacts:
	•	RequirementSet
	•	PlanIntentSet

Offline mode heuristic rules for requirement extraction:
	•	A requirement is any numbered or bulleted list item under a heading containing "requirement", "must", "shall", or "feature".
	•	Explicit IDs are detected via the pattern: REQ-\d+, FR-\d+, NFR-\d+, or similar alphanumeric-dash-number prefixes.
	•	If no explicit ID is found, Verifier assigns a synthetic ID with the prefix "SYN-" to avoid collision with user-defined IDs. The ID is formed as: "SYN-" + nearest parent heading (strip non-alphanumeric characters, uppercase, truncate to 10 chars) + "-" + zero-padded 3-digit item index within that heading (e.g., heading "5. Inputs" becomes "SYN-5INPUTS-001"). If a collision still occurs (identical truncated heading prefix + same index), a numeric suffix is appended starting at 2 for the second occurrence: the first occurrence keeps the base form (e.g., "SYN-5INPUTS-001"), the second becomes "SYN-5INPUTS-001-2", the third "SYN-5INPUTS-001-3", and so on.
	•	Acceptance criteria are detected as sub-items under a requirement, or items containing keywords: "accept", "criteria", "measurable", "verify", "assert".
	•	If no acceptance criteria are found for a requirement, it is flagged as verifiability: "low".

LLM mode: uses model to structure requirements and plan intents more reliably than heuristics.

9.3 Stage C — Code Semantics Extraction
	•	Parse Go AST for:
	•	exported functions/methods
	•	error returns
	•	branching complexity signals
	•	goroutine usage / channels / mutex
	•	HTTP handlers (heuristic signatures)
	•	DB query sites (QueryContext, ExecContext)
	•	input validation patterns

Artifacts:
	•	SymbolIndex
	•	RiskSignals (per symbol/package)

9.4 Stage D — Mapping

Core: map requirements/plan items to code symbols and boundaries.

Outputs:
	•	CoverageMap (conceptual coverage: requirement→symbols)
	•	UnmappedRequirements (no matching code; possible missing implementation)
	•	UntestedIntents (plan says integration exists; no tests found)

LLM mode improves mapping quality by reading spec+plan+code summaries.

9.5 Stage E — Test Strategy Synthesis

For each mapped requirement/intent/risk:
	•	Determine appropriate test category:
	•	unit: pure logic, validation, formatting
	•	integration: boundary between components, DB, external services
	•	contract: API behavior, OpenAPI conformance
	•	property: invariants, idempotency, serialization round-trip
	•	fuzz: parsers, decoders, validators
	•	concurrency: shared state, parallelism
	•	perf: only when acceptance criteria exist
	•	security: authz/authn, injection, unsafe deserialization
	•	Determine recommended test structure:
	•	table-driven tests
	•	golden files
	•	subtests
	•	test fixtures
	•	fake/mocks (dependency-free default: handwritten fakes)

9.6 Stage F — Gap Detection

Compare recommended tests to existing tests:
	•	Are there tests in the same package?
	•	Are there tests named/structured similarly?
	•	Do tests cover negative paths?
	•	Do tests assert the key acceptance criteria?

Heuristic signals:
	•	tests only cover happy path
	•	errors not asserted
	•	no tests around boundary components
	•	concurrency primitives without race tests

9.7 Stage G — Ranking and Reporting

Each recommendation gets:
	•	severity score (critical/high/medium/low)
	•	confidence score (how strong is mapping evidence)
	•	rationale
	•	evidence list

Ranking rules (rough):
	•	critical: security/authn/authz requirements missing tests; payment/PHI; data loss risks; concurrency hazards
	•	high: core functional requirements missing tests; boundary integrations untested
	•	medium: error path gaps; validation gaps
	•	low: minor edge cases; refactor-safety improvements

⸻

10. Integration With Your Toolchain

Verifier should be designed to “snap” into the same workflow patterns.

10.1 Standard IO Contracts
	•	Read files from repo.
	•	All report output (human-readable and JSON) is written exclusively to stdout.
	•	All warnings, notices, and error messages are written exclusively to stderr.
	•	No interactive prompts by default.
	•	JSON output stable for machine parsing.

10.2 Using Outputs From Others
	•	If speccritic produces structured findings JSON, Verifier can optionally ingest it:
	•	--speccritic findings.json
	•	it prioritizes requirements labeled “vague/unverifiable” and includes them
	•	If plancritic produces plan critique JSON:
	•	--plancritic findings.json
	•	it treats flagged plan risks as test targets
	•	If realitycheck produces intent deltas:
	•	--realitycheck report.json
	•	it generates tests to enforce corrected intent (especially regressions)
	•	If prism outputs review JSON:
	•	--prism review.json
	•	it promotes risky code sections to test recs

All of these are optional; Verifier stands alone with SPEC/PLAN/code.

External tool JSON input contracts:

Verifier treats each external tool's JSON as an opaque document and extracts only the following fields from the root-level JSON object. Additional top-level keys are permitted and ignored. If the file is not valid JSON (parse error), is empty, or a required field is missing, Verifier logs a warning to stderr and skips that input file. This is not a fatal error; analysis continues without that input.

SpecCritic (--speccritic):
	•	Root structure: {"issues": [...], ...}
	•	Required: issues[] array at root level. Each issue must have: id (string), severity (string: "critical"|"warn"|"info"), title (string).
	•	Optional per issue: recommendation (string), anchor (string: a requirement ID matching a REQ-\d+/FR-\d+/NFR-\d+ pattern, or a markdown heading anchor; if it does not match any known requirement, the boost is skipped for that issue).
	•	Verifier uses issues with severity "critical" or "warn" to boost related requirements: recommendations linked to a boosted requirement have their severity promoted by one level (low→medium, medium→high, high→critical; critical stays critical).

PlanCritic (--plancritic):
	•	Root structure: {"issues": [...], ...}
	•	Required: issues[] array at root level. Each issue must have: id (string), severity (string: "critical"|"high"|"medium"|"low"), title (string). Verifier maps PlanCritic severity values directly to its internal severity scale. Unknown severity values are treated as "low" with a warning logged to stderr.
	•	Optional per issue: recommendation (string), component (string).
	•	Verifier treats flagged plan risks as additional test targets.

RealityCheck (--realitycheck):
	•	Root structure: {"deltas": [...], ...}
	•	Required: deltas[] array at root level. Each delta must have: id (string), kind (string: "added"|"removed"|"changed"|"drift"; unknown values treated as "changed" with a warning), description (string).
	•	Optional per delta: spec_ref (string), code_ref (string).
	•	Verifier generates regression test recommendations for each delta. The TESTREC ID for RealityCheck-sourced recommendations uses "RC-{delta.id}" as the requirementID, the code_ref (or "unknown" if absent) as the targetSymbol, and "integration" as the default category.

Prism (--prism):
	•	Root structure: {"findings": [...], ...}
	•	Required: findings[] array at root level. Each finding must have: id (string), severity (string: "critical"|"high"|"medium"|"low"; unknown values treated as "low" with a warning), file (string).
	•	Optional per finding: line_start (int), line_end (int), message (string).
	•	Verifier promotes flagged code sections to test recommendations. The TESTREC ID for Prism-sourced recommendations uses "PRISM-{finding.id}" as the requirementID, the file path (or file:line_start if line_start is present) as the targetSymbol, and "unit" as the default category. Severity is mapped directly from the Prism finding severity.

10.3 Agent-Friendly Workflow

Example pipeline:

speccritic analyze --format json > speccritic.json
plancritic analyze --format json > plancritic.json
realitycheck verify --format json > reality.json
prism review --format json > prism.json

verifier analyze \
  --spec SPEC.md --plan PLAN.md \
  --speccritic speccritic.json \
  --plancritic plancritic.json \
  --realitycheck reality.json \
  --prism prism.json \
  --format json \
  --fail-on high


⸻

11. Scaffolding Behavior

When verifier scaffold --write is enabled:
	•	Never deletes files.
	•	When modifying an existing file, Verifier uses an append strategy: new test functions are appended to the end of the existing file without modifying existing content. A .bak copy is created before any modification. If a .bak file already exists, it is overwritten silently. If .bak creation fails (permissions, disk full), the scaffold operation aborts with exit code 1 and the original file is not modified.
	•	If an existing *_test.go file exists in the same package and the target function does not already have a test function in that file, Verifier appends the new test to that file.
	•	If the target function already has a test in the existing file, Verifier skips scaffolding for that target and logs a notice to stderr.
	•	If no *_test.go file exists in the package, Verifier creates a new <package>_test.go file in the same directory.

Scaffold content rules:
	•	Keep as testing package default (dependency-free).
	•	Handwritten fakes preferred over third-party mocks.
	•	Clear TODO markers with TESTREC-#### IDs so a coding agent can fill in.

TESTREC ID stability:
	•	TESTREC IDs are content-addressed: derived from SHA-256 of the UTF-8 string "{requirementID}\x00{targetSymbol}\x00{category}". The ID format is "TESTREC-" followed by the first 8 uppercase hex characters of the hash (e.g., TESTREC-A1B2C3D4).
	•	TESTREC IDs must be unique within a single analysis run. If a hash collision occurs (same first 8 hex chars for different inputs), a numeric suffix is appended (e.g., TESTREC-A1B2C3D4-2).
	•	The same requirement/target/category combination always produces the same TESTREC ID across runs, ensuring scaffold TODO markers remain valid and verifier explain <TESTREC-ID> works for previously generated IDs.

⸻

12. Implementation Requirements (Go)

12.1 Constraints
	•	Go 1.22+
	•	Zero non-stdlib dependencies in the base binary; at most one CLI parsing library is permitted as an explicit exception.
	•	Must run on macOS/Linux/Windows
	•	Individual Go files larger than 1MB are skipped for AST parsing with a warning.
	•	No hard limit on repository size or package count, but LLM mode truncates code summaries to fit within the model’s context window. Truncation strategy: packages are prioritized by risk signals (highest risk first); lowest-risk packages are dropped first. Recommendations for truncated packages fall back to offline heuristics and include a "truncated_context": true flag in their evidence. Truncation is reported in JSON meta.truncated_packages (array of dropped package paths) and a warning is emitted to stderr.

12.2 Packages (Non-normative, Suggested)

/cmd/verifier        main + CLI wiring
/internal/config     config parsing + defaults
/internal/repo        repo scanning, globs
/internal/parse       markdown extraction (offline)
/internal/golang      Go AST analysis
/internal/map         requirement/plan <-> code mapping
/internal/recommend   recommendation synthesis + ranking
/internal/report      output formatting (md/json/text)
/internal/llm         provider interface + prompts (optional)

12.3 Provider Interfaces

Define a minimal LLM interface:

type LLM interface {
  Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}

type CompletionRequest struct {
  Model       string    // model identifier (e.g., "gpt-4", "claude-sonnet-4-20250514")
  Messages    []Message // conversation messages (role + content)
  Temperature float64   // sampling temperature (default 0.2, valid range 0.0–2.0; out-of-range values cause exit code 1)
  MaxTokens   int       // maximum response tokens
  Seed        *int      // optional seed for deterministic sampling (provider-dependent)
}

type Message struct {
  Role    string // "system", "user", or "assistant"
  Content string
}

type CompletionResponse struct {
  Content    string // the model's text response
  PromptHash string // SHA-256 hash of the prompt for traceability
  Usage      Usage
}

type Usage struct {
  PromptTokens     int
  CompletionTokens int
}

Support multi-provider later; start with OpenAI/Anthropic/Ollama as optional modules behind build tags to keep the base binary clean.

12.4 Determinism
	•	Sort all findings by: severity desc, confidence desc, stable ID asc (lexicographic byte order on the full ID string, regardless of ID origin).
	•	In offline mode: output is fully deterministic for identical inputs and seed.
	•	In LLM mode:
	•	set low temperature default (0.2)
	•	pass --seed to provider if supported; otherwise use for ordering only
	•	log prompt hashes in JSON meta.prompt_hashes array for traceability

⸻

13. Prompting Strategy (LLM Mode)

LLM mode should be multi-step and bounded.

Suggested prompt stages:
	1.	Extract requirements from SPEC (IDs + acceptance criteria)
	2.	Extract plan intents (components + integrations)
	3.	Summarize code signals (Verifier can generate summaries itself; model only sees slices)
	4.	Map requirements/intents to code symbol candidates
	5.	Propose tests with categories and assertions
	6.	Compare against existing tests inventory and produce gaps

Guardrails:
	•	Model must cite evidence anchors (file + heading + symbol)
	•	Model must not hallucinate file paths; it can propose “unknown” if not found
	•	Model output must be strict JSON for internal stages; Verifier validates it

⸻

14. Quality Bar

Verifier must be useful even in offline mode.

Minimum useful offline results:
	•	flags packages with zero tests
	•	flags exported APIs with no test adjacency
	•	flags error paths and boundary calls with no tests
	•	flags concurrency usage with no race tests
	•	flags spec requirements that lack measurable criteria (basic heuristics)

LLM mode is expected to improve (non-normative goal, not a testable requirement):
	•	requirement extraction quality
	•	mapping accuracy
	•	test design recommendations

⸻

15. Example Report (Markdown)

Expected style:
	•	Summary block
	•	Critical gaps
	•	High gaps
	•	Unverifiable requirements
	•	Appendix: mapping table (req→symbols)

⸻

16. Roadmap Phases

Phase 1 (MVP — Offline)
	•	repo scan, Go AST scan
	•	basic spec/plan extraction via heuristics
	•	test inventory + gap heuristics
	•	markdown + JSON output
	•	CI gating

Phase 2 (LLM Assist)
	•	requirement + plan extraction via LLM
	•	mapping improvements
	•	better recommendations + evidence

Phase 3 (Scaffold)
	•	safe test scaffolding with IDs
	•	minimal edits / append-only strategies

Phase 4 (Ecosystem)
	•	optional ingestion of SpecCritic/PlanCritic/RealityCheck/Prism JSON
	•	cross-tool correlation in report

⸻

17. Naming and Branding Notes

Tool name: Verifier

One-liner:

“Spec-driven test strategy analysis for real codebases.”

Repo description (<=350 chars):

Verifier is a Go CLI that reads your specs, implementation plans, and codebase to determine what tests should exist—and which ones are missing. It recommends unit/integration/contract/property/fuzz/concurrency/perf/security tests with evidence, designed for agentic workflows.


Verifier — PLAN.md

1. Overview

This plan implements Verifier as described in SPEC.md across four phases. Phase 1 (MVP Offline) is broken into seven sub-phases to keep each step testable and reviewable. Later phases build incrementally on Phase 1's foundation.

Estimated structure: ~15 packages, ~8,000–12,000 lines of Go (Phase 1), stdlib-only plus one YAML parser exception.

---

2. Dependency and Build Decisions

2.1 Module Setup
- Go 1.22+ with module path github.com/dshills/verifier
- Single binary target: cmd/verifier

2.2 Dependencies
- stdlib only for all core logic (go/ast, go/parser, go/token, encoding/json, crypto/sha256, os, path/filepath, regexp, sort, strings, text/template, flag, io, fmt, time)
- Exception: gopkg.in/yaml.v3 for .verifier.yaml parsing. The SPEC permits "at most one CLI parsing library" as an exception; this plan uses that single exception slot for YAML parsing instead, since stdlib flag is sufficient for CLI parsing.
- LLM provider packages (Phase 2) gated behind build tags to keep base binary clean.

2.3 Version Injection
- Build-time injection via -ldflags for the version string used in JSON meta.version.
- Makefile or go build script with: -ldflags "-X main.version=$(VERSION)"

---

3. Package Layout

```
cmd/verifier/              main.go — CLI entry, flag parsing, command dispatch
internal/
  config/                  Config loading, validation, CLI-flag merging
  pipeline/                Stage orchestrator, degraded mode logic
  domain/                  All shared types: Requirement, PlanIntent, Symbol,
                           Recommendation, Evidence, Report, etc.
  repo/                    Stage A — repository scanning, go.mod detection,
                           package listing, test inventory, boundary detection
  parse/                   Stage B — markdown parsing, requirement extraction
                           heuristics, synthetic ID generation, plan extraction
  golang/                  Stage C — Go AST analysis, symbol indexing,
                           risk signal computation
  mapping/                 Stage D — requirement-to-symbol mapping,
                           confidence scoring
  strategy/                Stage E — test category assignment, test structure
                           recommendation
  gaps/                    Stage F — gap detection, existing test comparison
  ranking/                 Stage G — severity assignment, TESTREC ID generation,
                           sorting, risk_score computation
  report/                  Output formatting: JSON, markdown, text
  llm/                     Phase 2 — LLM interface, prompt construction,
                           response parsing
  scaffold/                Phase 3 — test file generation, append strategy,
                           .bak management
  ecosystem/               Phase 4 — external tool JSON ingestion
                           (SpecCritic, PlanCritic, RealityCheck, Prism)
```

---

4. Domain Types

Defined in internal/domain/ early in Phase 1a so all packages share a common vocabulary. These types map directly to the SPEC.md JSON schema and pipeline artifacts.

4.1 Pipeline Artifacts

```
RepoGraph           — packages []PackageInfo, imports map, module root path
PackageInfo         — path, name, dir, goFiles []string, testFiles []string,
                      isCmd bool, hasTests bool
BoundaryMap         — http []BoundaryEntry, db []BoundaryEntry,
                      fs []BoundaryEntry, external []BoundaryEntry
BoundaryEntry       — file, symbol, line, kind string
TestInventory       — tests []TestInfo
TestInfo            — file, package, funcName string, isSubtest bool

RequirementSet      — requirements []Requirement
Requirement         — id, text string, verifiability string (high|medium|low),
                      issues []string,
                      evidence []Evidence,
                      acceptanceCriteria []string,
                      source string (spec file path),
                      headingContext string
PlanIntentSet       — intents []PlanIntent
PlanIntent          — id, component, responsibility, description string,
                      integrationPoints []string,
                      source string

SymbolIndex         — symbols []Symbol
Symbol              — name, package, file string, lineStart, lineEnd int,
                      kind string (function|method|type|interface),
                      exported bool,
                      signature string,
                      receiverType string (for methods)
RiskSignals         — signals []RiskSignal
RiskSignal          — symbol string, file string, package string,
                      risks []string (error_path, concurrency, boundary,
                      complexity, http_handler, db_query, input_validation)

CoverageMap         — mappings []CoverageMapping
CoverageMapping     — requirementID string, symbols []string, confidence float64
UnmappedRequirements — ids []string
UntestedIntents     — intents []string

Recommendation      — (matches JSON schema exactly: id, severity, confidence,
                      category, target, covers, proposal, evidence,
                      existingTests)
```

4.2 Output Types

```
Report              — meta Meta, summary Summary,
                      requirements []Requirement,
                      recommendations []Recommendation,
                      scaffolds []Scaffold
Meta                — tool, version, repoRoot, timestamp string,
                      seed *int, mode string,
                      llmFallbacks []string, promptHashes []PromptHash,
                      truncatedPackages []string,
                      inputs InputFiles
Summary             — riskScore int, totalFindings int, truncated bool,
                      missingRecommendations int,
                      unverifiableRequirements int,
                      coverageSignal CoverageSignal
```

4.3 Pipeline Artifacts Container

Passed between stages. Zero values indicate the stage was skipped.

```
Artifacts           — repoGraph *RepoGraph,
                      boundaryMap *BoundaryMap,
                      testInventory *TestInventory,
                      requirementSet *RequirementSet,
                      planIntentSet *PlanIntentSet,
                      symbolIndex *SymbolIndex,
                      riskSignals *RiskSignals,
                      coverageMap *CoverageMap,
                      unmappedRequirements *UnmappedRequirements,
                      untestedIntents *UntestedIntents,
                      recommendations []Recommendation
```

4.4 Config Type

```
Config              — mode string, format string,
                      root string,
                      specPaths []string, planPaths []string,
                      exclude []string, include []string,
                      failOn string, seed *int,
                      maxFindings int,
                      timeout time.Duration,
                      configPath string,
                      llm LLMConfig
LLMConfig           — provider, model string, temperature float64,
                      maxTokens int
```

---

5. Phase 1 — MVP Offline

Goal: verifier analyze produces a useful test gap report using only static heuristics and Go AST analysis. No LLM. JSON + markdown + text output. CI gating via --fail-on.

### Phase 1a — Foundation

Deliverables:
- Go module initialized, Makefile with build/test/lint targets, .golangci.yml (exclude testdata/ from linting, set Go version to 1.22)
- Logging strategy: use stdlib log/slog for structured logging to stderr. Warning format: "WARNING: {message}". Error format: "ERROR: {message}". Notice format: "NOTICE: {message}". No log output to stdout.
- internal/domain/ with all shared types from Section 4
- internal/config/ with full config loading:
  - Parse .verifier.yaml via yaml.v3
  - Parse CLI flags via stdlib flag package
  - Merge logic: CLI flags replace config values; list-type flags replace (not merge)
  - Validation: temperature range, seed range, timeout > 0, glob syntax, path resolution (directory check, readability), duplicate path dedup, format/mode enum checks
  - Error behavior per SPEC: malformed YAML → exit 1, missing default config → continue, missing explicit --config → exit 1
- cmd/verifier/main.go with command dispatch:
  - analyze (stub), init, explain (stub), scaffold (stub)
  - verifier init writes .verifier.yaml from defaults; exits 1 if file exists unless --force
  - version flag prints build-injected version
- internal/pipeline/ with stage orchestrator:
  - Runs stages A→B→C→D→E→F→G in sequence
  - Degraded mode paths (in priority order):
    1. no go.mod AND no spec AND no plan AND no .go files → exit 0 with warning "no analyzable inputs"
    2. no spec AND no go.mod AND .go files present → run A→F→G; Stage F produces only zero-test-package findings; emit warning "code-semantic signals unavailable"
    3. no spec AND has go.mod → run A→C→F→G (degraded offline)
    4. has spec AND no go.mod → run A→B→D→E→F→G (skip C; mapping/strategy operate without code signals)
    5. full mode → run A→B→C→D→E→F→G
  - Each stage receives prior artifacts and returns its own
  - Stage interface: Execute(ctx, *Artifacts) → error

Testing:
- Config parsing: table-driven tests covering YAML + CLI merge, validation errors, missing files, directory-as-path detection, seed range (0 valid, 2^31 invalid), timeout <= 0, invalid duration format, temperature out of range
- Pipeline orchestrator: test all 5 degraded mode paths with mock stages
- Init command: test file creation, exists-error, --force overwrite, atomic creation (os.O_EXCL)
- Explain command: test --input file, stdin piped, both (precedence), unknown TESTREC-ID, re-run fallback
- Stub commands: verify scaffold stub prints to stderr and exits 0

### Phase 1b — Stage A: Repository Inventory

Deliverables:
- internal/repo/ package
- go.mod detection: look for go.mod at --root directory only (do not walk above --root); parse module path
- Package enumeration: filesystem walk respecting --include/--exclude globs
  - --exclude takes precedence when both match
  - Skip files > 1MB with warning
  - Classify: cmd packages (under cmd/), internal, pkg, root
- Test file identification: *_test.go files, extract test function names via simple regex (func Test\w+). This produces the initial TestInventory. Stage C enriches it with subtest detection and line numbers via AST; Stage C output replaces Stage A test entries for files it successfully parses.
- Entrypoint detection: main packages, main() functions
- Boundary detection heuristics:
  - HTTP: imports of net/http, gorilla/mux, gin-gonic/gin, go-chi/chi; function signatures matching http.Handler/http.HandlerFunc patterns
  - DB: imports of database/sql; calls to QueryContext, ExecContext, Query, Exec
  - FS: calls to os.Open, os.Create, os.ReadFile, os.WriteFile, io/ioutil usage
  - External: imports of known HTTP client packages: net/http (Client usage), github.com/go-resty/resty
  - Message queues: imports of github.com/streadway/amqp, github.com/segmentio/kafka-go, github.com/nats-io/nats.go (initial set; extensible in later phases)
- Build and return RepoGraph, BoundaryMap, TestInventory

Testing:
- Filesystem walk: use testdata/ directories with known structure
- Boundary detection: Go files with known imports, assert correct classification
- Glob filtering: test include/exclude precedence, invalid patterns

### Phase 1c — Stage B: Spec & Plan Extraction

Deliverables:
- internal/parse/ package
- Markdown heading parser: split document into heading-tree structure
  - Track heading level, text, line numbers
  - Support ATX headings (#, ##, etc.) and setext (underline) headings
- Requirement extraction:
  - Scan for headings containing keywords (case-insensitive): "requirement", "must", "shall", "feature"
  - Extract numbered/bulleted list items under matching headings
  - Detect explicit IDs via regex: /^(REQ|FR|NFR)-\d+/ and similar alphanumeric-dash-number patterns
  - Synthetic ID generation:
    - Prefix: "SYN-"
    - Heading normalization: strip non-alphanumeric, uppercase, truncate to 10 chars
    - Format: SYN-{heading}-{zero-padded 3-digit index}
    - Collision detection: global across all SYN- IDs; first keeps base, second gets -2, etc.
  - Acceptance criteria detection: sub-items or items with keywords "accept", "criteria", "measurable", "verify", "assert" (case-insensitive)
  - Verifiability assignment: "low" if no acceptance criteria found
- Requirement ID uniqueness enforcement: across all spec files, both explicit and synthetic; exit 1 on conflict
- Plan intent extraction:
  - Scan for headings containing: "component", "module", "architecture", "design", "plan", "integration"
  - Extract component names, responsibilities, integration points from list items
  - Assign plan intent IDs using same normalization as SYN- IDs: "PLAN-" + heading (strip non-alphanumeric, uppercase, truncate to 10 chars) + "-" + zero-padded 3-digit index. Collision handling: same rules as SYN- IDs (first keeps base, second gets -2, etc.)
- Build and return RequirementSet, PlanIntentSet

Testing:
- Golden file tests: sample SPEC.md files → expected RequirementSet JSON
- Synthetic ID generation: collision scenarios, truncation edge cases
- Multi-file extraction: duplicate ID detection
- Edge cases: empty spec, no headings, no requirements, mixed explicit/synthetic IDs

### Phase 1d — Stage C: Go AST Analysis

Deliverables:
- internal/golang/ package
- AST walker using go/parser and go/ast:
  - Parse each .go file (skip > 1MB, skip _test.go for symbol index, include _test.go for test inventory enrichment)
  - Extract exported functions and methods: name, receiver type, file, line range, signature
  - Extract exported types and interfaces
- Error return detection:
  - Functions returning error as last return value
  - Count of error-returning paths (basic: count return statements with non-nil error)
- Concurrency detection:
  - go statements (goroutine launches)
  - Channel operations (make(chan ...), <-, range over channel)
  - sync.Mutex, sync.RWMutex, sync.WaitGroup usage
- HTTP handler detection:
  - Functions with signature (http.ResponseWriter, *http.Request)
  - Functions registered via Handle/HandleFunc patterns
- DB query detection:
  - Calls to methods named Query, QueryContext, QueryRow, Exec, ExecContext on sql.DB, sql.Tx types
- Input validation patterns:
  - Functions with string comparisons, regex matches, or length checks within the first 20 AST statements of the function body, or within any function called from the body whose name matches /[Vv]alidat|[Cc]heck|[Pp]ars|[Ss]anitiz/
- Risk signal computation per symbol:
  - error_path: function returns error
  - concurrency: contains go/channel/mutex usage
  - boundary: is an HTTP handler or DB call site
  - complexity: high branching (> 5 if/switch branches)
  - http_handler: matches handler signature
  - db_query: contains DB calls
  - input_validation: contains validation patterns
- Build and return SymbolIndex, RiskSignals

Testing:
- Parse testdata/ Go files with known structures
- Assert correct symbol extraction, risk signal assignment
- Test file size skip behavior
- Test error return detection accuracy

### Phase 1e — Stage D+E: Mapping and Strategy

Deliverables:
- internal/mapping/ package (Stage D)
- internal/strategy/ package (Stage E)

Stage D — Mapping:
- Name-based matching: tokenize requirement text and symbol names, compute overlap
  - Normalize: lowercase, split on word boundaries (camelCase, snake_case, kebab-case), remove stop words
  - Stop words: "the", "a", "an", "is", "are", "for", "to", "of", "in", "with", "and", "or", "be", "it", "that", "this", "should", "must", "shall", "will", "can"
  - Match score: Jaccard similarity (|intersection| / |union|) between requirement tokens and symbol name tokens
- Package proximity: if requirement mentions a package name, boost symbols in that package
- Confidence scoring (offline mode):
  - 1.0: exact name match in same package
  - 0.7: partial name match (Jaccard >= 0.3) in same package
  - 0.5: partial name match in different package
  - 0.2: package proximity only (requirement mentions package, no symbol name match)
  - 0.1: heuristic guess with no direct match
- Build CoverageMap: for each requirement, list matched symbols with confidence
- Build UnmappedRequirements: requirements with no symbol match above 0.1 threshold
- Build UntestedIntents: plan intents referencing components with no tests

Stage E — Test Strategy:
- For each CoverageMapping entry, assign test category based on risk signals:
  - If symbol has http_handler risk → contract if an OpenAPI/Swagger file (openapi.yaml, swagger.json, openapi.json) is detected in the repo; otherwise → integration
  - If symbol has db_query risk → integration
  - If symbol has concurrency risk → concurrency
  - If symbol has error_path risk and is a validator (has input_validation risk signal, or name matches /[Vv]alidat|[Pp]ars|[Dd]ecode|[Uu]nmarshal/) → fuzz
  - If requirement mentions "performance", "latency", "throughput" and has acceptance criteria → perf
  - If requirement mentions "auth", "security", "injection", "token" → security
  - If symbol has input_validation risk → property (for invariant testing)
  - Default: unit
- For each recommendation, suggest test structure:
  - Table-driven by default for unit tests
  - Subtests for integration tests
  - Golden files for output-heavy functions

Testing:
- Mapping: fabricated RequirementSet + SymbolIndex → expected CoverageMap
- Strategy: symbols with specific risk signals → expected categories
- Edge cases: no matches, all matches, mixed confidence levels

### Phase 1f — Stage F+G: Gaps, Ranking, IDs

Deliverables:
- internal/gaps/ package (Stage F)
- internal/ranking/ package (Stage G)

Stage F — Gap Detection:
- For each recommendation from Stage E:
  - Check TestInventory for tests in the same package
  - Check for test function names similar to the target symbol (e.g., TestCreateUser for CreateUser)
  - Check for subtest patterns
  - Mark as existing_test if found, with gap annotation:
    - "missing_negative_paths": no test names containing Error, Invalid, Fail, Bad
    - "missing_boundary_tests": boundary symbol with no integration test
    - "missing_race_tests": concurrency symbol with no test in _test.go using t.Parallel or race build tag
    - "happy_path_only": test exists but name suggests only positive case
- In degraded mode (no spec/plan): generate findings directly from code signals:
  - Zero-test packages: any package with .go files but no _test.go
  - Untested exports: exported functions with no corresponding Test* function
  - Untested error paths: functions returning error with no error-asserting test
  - Untested concurrency: goroutine/channel usage with no race test

Stage G — Ranking:
- TESTREC ID generation:
  - Input: UTF-8 "{requirementID}\x00{targetSymbol}\x00{category}"
  - Hash: SHA-256, take first 8 uppercase hex chars
  - Format: "TESTREC-{hash}"
  - Collision handling: within a run, track used IDs; append -2, -3, etc. on collision
- Severity assignment using SPEC ranking rules:
  - critical: security/auth risks, concurrency hazards, data loss risks
  - high: core functional requirement gaps, untested boundary integrations
  - medium: error path gaps, validation gaps
  - low: minor edge cases, refactor-safety
  - Keyword detection in requirement text for security/auth/payment/PHI
- Confidence: carried from Stage D mapping
- risk_score computation: min(100, critical*10 + high*5 + medium*2 + low*1) from full pre-truncation set
- Sorting: severity desc → confidence desc → ID asc (lexicographic byte order)
- Truncation: apply --max-findings using full sort key; set summary.truncated and summary.total_findings
- --fail-on evaluation: check if any recommendation has severity >= threshold; set exit code 2 if so

Testing:
- TESTREC ID: known inputs → expected hash output; collision scenarios
- Severity assignment: requirements with specific keywords → expected severity
- Sorting: verify deterministic output order; same seed + same input → identical output; different seeds may produce different tie-breaking order
- Truncation: --max-findings 200 truncates correctly; --max-findings 0 means unlimited (no truncation); --max-findings -1 means unlimited; --max-findings 1 truncates to 1 finding; verify truncated flag and total_findings accuracy
- risk_score: manual computation verification; risk_score computed from pre-truncation set; risk_score recomputed after Phase 4 severity boosts
- Degraded mode: verify code-only findings without spec/plan

### Phase 1g — Output Formatting and CLI Wiring

Deliverables:
- internal/report/ package
- Full CLI wiring in cmd/verifier/

JSON Formatter:
- Marshal Report struct to JSON matching SPEC schema exactly
- Stable field ordering (use struct tags, not map)
- Indent with 2 spaces for human readability
- All output to stdout; warnings to stderr

Markdown Formatter:
- Template-based using text/template
- Sections: Summary, Critical Gaps, High Gaps, Medium Gaps, Low Gaps, Unverifiable Requirements, Appendix (requirement→symbol mapping table)
- Each recommendation: title, category, target, evidence citations, existing test notes

Text Formatter:
- Plain text, no formatting
- Tabular where appropriate (using fixed-width columns)
- Same section structure as markdown

CLI Wiring:
- Wire analyze command: load config → validate → run pipeline → format → output → exit code
- Wire init command: generate default YAML
- Implement verifier explain command:
  - Read prior analysis JSON from --input <file>, stdin (piped), or re-run analysis
  - --input takes precedence over stdin; log notice if both provided
  - Exact-match TESTREC-ID lookup; exit 1 with helpful message if not found
  - Print detailed recommendation explanation to stdout
- Stub scaffold command (prints "not yet implemented, see Phase 3" to stderr, exit 0)
- stderr/stdout discipline: all report output to stdout, all diagnostics to stderr
- Exit code logic: 0 (success), 1 (error), 2 (--fail-on exceeded)

Testing:
- JSON output: golden file tests comparing output to expected JSON
- Markdown output: golden file tests
- End-to-end: run verifier analyze on testdata/ repos, verify exit codes and output structure
- Exit code: test --fail-on with various severity levels

---

6. Phase 2 — LLM Assist

Goal: Add optional LLM-assisted requirement extraction, mapping, and test strategy. Falls back to offline on failure.

### Phase 2a — LLM Interface and Provider

Deliverables:
- internal/llm/ package with LLM interface, CompletionRequest/Response types
- OpenAI provider implementation (behind build tag `llm_openai`)
- Anthropic provider implementation (behind build tag `llm_anthropic`)
- Provider selection from config: llm.provider field
- Timeout handling: context.WithTimeout from --timeout flag
- Retry logic: one retry on malformed JSON response
- Prompt hash computation: SHA-256 of full prompt text, stored in PromptHash struct

Security:
- API keys are read exclusively from environment variables (OPENAI_API_KEY, ANTHROPIC_API_KEY). They must never appear in .verifier.yaml or any config file.
- LLMConfig struct must not contain an api_key field.

Configuration validation:
- --mode llm requires provider + model to be set; exit 1 if missing
- Temperature validated (0.0–2.0) regardless of mode when present in config

### Phase 2b — LLM-Assisted Stages B, D, E

Deliverables:
- LLM-enhanced Stage B: structured prompt to extract requirements with IDs, text, acceptance criteria from spec markdown; validate model JSON output against RequirementSet schema
- LLM-enhanced Stage D: prompt with spec requirements + code symbol summaries; model proposes mappings with confidence scores; validate confidence is 0.0–1.0, default 0.5 if missing
- LLM-enhanced Stage E: prompt with mapped requirements + risk signals; model proposes test categories and detailed proposals

Prompt construction:
- System prompt defines output JSON schema
- User prompt includes relevant slices of spec/plan/code (not entire files)
- Context window management:
  - Prioritize packages by risk signals (highest risk first)
  - Drop lowest-risk packages when context limit approached
  - Record dropped packages in meta.truncated_packages
  - Recommendations for dropped packages fall back to offline; evidence includes "truncated_context": true

Fallback behavior:
- Per-stage: if LLM fails, log warning, use offline result for that stage
- Track fallbacks in meta.llm_fallbacks array
- meta.mode = "llm" if any stage used LLM successfully

### Phase 2c — Seed and Determinism

Deliverables:
- Pass --seed to provider API if supported (OpenAI supports seed param)
- If provider does not support seed, log warning, use seed only for PRNG ordering
- Log prompt hashes in meta.prompt_hashes

Testing:
- Mock LLM provider: return canned responses, verify pipeline behavior
- Fallback: simulate provider failure, verify offline fallback and meta.llm_fallbacks
- Timeout: simulate slow provider, verify context cancellation and fallback
- Malformed response: simulate bad JSON, verify retry + fallback
- Context truncation: large symbol index, verify truncation and meta.truncated_packages
- Prompt hash: verify SHA-256 computation

---

7. Phase 3 — Scaffold

Goal: verifier scaffold generates skeleton test files with TESTREC-ID TODO markers.

### Phase 3a — Scaffold Engine

Deliverables:
- internal/scaffold/ package
- Input: ranked recommendations from a prior analyze run (loaded from JSON via --input <file>, from stdin if piped, or re-runs analysis if neither is provided — same precedence as verifier explain)
- --limit N: select top N critical-severity recommendations; if zero critical, emit notice, exit 0
- --dry-run (default): print planned changes to stdout without writing
- --write: execute file modifications

File strategy:
- For each recommendation target:
  1. Check if *_test.go exists in the target's package directory
  2. If exists and target function has no test → append to existing file (with .bak backup first)
  3. If exists and target function already has a test → skip, log notice
  4. If no *_test.go exists → create new <package>_test.go
- .bak handling: create .bak before modifying; overwrite existing .bak silently; if .bak write fails → abort with exit 1, no modification

### Phase 3b — Test Template Generation

Deliverables:
- Go test templates using text/template
- --style std (default): stdlib testing package only, no external dependencies
- --style go-testify: testify assertions (dependency noted in scaffold comment)
- Template content per test category:
  - unit: table-driven test with TODO assertions and TESTREC-ID
  - integration: subtest structure with setup/teardown comments
  - concurrency: t.Parallel() with race-condition check TODO
  - fuzz: Fuzz function skeleton
  - property: property-based test skeleton with TODO invariant
  - security: test checking auth/validation with TODO
- Each generated test includes: // TODO(TESTREC-{ID}): {proposal.title}
- --package-layout: flat (all in package dir) vs standard (respect internal/pkg/cmd)

### Phase 3c — Scaffold JSON Output

Deliverables:
- verifier scaffold --format json outputs scaffold plan as JSON with:
  - path, actions (add_test_case, create_file), testrec_id, status (written|skipped|dry_run)
- Wire scaffold command into CLI dispatch

Testing:
- Append to existing file: verify .bak created, content appended, original unchanged
- Create new file: verify file created with correct package declaration
- Skip existing test: verify no modification, notice logged
- .bak failure: simulate write error, verify abort behavior
- --dry-run: verify no files written, output shows planned changes
- --limit: verify only top N critical gaps scaffolded
- Template correctness: parse generated Go code, verify it compiles

---

8. Phase 4 — Ecosystem Integration

Goal: Ingest JSON from SpecCritic, PlanCritic, RealityCheck, and Prism to enhance analysis.

### Phase 4a — External Tool JSON Parsing

Deliverables:
- internal/ecosystem/ package
- Parsers for each tool's root-level JSON structure:
  - SpecCritic: {"issues": [...]} — extract id, severity, title, anchor
  - PlanCritic: {"issues": [...]} — extract id, severity, title, component
  - RealityCheck: {"deltas": [...]} — extract id, kind, description, spec_ref, code_ref
  - Prism: {"findings": [...]} — extract id, severity, file, line_start, line_end, message
- Validation: invalid JSON → warn + skip; missing required fields → warn + skip
- Severity mapping: unknown values → "low" with warning
- RealityCheck kind mapping: unknown values → "changed" with warning

### Phase 4b — Integration into Pipeline

Deliverables:
- SpecCritic integration (between Stage B and D):
  - Match anchors to requirement IDs (REQ-\d+ pattern or heading match)
  - Boost: promote severity of linked recommendations by one level (low→medium, medium→high, high→critical)
- PlanCritic integration (between Stage B and D):
  - Treat flagged components as additional test targets
  - Generate recommendations for plan risks even if no code mapping exists
- RealityCheck integration (after Stage E):
  - Generate regression test recommendations per delta
  - TESTREC ID: requirementID="RC-{delta.id}", targetSymbol=code_ref or "unknown", category="integration"
- Prism integration (after Stage E):
  - Promote flagged code sections to test recommendations
  - TESTREC ID: requirementID="PRISM-{finding.id}", targetSymbol=file:line_start or file, category="unit"
  - Severity mapped directly from Prism finding

Ordering:
- All Phase 4 severity boosts (SpecCritic) and new recommendations (RealityCheck, Prism) are applied before Stage G ranking.
- risk_score is computed as the final step in Stage G, after all Phase 4 modifications, ensuring it reflects boosted severities.

CLI flags:
- --speccritic, --plancritic, --realitycheck, --prism: each takes a file path
- All optional; analysis proceeds without them

Testing:
- Each parser: valid JSON, malformed JSON, missing fields, unknown severity/kind values
- SpecCritic boost: verify severity promotion with anchor matching
- RealityCheck: verify TESTREC ID generation for deltas
- Prism: verify TESTREC ID generation and severity mapping
- End-to-end: analyze with all four external inputs, verify merged output

---

9. Testing Strategy

9.1 Unit Tests
- Every package gets *_test.go files
- Table-driven tests as the default pattern
- testdata/ directories for fixture files (Go source files, SPEC.md samples, JSON inputs)
- Golden file tests for output formatters (JSON, markdown, text)
- All exported functions must have at least one test. All error return paths must have at least one test asserting the error condition.

9.2 Integration Tests
- End-to-end tests in cmd/verifier/:
  - Run binary against testdata/ repositories
  - Verify JSON output structure, exit codes, stderr warnings
  - Test degraded mode scenarios (no spec, no go.mod, no spec + no go.mod)
- Pipeline integration: run full Stage A→G on testdata/ repo, compare output to golden file

9.3 Testdata Repositories
- Create testdata/repos/ with small synthetic Go projects:
  - testdata/repos/basic/: simple Go project with spec, plan, source, and tests
  - testdata/repos/no-tests/: Go project with zero test files
  - testdata/repos/no-spec/: Go project with no SPEC.md
  - testdata/repos/no-gomod/: directory with markdown files but no Go code
  - testdata/repos/concurrency/: project using goroutines, channels, mutexes
  - testdata/repos/http-api/: project with HTTP handlers and DB queries

9.4 Race Detection
- All tests run with -race in CI
- Verifier itself should be race-free (no shared mutable state between stages)

---

10. Implementation Order and Dependencies

```
Phase 1a: Foundation
  ├── domain types (no deps)
  ├── config (depends on: domain)
  ├── pipeline orchestrator (depends on: domain, config)
  └── CLI skeleton + init command (depends on: config)

Phase 1b: Stage A — Repo Inventory
  └── depends on: domain, config (for include/exclude globs)

Phase 1c: Stage B — Spec/Plan Extraction
  └── depends on: domain

Phase 1d: Stage C — Go AST Analysis
  └── depends on: domain

Phase 1e: Stages D+E — Mapping + Strategy
  └── depends on: domain, Phase 1c artifacts, Phase 1d artifacts

Phase 1f: Stages F+G — Gaps + Ranking
  └── depends on: domain, Phase 1b artifacts, Phase 1e artifacts

Phase 1g: Output + CLI Wiring
  └── depends on: all Phase 1 stages, report package, config

Phase 2a: LLM Interface
  └── depends on: domain

Phase 2b: LLM-Assisted Stages
  └── depends on: Phase 2a, Phase 1c (Stage B), Phase 1e (Stages D, E)

Phase 2c: Seed/Determinism
  └── depends on: Phase 2a

Phase 3a: Scaffold Engine
  └── depends on: domain, ranking (for TESTREC IDs)

Phase 3b: Test Templates
  └── depends on: Phase 3a

Phase 3c: Scaffold JSON
  └── depends on: Phase 3a, report

Phase 4a: External JSON Parsing
  └── depends on: domain

Phase 4b: Pipeline Integration
  └── depends on: Phase 4a, Phase 1f (ranking), Phase 1e (strategy)
```

---

11. Risks and Mitigations

11.1 Offline Mapping Quality
Risk: Name-based matching (Jaccard on tokens) may produce poor requirement-to-symbol mappings, generating noisy/useless recommendations.
Mitigation: Set a minimum confidence threshold (0.2) below which mappings are discarded. Report UnmappedRequirements explicitly so users see what couldn't be matched. LLM mode in Phase 2 substantially improves this.

11.2 Markdown Parsing Fragility
Risk: Real-world SPEC.md files use inconsistent formatting (mixed heading styles, non-standard list markers, embedded code blocks that look like requirements).
Mitigation: Be permissive in parsing. Skip content inside fenced code blocks. Use a state machine that tracks heading context. Add golden file tests for a variety of real-world markdown patterns.

11.3 Go AST Analysis Scope
Risk: Complex Go patterns (code generation, reflection, build tags, cgo) may not be analyzable via static AST walking.
Mitigation: Focus on the common patterns listed in SPEC Section 9.3. Skip generated files (*.gen.go via default exclude). Document limitations. Avoid false positives — prefer under-reporting to over-reporting.

11.4 TESTREC ID Stability
Risk: Content-addressed IDs depend on exact string matching of requirementID, targetSymbol, and category. Minor changes to the extraction pipeline could change IDs across versions.
Mitigation: Lock down the ID computation function early with comprehensive test cases. Document that ID stability is guaranteed within a version but not across major versions. Include version in JSON meta.

11.5 Build Tag Complexity for LLM Providers
Risk: Build tags for LLM providers add complexity to the build process and CI matrix.
Mitigation: Keep the LLM interface minimal. Provide a default no-op provider that returns an error. Only gate provider implementations, not the interface definition. Document build tags in README.

---

12. Definition of Done per Phase

Phase 1: verifier analyze runs on a Go repository in offline mode and produces correct JSON, markdown, and text output. Exit codes work correctly. golangci-lint clean. All tests pass with -race.

Phase 2: verifier analyze --mode llm produces enhanced output using a configured LLM provider. Falls back gracefully on failure. meta.llm_fallbacks and meta.prompt_hashes populated correctly.

Phase 3: verifier scaffold --write generates compilable test skeleton files with TESTREC-ID TODO markers. .bak safety mechanism works. --dry-run shows planned changes without writing.

Phase 4: verifier analyze with --speccritic/--plancritic/--realitycheck/--prism flags correctly ingests external tool JSON outputs and merges findings into the report with correct severity boosting and TESTREC ID generation.

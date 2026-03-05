# Verifier

Spec-driven test strategy analysis for real codebases.

Verifier is a Go CLI that reads your specs, implementation plans, and codebase to determine what tests should exist — and which ones are missing. It recommends unit, integration, contract, property, fuzz, concurrency, performance, and security tests with evidence, designed for agentic workflows.

## Features

- **7-stage analysis pipeline** — Repo inventory, spec/plan extraction, Go AST analysis, requirement-to-code mapping, test strategy synthesis, gap detection, and severity ranking
- **Two modes** — `offline` (static heuristics + Go AST, fully deterministic) and `llm` (model-assisted extraction, mapping, and strategy)
- **Structured output** — JSON (stable schema for CI/agents), Markdown, and plain text
- **CI gating** — `--fail-on` threshold with exit code 2 when exceeded
- **Test scaffolding** — Generate skeleton test files with TODO markers and TESTREC IDs
- **Ecosystem integration** — Optional ingestion of SpecCritic, PlanCritic, RealityCheck, and Prism JSON outputs
- **Zero dependencies** — stdlib only, single binary

## Installation

```bash
go install github.com/dshills/verifier/cmd/verifier@latest
```

Or build from source:

```bash
git clone https://github.com/dshills/verifier.git
cd verifier
make build
# Binary at bin/verifier
```

## Quick Start

```bash
# Analyze a repo with spec and plan files
verifier analyze --root . --spec SPEC.md --plan PLAN.md

# JSON output for CI pipelines
verifier analyze --format json --fail-on high

# LLM-assisted analysis
verifier analyze --mode llm --llm-provider openai --llm-model gpt-4o

# Generate test scaffolds
verifier analyze --format json > report.json
verifier scaffold --input report.json --write

# Explain a specific recommendation
verifier explain --input report.json TESTREC-A1B2C3D4
```

## Commands

### `verifier analyze`

Primary command. Analyzes a repository against its spec and plan to produce a test gap report.

```
Flags:
  --root <dir>           Repository root directory (default: .)
  --spec <paths>         Comma-separated spec file paths
  --plan <paths>         Comma-separated plan file paths
  --format <fmt>         Output format: json, md, text (default: md)
  --mode <mode>          Analysis mode: offline, llm (default: offline)
  --config <file>        Path to config file (default: .verifier.yaml)
  --fail-on <severity>   Fail if severity >= threshold: none, low, medium, high, critical (default: none)
  --seed <int>           Deterministic seed for reproducible output
  --max-findings <int>   Maximum findings to report, 0 = unlimited (default: 0)
  --include <glob>       Comma-separated include globs
  --exclude <glob>       Comma-separated exclude globs
  --timeout <duration>   Analysis timeout, e.g. 2m, 30s (default: 2m)

LLM flags:
  --llm-provider <name>  LLM provider: openai, anthropic
  --llm-model <name>     Model identifier

Ecosystem flags:
  --spec-critic <file>   Path to SpecCritic JSON output
  --plan-critic <file>   Path to PlanCritic JSON output
  --reality-check <file> Path to RealityCheck JSON output
  --prism <file>         Path to Prism JSON output
```

### `verifier scaffold`

Generates skeleton test files with TESTREC-ID TODO markers for a coding agent to fill in.

```
Flags:
  --input <file>    Path to prior analysis JSON (or pipe via stdin)
  --limit <n>       Only scaffold top N critical recommendations
  --dry-run         Print planned changes without writing (default: true)
  --write           Execute file modifications
  --style <style>   Test style: std, go-testify (default: std)
```

Scaffold behavior:
- Never deletes files
- Appends new tests to existing `_test.go` files with `.bak` backup
- Creates new `<package>_test.go` when no test file exists
- Skips targets that already have tests

### `verifier init`

Creates a default `.verifier.yaml` configuration file.

```
Flags:
  --force    Overwrite existing config file
```

### `verifier explain <TESTREC-ID>`

Prints a detailed explanation for a specific test recommendation.

```
Flags:
  --input <file>    Path to prior analysis JSON (or pipe via stdin)
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0    | Success, no gating triggered |
| 1    | Runtime error (bad config, parse error, IO) |
| 2    | Analysis success, but `--fail-on` threshold exceeded |

## Configuration

Create `.verifier.yaml` in your project root (or run `verifier init`):

```yaml
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
  model: gpt-4o
  temperature: 0.2
  max_tokens: 8000
```

CLI flags override config file values. For list-type values (`spec`, `plan`, `exclude`), CLI flags replace (not merge with) config file values.

LLM API keys are read from environment variables:
- `OPENAI_API_KEY` for OpenAI
- `ANTHROPIC_API_KEY` for Anthropic

## Analysis Pipeline

Verifier runs a 7-stage sequential pipeline:

```
Stage A: Repo Inventory    -> RepoGraph, BoundaryMap, TestInventory
Stage B: Spec/Plan Extract -> RequirementSet, PlanIntentSet
Stage C: Go AST Analysis   -> SymbolIndex, RiskSignals
Stage D: Mapping           -> CoverageMap, UnmappedRequirements, UntestedIntents
Stage E: Test Strategy     -> Recommendations with categories and proposals
Stage F: Gap Detection     -> Annotated recommendations with existing test info
Stage G: Ranking           -> Sorted, scored recommendations with TESTREC IDs
```

### Degraded Modes

The pipeline adapts when inputs are missing:

| Condition | Stages Run | Notes |
|-----------|-----------|-------|
| Full (spec + plan + go.mod) | A -> B -> C -> D -> E -> F -> G | Complete analysis |
| No spec, has go.mod | A -> C -> F -> G | Code-only heuristic analysis |
| Has spec, no go.mod | A -> B -> D -> E -> F -> G | Spec analysis without code semantics |
| No spec, no go.mod, has .go | A -> F -> G | Zero-test package detection only |
| No analyzable inputs | Exit 0 with warning | Nothing to analyze |

### Test Categories

Verifier recommends tests across 8 categories based on code signals and spec content:

- **unit** — Pure logic, validation, formatting (table-driven by default)
- **integration** — Boundary crossings: HTTP handlers, DB queries, external services
- **contract** — API behavior conformance (when OpenAPI/Swagger detected)
- **property** — Invariants, idempotency, round-trip serialization
- **fuzz** — Parsers, decoders, validators with error paths
- **concurrency** — Goroutines, channels, mutexes, shared state
- **perf** — Only when spec defines measurable performance criteria
- **security** — Auth, authorization, injection, token handling

### TESTREC IDs

Every recommendation gets a content-addressed ID:
- Computed from SHA-256 of `{requirementID}\x00{targetSymbol}\x00{category}`
- Format: `TESTREC-` + first 8 uppercase hex chars (e.g., `TESTREC-A1B2C3D4`)
- Stable across runs for the same inputs
- Used in scaffold TODO markers and `verifier explain` lookups

### Severity and Risk

Recommendations are assigned severity based on the nature of the gap:

| Severity | Criteria |
|----------|----------|
| critical | Security/auth gaps, concurrency hazards, data loss risks |
| high | Core functional requirements untested, boundary integrations |
| medium | Error path gaps, validation gaps |
| low | Minor edge cases, refactor safety |

Risk score: `min(100, critical*10 + high*5 + medium*2 + low*1)`

## Ecosystem Integration

Verifier composes with other tools in an agentic workflow:

```bash
# Run ecosystem tools
speccritic check SPEC.md --format json > speccritic.json
plancritic check PLAN.md --format json > plancritic.json
realitycheck check --format json > reality.json
prism review --format json > prism.json

# Analyze with all inputs
verifier analyze \
  --spec SPEC.md --plan PLAN.md \
  --spec-critic speccritic.json \
  --plan-critic plancritic.json \
  --reality-check reality.json \
  --prism prism.json \
  --format json \
  --fail-on high
```

| Tool | Input Flag | Effect |
|------|-----------|--------|
| [SpecCritic](https://github.com/dshills/speccritic) | `--spec-critic` | Boosts severity of recommendations linked to flagged requirements |
| [PlanCritic](https://github.com/dshills/plancritic) | `--plan-critic` | Generates recommendations for plan-identified risks |
| [RealityCheck](https://github.com/dshills/realitycheck) | `--reality-check` | Generates regression test recommendations for spec-code deltas |
| [Prism](https://github.com/dshills/prism) | `--prism` | Promotes code review findings to test recommendations |

All ecosystem inputs are optional. Verifier produces useful results with just a spec and codebase.

## JSON Output Schema

The `--format json` output follows a stable schema:

```json
{
  "meta": {
    "tool": "verifier",
    "version": "0.1.0",
    "repo_root": ".",
    "timestamp": "2024-01-01T00:00:00Z",
    "seed": null,
    "mode": "offline",
    "inputs": {
      "spec_files": ["SPEC.md"],
      "plan_files": ["PLAN.md"]
    }
  },
  "summary": {
    "risk_score": 42,
    "total_findings": 15,
    "truncated": false,
    "missing_recommendations": 3,
    "unverifiable_requirements": 2
  },
  "requirements": [
    {
      "id": "REQ-001",
      "text": "Users must authenticate before accessing resources",
      "verifiability": "high",
      "issues": [],
      "evidence": [{"kind": "spec", "file": "SPEC.md", "anchor": "REQ-001"}]
    }
  ],
  "recommendations": [
    {
      "id": "TESTREC-A1B2C3D4",
      "severity": "critical",
      "confidence": 0.85,
      "category": "security",
      "target": {
        "kind": "function",
        "name": "AuthMiddleware",
        "file": "internal/auth/middleware.go",
        "line_start": 15,
        "line_end": 45
      },
      "covers": {
        "requirements": ["REQ-001"],
        "plan_items": ["PLAN-AUTH-001"],
        "risks": ["boundary", "http_handler"]
      },
      "proposal": {
        "title": "Test authentication bypass scenarios",
        "approach": "table-driven test with invalid/missing/expired tokens",
        "assertions": ["returns 401 for missing token", "returns 403 for expired token"]
      },
      "evidence": [
        {"kind": "spec", "file": "SPEC.md", "anchor": "REQ-001"},
        {"kind": "code", "file": "internal/auth/middleware.go", "symbol": "AuthMiddleware"}
      ],
      "existing_tests": []
    }
  ]
}
```

## CI Usage

Add to your CI pipeline to gate on test coverage gaps:

```yaml
# GitHub Actions example
- name: Analyze test gaps
  run: |
    verifier analyze --format json --fail-on high > report.json
    # Exit code 2 if any high/critical gaps found
```

```yaml
# With ecosystem tools
- name: Full analysis
  run: |
    speccritic check SPEC.md --format json > speccritic.json
    verifier analyze \
      --spec-critic speccritic.json \
      --format json \
      --fail-on high
```

## Development

```bash
# Build
make build

# Run all tests with race detection
make test

# Lint
make lint

# Run a single package's tests
go test ./internal/parse/... -race -count=1

# Run a specific test
go test -run TestExtractRequirements ./internal/parse/...
```

Requires Go 1.22+ and [golangci-lint](https://golangci-lint.run/).

## Architecture

```
cmd/verifier/           CLI entry point and command dispatch
internal/
  config/               Config loading, validation, CLI-flag merging
  pipeline/             Stage orchestrator, degraded mode detection
  domain/               Shared types (Requirement, Symbol, Recommendation, etc.)
  repo/                 Stage A: repo scanning, boundary detection
  parse/                Stage B: markdown parsing, requirement extraction
  golang/               Stage C: Go AST analysis, risk signal detection
  mapping/              Stage D: requirement-to-symbol mapping
  strategy/             Stage E: test category assignment
  gaps/                 Stage F: gap detection, existing test comparison
  ranking/              Stage G: severity, TESTREC IDs, sorting, risk score
  report/               Output formatting (JSON, markdown, text)
  llm/                  LLM provider interface (OpenAI, Anthropic)
  scaffold/             Test skeleton generation
  ecosystem/            External tool JSON ingestion
```

## License

MIT

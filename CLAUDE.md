# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Verifier is a Go CLI that analyzes a repository against its SPEC.md and PLAN.md to produce a structured test gap report. It recommends what tests should exist (unit, integration, contract, property, fuzz, concurrency, perf, security) and identifies which are missing or weak. Designed for agentic workflows alongside SpecCritic, PlanCritic, RealityCheck, and Prism.

## Build & Test Commands

```bash
go build ./...                        # build all packages
go build -o verifier ./cmd/verifier   # build the CLI binary
go test ./...                         # run all tests
go test ./internal/parse/...          # run tests for a single package
go test -run TestFunctionName ./...   # run a single test by name
go test -race ./...                   # run tests with race detector
golangci-lint run ./...               # lint (run after any Go changes)
```

## Architecture

The spec defines a multi-stage analysis pipeline (stages A-G):

- **Stage A — Repo Inventory**: Scan Go modules, packages, test files, entrypoints, and external boundaries (HTTP/DB/FS)
- **Stage B — Spec & Plan Extraction**: Parse SPEC.md/PLAN.md into structured requirements and plan intents
- **Stage C — Code Semantics**: Go AST analysis for exported APIs, error returns, concurrency primitives, boundary calls
- **Stage D — Mapping**: Map requirements/plan items to code symbols
- **Stage E — Test Strategy**: Determine appropriate test category per requirement/risk
- **Stage F — Gap Detection**: Compare recommended tests against existing tests
- **Stage G — Ranking & Reporting**: Score by severity/confidence, produce final output

### CLI Commands

- `verifier analyze` — primary command, produces test gap report
- `verifier scaffold` — writes skeleton test files (append-only, never deletes)
- `verifier init` — generates `.verifier.yaml` defaults
- `verifier explain <TESTREC-ID>` — deep explanation for a recommendation

### Exit Codes

- `0` — success, no gating triggered
- `1` — runtime error
- `2` — analysis success but `--fail-on` threshold exceeded

## Key Design Constraints

- **Minimal dependencies**: stdlib only preferred; tiny CLI parser at most
- **Go 1.22+**
- **Two modes**: `offline` (static heuristics + Go AST) and `llm` (model-assisted)
- **Deterministic output**: stable ordering by severity desc → confidence desc → ID asc; `--seed` flag for reproducibility
- **Output formats**: `text`, `md`, `json` — JSON schema is defined in `specs/SPEC.md` section 6.2
- **LLM interface**: minimal `Complete(ctx, req) (resp, error)` pattern; supports OpenAI and Anthropic providers
- **Ecosystem integration**: optional ingestion of SpecCritic/PlanCritic/RealityCheck/Prism JSON outputs via CLI flags

### Package Layout

```
/cmd/verifier          # main + CLI wiring
/internal/config       # config parsing + defaults
/internal/pipeline     # stage orchestrator, degraded mode logic
/internal/domain       # shared types (Requirement, Symbol, Recommendation, etc.)
/internal/repo         # Stage A: repo scanning, go.mod detection, boundary detection
/internal/parse        # Stage B: markdown parsing, requirement/plan extraction
/internal/golang       # Stage C: Go AST analysis, symbol indexing, risk signals
/internal/mapping      # Stage D: requirement-to-symbol mapping, confidence scoring
/internal/strategy     # Stage E: test category assignment
/internal/gaps         # Stage F: gap detection, existing test comparison
/internal/ranking      # Stage G: severity, TESTREC IDs, sorting, risk_score
/internal/report       # output formatting (md/json/text)
/internal/llm          # Phase 2: LLM interface + providers (OpenAI, Anthropic)
/internal/scaffold     # Phase 3: test file generation
/internal/ecosystem    # Phase 4: external tool JSON ingestion
```

## Dependencies

- **stdlib only** — zero external dependencies, including a hand-rolled YAML parser for config.

## Specification & Plan

- `specs/SPEC.md` — full specification (JSON schema, CLI flags, pipeline behavior, exit codes)
- `specs/PLAN.md` — phased implementation plan (Phase 1: MVP Offline, Phase 2: LLM, Phase 3: Scaffold, Phase 4: Ecosystem)

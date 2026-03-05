package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/dshills/verifier/internal/domain"
)

// Stage is the interface implemented by each pipeline stage.
type Stage interface {
	Name() string
	Execute(ctx context.Context, cfg *domain.Config, arts *domain.Artifacts) error
}

// Pipeline runs analysis stages in sequence.
type Pipeline struct {
	stageA Stage // repo inventory
	stageB Stage // spec/plan extraction
	stageC Stage // Go AST analysis
	stageD Stage // mapping
	stageE Stage // test strategy
	stageF Stage // gap detection
	stageG Stage // ranking and reporting
}

// New creates a pipeline with the given stages.
func New(a, b, c, d, e, f, g Stage) *Pipeline {
	return &Pipeline{
		stageA: a, stageB: b, stageC: c,
		stageD: d, stageE: e, stageF: f, stageG: g,
	}
}

// DegradedMode determines which pipeline path to run.
type DegradedMode int

const (
	ModeFull       DegradedMode = iota // A→B→C→D→E→F→G
	ModeNoGoMod                        // A→B→D→E→F→G (skip C)
	ModeNoSpec                         // A→C→F→G
	ModeNoSpecNoGo                     // A→F→G
	ModeNoInputs                       // exit 0
)

// DetectMode determines the pipeline mode based on available inputs.
func DetectMode(cfg *domain.Config) DegradedMode {
	hasSpec := hasAnyFile(cfg.SpecPaths)
	hasPlan := hasAnyFile(cfg.PlanPaths)
	hasGoMod := hasFile(filepath.Join(cfg.Root, "go.mod"))
	hasGoFiles := hasGoFilesInTree(cfg.Root)

	hasSpecOrPlan := hasSpec || hasPlan

	if !hasGoMod && !hasSpecOrPlan && !hasGoFiles {
		return ModeNoInputs
	}
	if !hasSpecOrPlan && !hasGoMod {
		if hasGoFiles {
			return ModeNoSpecNoGo
		}
		return ModeNoInputs
	}
	if !hasSpecOrPlan && hasGoMod {
		return ModeNoSpec
	}
	if hasSpecOrPlan && !hasGoMod {
		return ModeNoGoMod
	}
	return ModeFull
}

// Run executes the pipeline stages appropriate for the detected mode.
func (p *Pipeline) Run(ctx context.Context, cfg *domain.Config) (*domain.Artifacts, error) {
	mode := DetectMode(cfg)
	arts := &domain.Artifacts{}

	switch mode {
	case ModeNoInputs:
		slog.Warn("no analyzable inputs found (no go.mod, no spec, no plan, no Go files)")
		return arts, nil

	case ModeNoSpecNoGo:
		slog.Warn("no spec/plan files and no go.mod found; running limited analysis")
		slog.Warn("code-semantic signals unavailable")
		return p.runStages(ctx, cfg, arts, p.stageA, p.stageF, p.stageG)

	case ModeNoSpec:
		slog.Warn("no spec/plan files found; running degraded offline mode (code-only)")
		return p.runStages(ctx, cfg, arts, p.stageA, p.stageC, p.stageF, p.stageG)

	case ModeNoGoMod:
		slog.Warn("no go.mod found; skipping Go AST analysis")
		return p.runStages(ctx, cfg, arts, p.stageA, p.stageB, p.stageD, p.stageE, p.stageF, p.stageG)

	default: // ModeFull
		return p.runStages(ctx, cfg, arts, p.stageA, p.stageB, p.stageC, p.stageD, p.stageE, p.stageF, p.stageG)
	}
}

func (p *Pipeline) runStages(ctx context.Context, cfg *domain.Config, arts *domain.Artifacts, stages ...Stage) (*domain.Artifacts, error) {
	for _, s := range stages {
		if s == nil {
			continue
		}
		if err := s.Execute(ctx, cfg, arts); err != nil {
			return arts, fmt.Errorf("stage %s: %w", s.Name(), err)
		}
	}
	return arts, nil
}

func hasAnyFile(paths []string) bool {
	for _, p := range paths {
		if hasFile(p) {
			return true
		}
	}
	return false
}

func hasFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func hasGoFilesInTree(root string) bool {
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && filepath.Ext(path) == ".go" {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

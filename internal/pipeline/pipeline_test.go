package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dshills/verifier/internal/domain"
)

type mockStage struct {
	name     string
	executed bool
}

func (m *mockStage) Name() string { return m.name }
func (m *mockStage) Execute(_ context.Context, _ *domain.Config, _ *domain.Artifacts) error {
	m.executed = true
	return nil
}

func newMocks() (a, b, c, d, e, f, g *mockStage) {
	return &mockStage{name: "A"}, &mockStage{name: "B"}, &mockStage{name: "C"},
		&mockStage{name: "D"}, &mockStage{name: "E"}, &mockStage{name: "F"}, &mockStage{name: "G"}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestDetectModeNoInputs(t *testing.T) {
	dir := t.TempDir()
	cfg := &domain.Config{Root: dir}
	mode := DetectMode(cfg)
	if mode != ModeNoInputs {
		t.Errorf("got mode %d, want ModeNoInputs (%d)", mode, ModeNoInputs)
	}
}

func TestDetectModeNoSpecNoGo(t *testing.T) {
	dir := t.TempDir()
	// Create a .go file but no go.mod and no spec
	writeFile(t, filepath.Join(dir, "main.go"), "package main")
	cfg := &domain.Config{Root: dir}
	mode := DetectMode(cfg)
	if mode != ModeNoSpecNoGo {
		t.Errorf("got mode %d, want ModeNoSpecNoGo (%d)", mode, ModeNoSpecNoGo)
	}
}

func TestDetectModeNoSpec(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module test")
	cfg := &domain.Config{Root: dir}
	mode := DetectMode(cfg)
	if mode != ModeNoSpec {
		t.Errorf("got mode %d, want ModeNoSpec (%d)", mode, ModeNoSpec)
	}
}

func TestDetectModeNoGoMod(t *testing.T) {
	dir := t.TempDir()
	specFile := filepath.Join(dir, "SPEC.md")
	writeFile(t, specFile, "# Spec")
	cfg := &domain.Config{
		Root:      dir,
		SpecPaths: []string{specFile},
	}
	mode := DetectMode(cfg)
	if mode != ModeNoGoMod {
		t.Errorf("got mode %d, want ModeNoGoMod (%d)", mode, ModeNoGoMod)
	}
}

func TestDetectModeFull(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module test")
	specFile := filepath.Join(dir, "SPEC.md")
	writeFile(t, specFile, "# Spec")
	cfg := &domain.Config{
		Root:      dir,
		SpecPaths: []string{specFile},
	}
	mode := DetectMode(cfg)
	if mode != ModeFull {
		t.Errorf("got mode %d, want ModeFull (%d)", mode, ModeFull)
	}
}

func TestRunFull(t *testing.T) {
	a, b, c, d, e, f, g := newMocks()
	p := New(a, b, c, d, e, f, g)

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module test")
	specFile := filepath.Join(dir, "SPEC.md")
	writeFile(t, specFile, "# Spec")

	cfg := &domain.Config{Root: dir, SpecPaths: []string{specFile}}
	_, err := p.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	for _, s := range []*mockStage{a, b, c, d, e, f, g} {
		if !s.executed {
			t.Errorf("stage %s not executed in full mode", s.name)
		}
	}
}

func TestRunNoSpec(t *testing.T) {
	a, b, c, d, e, f, g := newMocks()
	p := New(a, b, c, d, e, f, g)

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module test")

	cfg := &domain.Config{Root: dir}
	_, err := p.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if !a.executed || !c.executed || !f.executed || !g.executed {
		t.Error("expected A, C, F, G to run in NoSpec mode")
	}
	if b.executed || d.executed || e.executed {
		t.Error("expected B, D, E to not run in NoSpec mode")
	}
}

func TestRunNoGoMod(t *testing.T) {
	a, b, c, d, e, f, g := newMocks()
	p := New(a, b, c, d, e, f, g)

	dir := t.TempDir()
	specFile := filepath.Join(dir, "SPEC.md")
	writeFile(t, specFile, "# Spec")

	cfg := &domain.Config{Root: dir, SpecPaths: []string{specFile}}
	_, err := p.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if !a.executed || !b.executed || !d.executed || !e.executed || !f.executed || !g.executed {
		t.Error("expected A, B, D, E, F, G to run in NoGoMod mode")
	}
	if c.executed {
		t.Error("expected C to not run in NoGoMod mode")
	}
}

func TestRunNoInputs(t *testing.T) {
	a, b, c, d, e, f, g := newMocks()
	p := New(a, b, c, d, e, f, g)

	dir := t.TempDir()
	cfg := &domain.Config{Root: dir}
	_, err := p.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	for _, s := range []*mockStage{a, b, c, d, e, f, g} {
		if s.executed {
			t.Errorf("stage %s should not execute in NoInputs mode", s.name)
		}
	}
}

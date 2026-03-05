package config

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dshills/verifier/internal/domain"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()
	if cfg.Mode != DefaultMode {
		t.Errorf("mode = %q, want %q", cfg.Mode, DefaultMode)
	}
	if cfg.Format != DefaultFormat {
		t.Errorf("format = %q, want %q", cfg.Format, DefaultFormat)
	}
	if cfg.MaxFindings != DefaultMaxFindings {
		t.Errorf("max_findings = %d, want %d", cfg.MaxFindings, DefaultMaxFindings)
	}
	if cfg.Timeout != DefaultTimeout {
		t.Errorf("timeout = %v, want %v", cfg.Timeout, DefaultTimeout)
	}
}

func TestParseYAML(t *testing.T) {
	yaml := `
mode: llm
format: json
spec:
  - myspec.md
  - other.md
plan:
  - myplan.md
exclude:
  - "**/vendor/**"
ci:
  fail_on: high
llm:
  provider: openai
  model: gpt-4
  temperature: 0.5
  max_tokens: 4000
`
	cfg := &domain.Config{}
	if err := parseYAML([]byte(yaml), cfg); err != nil {
		t.Fatalf("parseYAML: %v", err)
	}
	if cfg.Mode != "llm" {
		t.Errorf("mode = %q, want llm", cfg.Mode)
	}
	if cfg.Format != "json" {
		t.Errorf("format = %q, want json", cfg.Format)
	}
	if len(cfg.SpecPaths) != 2 {
		t.Errorf("spec paths len = %d, want 2", len(cfg.SpecPaths))
	}
	if len(cfg.PlanPaths) != 1 {
		t.Errorf("plan paths len = %d, want 1", len(cfg.PlanPaths))
	}
	if cfg.CI.FailOn != "high" {
		t.Errorf("ci.fail_on = %q, want high", cfg.CI.FailOn)
	}
	if cfg.LLM.Provider != "openai" {
		t.Errorf("llm.provider = %q, want openai", cfg.LLM.Provider)
	}
	if cfg.LLM.Model != "gpt-4" {
		t.Errorf("llm.model = %q, want gpt-4", cfg.LLM.Model)
	}
	if cfg.LLM.Temperature != 0.5 {
		t.Errorf("llm.temperature = %f, want 0.5", cfg.LLM.Temperature)
	}
	if cfg.LLM.MaxTokens != 4000 {
		t.Errorf("llm.max_tokens = %d, want 4000", cfg.LLM.MaxTokens)
	}
}

func TestValidateMode(t *testing.T) {
	tests := []struct {
		mode    string
		wantErr bool
	}{
		{"offline", false},
		{"llm", true}, // requires provider+model
		{"invalid", true},
	}
	for _, tt := range tests {
		cfg := Defaults()
		cfg.Mode = tt.mode
		err := Validate(cfg)
		if (err != nil) != tt.wantErr {
			t.Errorf("mode=%q: err=%v, wantErr=%v", tt.mode, err, tt.wantErr)
		}
	}
}

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		format  string
		wantErr bool
	}{
		{"text", false},
		{"md", false},
		{"json", false},
		{"xml", true},
	}
	for _, tt := range tests {
		cfg := Defaults()
		cfg.Format = tt.format
		err := Validate(cfg)
		if (err != nil) != tt.wantErr {
			t.Errorf("format=%q: err=%v, wantErr=%v", tt.format, err, tt.wantErr)
		}
	}
}

func TestValidateFailOn(t *testing.T) {
	tests := []struct {
		failOn  string
		wantErr bool
	}{
		{"none", false},
		{"low", false},
		{"medium", false},
		{"high", false},
		{"critical", false},
		{"extreme", true},
	}
	for _, tt := range tests {
		cfg := Defaults()
		cfg.FailOn = tt.failOn
		err := Validate(cfg)
		if (err != nil) != tt.wantErr {
			t.Errorf("fail_on=%q: err=%v, wantErr=%v", tt.failOn, err, tt.wantErr)
		}
	}
}

func TestValidateSeed(t *testing.T) {
	tests := []struct {
		seed    int
		wantErr bool
	}{
		{0, false},
		{42, false},
		{math.MaxInt32, false},
		{math.MaxInt32 + 1, true},
		{-1, true},
	}
	for _, tt := range tests {
		cfg := Defaults()
		s := tt.seed
		cfg.Seed = &s
		err := Validate(cfg)
		if (err != nil) != tt.wantErr {
			t.Errorf("seed=%d: err=%v, wantErr=%v", tt.seed, err, tt.wantErr)
		}
	}
}

func TestValidateTemperature(t *testing.T) {
	tests := []struct {
		temp    float64
		wantErr bool
	}{
		{0.0, false},
		{1.0, false},
		{2.0, false},
		{-0.1, true},
		{2.1, true},
	}
	for _, tt := range tests {
		cfg := Defaults()
		cfg.LLM.Temperature = tt.temp
		err := Validate(cfg)
		if (err != nil) != tt.wantErr {
			t.Errorf("temperature=%f: err=%v, wantErr=%v", tt.temp, err, tt.wantErr)
		}
	}
}

func TestValidateLLMRequiresProvider(t *testing.T) {
	cfg := Defaults()
	cfg.Mode = "llm"
	cfg.LLM.Provider = ""
	cfg.LLM.Model = "gpt-4"
	if err := Validate(cfg); err == nil {
		t.Error("expected error for llm mode without provider")
	}
}

func TestValidateLLMRequiresModel(t *testing.T) {
	cfg := Defaults()
	cfg.Mode = "llm"
	cfg.LLM.Provider = "openai"
	cfg.LLM.Model = ""
	if err := Validate(cfg); err == nil {
		t.Error("expected error for llm mode without model")
	}
}

func TestValidateDirectoryAsPath(t *testing.T) {
	dir := t.TempDir()
	cfg := Defaults()
	cfg.SpecPaths = []string{dir}
	err := Validate(cfg)
	if err == nil {
		t.Error("expected error for directory as spec path")
	}
}

func TestMergeCLI(t *testing.T) {
	dst := Defaults()
	cli := &domain.Config{
		Mode:   "llm",
		Format: "json",
		Root:   "/tmp/test",
	}
	mergeCLI(dst, cli)
	if dst.Mode != "llm" {
		t.Errorf("mode = %q, want llm", dst.Mode)
	}
	if dst.Format != "json" {
		t.Errorf("format = %q, want json", dst.Format)
	}
	if dst.Root != "/tmp/test" {
		t.Errorf("root = %q, want /tmp/test", dst.Root)
	}
}

func TestMergeCLIListReplace(t *testing.T) {
	dst := Defaults()
	dst.SpecPaths = []string{"old.md"}
	cli := &domain.Config{
		SpecPaths: []string{"new1.md", "new2.md"},
	}
	mergeCLI(dst, cli)
	if len(dst.SpecPaths) != 2 || dst.SpecPaths[0] != "new1.md" {
		t.Errorf("spec paths not replaced: %v", dst.SpecPaths)
	}
}

func TestLoadMissingDefaultConfig(t *testing.T) {
	// Change to temp dir where no .verifier.yaml exists
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load with missing default config: %v", err)
	}
	if cfg.Mode != DefaultMode {
		t.Errorf("mode = %q, want %q", cfg.Mode, DefaultMode)
	}
}

func TestLoadExplicitMissingConfig(t *testing.T) {
	cli := &domain.Config{ConfigPath: "/nonexistent/config.yaml"}
	_, err := Load(cli)
	if err == nil {
		t.Error("expected error for missing explicit config")
	}
}

func TestLoadWithYAMLFile(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `mode: llm
format: json
llm:
  provider: openai
  model: gpt-4
  temperature: 0.3
`
	path := filepath.Join(dir, "test.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cli := &domain.Config{
		ConfigPath: path,
		Root:       dir,
	}
	cfg, err := Load(cli)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Mode != "llm" {
		t.Errorf("mode = %q, want llm", cfg.Mode)
	}
	if cfg.LLM.Provider != "openai" {
		t.Errorf("provider = %q, want openai", cfg.LLM.Provider)
	}
}

func TestValidateTimeout(t *testing.T) {
	cfg := Defaults()
	cfg.Mode = "llm"
	cfg.LLM.Provider = "openai"
	cfg.LLM.Model = "gpt-4"
	cfg.Timeout = -1 * time.Second
	err := Validate(cfg)
	if err == nil {
		t.Error("expected error for negative timeout in llm mode")
	}
}

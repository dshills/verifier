package config

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dshills/verifier/internal/domain"
)

// clearVerifierEnv unsets all VERIFIER_* environment variables for test isolation.
func clearVerifierEnv(t *testing.T) {
	t.Helper()
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "VERIFIER_") {
			key, _, _ := strings.Cut(kv, "=")
			old := os.Getenv(key)
			_ = os.Unsetenv(key)
			t.Cleanup(func() { _ = os.Setenv(key, old) })
		}
	}
}

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
	clearVerifierEnv(t)
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
	clearVerifierEnv(t)
	cli := &domain.Config{ConfigPath: "/nonexistent/config.yaml"}
	_, err := Load(cli)
	if err == nil {
		t.Error("expected error for missing explicit config")
	}
}

func TestLoadWithYAMLFile(t *testing.T) {
	clearVerifierEnv(t)
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

func TestMergeEnvAllSettings(t *testing.T) {
	clearVerifierEnv(t)
	t.Setenv("VERIFIER_MODE", "llm")
	t.Setenv("VERIFIER_FORMAT", "json")
	t.Setenv("VERIFIER_ROOT", "/tmp/myrepo")
	t.Setenv("VERIFIER_SPEC", "a.md,b.md")
	t.Setenv("VERIFIER_PLAN", "p1.md, p2.md")
	t.Setenv("VERIFIER_EXCLUDE", "**/gen/**")
	t.Setenv("VERIFIER_INCLUDE", "**/*.go")
	t.Setenv("VERIFIER_FAIL_ON", "critical")
	t.Setenv("VERIFIER_SEED", "42")
	t.Setenv("VERIFIER_MAX_FINDINGS", "50")
	t.Setenv("VERIFIER_TIMEOUT", "5m")
	t.Setenv("VERIFIER_LLM_PROVIDER", "gemini")
	t.Setenv("VERIFIER_LLM_MODEL", "gemini-2.0-flash")
	t.Setenv("VERIFIER_LLM_TEMPERATURE", "0.7")
	t.Setenv("VERIFIER_LLM_MAX_TOKENS", "4096")
	t.Setenv("VERIFIER_SPEC_CRITIC", "/tmp/sc.json")
	t.Setenv("VERIFIER_PLAN_CRITIC", "/tmp/pc.json")
	t.Setenv("VERIFIER_REALITY_CHECK", "/tmp/rc.json")
	t.Setenv("VERIFIER_PRISM", "/tmp/pr.json")

	cfg := Defaults()
	mergeEnv(cfg)

	if cfg.Mode != "llm" {
		t.Errorf("mode = %q, want llm", cfg.Mode)
	}
	if cfg.Format != "json" {
		t.Errorf("format = %q, want json", cfg.Format)
	}
	if cfg.Root != "/tmp/myrepo" {
		t.Errorf("root = %q, want /tmp/myrepo", cfg.Root)
	}
	if len(cfg.SpecPaths) != 2 || cfg.SpecPaths[0] != "a.md" || cfg.SpecPaths[1] != "b.md" {
		t.Errorf("spec = %v, want [a.md b.md]", cfg.SpecPaths)
	}
	if len(cfg.PlanPaths) != 2 || cfg.PlanPaths[0] != "p1.md" || cfg.PlanPaths[1] != "p2.md" {
		t.Errorf("plan = %v, want [p1.md p2.md]", cfg.PlanPaths)
	}
	if len(cfg.Exclude) != 1 || cfg.Exclude[0] != "**/gen/**" {
		t.Errorf("exclude = %v, want [**/gen/**]", cfg.Exclude)
	}
	if len(cfg.Include) != 1 || cfg.Include[0] != "**/*.go" {
		t.Errorf("include = %v, want [**/*.go]", cfg.Include)
	}
	if cfg.FailOn != "critical" {
		t.Errorf("fail_on = %q, want critical", cfg.FailOn)
	}
	if cfg.Seed == nil || *cfg.Seed != 42 {
		t.Errorf("seed = %v, want 42", cfg.Seed)
	}
	if cfg.MaxFindings != 50 {
		t.Errorf("max_findings = %d, want 50", cfg.MaxFindings)
	}
	if cfg.Timeout != 5*time.Minute {
		t.Errorf("timeout = %v, want 5m", cfg.Timeout)
	}
	if cfg.LLM.Provider != "gemini" {
		t.Errorf("provider = %q, want gemini", cfg.LLM.Provider)
	}
	if cfg.LLM.Model != "gemini-2.0-flash" {
		t.Errorf("model = %q, want gemini-2.0-flash", cfg.LLM.Model)
	}
	if cfg.LLM.Temperature != 0.7 {
		t.Errorf("temperature = %f, want 0.7", cfg.LLM.Temperature)
	}
	if cfg.LLM.MaxTokens != 4096 {
		t.Errorf("max_tokens = %d, want 4096", cfg.LLM.MaxTokens)
	}
	if cfg.SpecCriticPath != "/tmp/sc.json" {
		t.Errorf("spec_critic = %q, want /tmp/sc.json", cfg.SpecCriticPath)
	}
	if cfg.PlanCriticPath != "/tmp/pc.json" {
		t.Errorf("plan_critic = %q, want /tmp/pc.json", cfg.PlanCriticPath)
	}
	if cfg.RealityCheckPath != "/tmp/rc.json" {
		t.Errorf("reality_check = %q, want /tmp/rc.json", cfg.RealityCheckPath)
	}
	if cfg.PrismPath != "/tmp/pr.json" {
		t.Errorf("prism = %q, want /tmp/pr.json", cfg.PrismPath)
	}
}

func TestMergeEnvOverriddenByCLI(t *testing.T) {
	clearVerifierEnv(t)
	t.Setenv("VERIFIER_LLM_PROVIDER", "gemini")
	t.Setenv("VERIFIER_LLM_MODEL", "gemini-2.0-flash")
	t.Setenv("VERIFIER_FORMAT", "json")

	cfg := Defaults()
	mergeEnv(cfg)
	mergeCLI(cfg, &domain.Config{
		Format: "text",
		LLM: domain.LLMConfig{
			Provider: "anthropic",
			Model:    "claude-haiku-4-5-20251001",
		},
	})

	if cfg.LLM.Provider != "anthropic" {
		t.Errorf("provider = %q, want anthropic (CLI override)", cfg.LLM.Provider)
	}
	if cfg.LLM.Model != "claude-haiku-4-5-20251001" {
		t.Errorf("model = %q, want claude-haiku-4-5-20251001 (CLI override)", cfg.LLM.Model)
	}
	if cfg.Format != "text" {
		t.Errorf("format = %q, want text (CLI override)", cfg.Format)
	}
}

func TestMergeEnvEmptyNoOverride(t *testing.T) {
	clearVerifierEnv(t)

	cfg := Defaults()
	cfg.LLM.Provider = "openai"
	cfg.LLM.Model = "gpt-4o"
	mergeEnv(cfg)

	if cfg.Mode != DefaultMode {
		t.Errorf("mode = %q, want %q (unchanged)", cfg.Mode, DefaultMode)
	}
	if cfg.LLM.Provider != "openai" {
		t.Errorf("provider = %q, want openai (unchanged)", cfg.LLM.Provider)
	}
	if cfg.LLM.Model != "gpt-4o" {
		t.Errorf("model = %q, want gpt-4o (unchanged)", cfg.LLM.Model)
	}
}

func TestMergeEnvInvalidNumericIgnored(t *testing.T) {
	clearVerifierEnv(t)
	t.Setenv("VERIFIER_SEED", "notanumber")
	t.Setenv("VERIFIER_MAX_FINDINGS", "bad")
	t.Setenv("VERIFIER_TIMEOUT", "invalid")
	t.Setenv("VERIFIER_LLM_TEMPERATURE", "xyz")
	t.Setenv("VERIFIER_LLM_MAX_TOKENS", "abc")

	cfg := Defaults()
	mergeEnv(cfg)

	if cfg.Seed != nil {
		t.Errorf("seed = %v, want nil (invalid ignored)", cfg.Seed)
	}
	if cfg.MaxFindings != DefaultMaxFindings {
		t.Errorf("max_findings = %d, want %d (invalid ignored)", cfg.MaxFindings, DefaultMaxFindings)
	}
	if cfg.Timeout != DefaultTimeout {
		t.Errorf("timeout = %v, want %v (invalid ignored)", cfg.Timeout, DefaultTimeout)
	}
}

func TestResolveAPIKey(t *testing.T) {
	// Known provider with key set
	t.Setenv("OPENAI_API_KEY", "test-key-123")
	key, err := resolveAPIKey("openai")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "test-key-123" {
		t.Errorf("key = %q, want test-key-123", key)
	}

	// Known provider with key unset
	t.Setenv("ANTHROPIC_API_KEY", "")
	_, err = resolveAPIKey("anthropic")
	if err == nil {
		t.Error("expected error for empty ANTHROPIC_API_KEY")
	}

	// Gemini provider
	t.Setenv("GEMINI_API_KEY", "gemini-key")
	key, err = resolveAPIKey("gemini")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "gemini-key" {
		t.Errorf("key = %q, want gemini-key", key)
	}

	// Unknown provider
	_, err = resolveAPIKey("unknown-provider")
	if err == nil {
		t.Error("expected error for unknown provider")
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

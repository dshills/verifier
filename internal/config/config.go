package config

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dshills/verifier/internal/domain"
)

const (
	DefaultMode        = "offline"
	DefaultFormat      = "md"
	DefaultMaxFindings = 200
	DefaultTimeout     = 2 * time.Minute
	DefaultFailOn      = "none"
	DefaultConfigFile  = ".verifier.yaml"
)

var defaultSpecPaths = []string{"SPEC.md", "specs/SPEC.md"}
var defaultPlanPaths = []string{"PLAN.md", "plans/PLAN.md"}

var defaultExclude = []string{
	"**/vendor/**",
	"**/node_modules/**",
	"**/*.gen.go",
}

// Defaults returns a Config with built-in default values.
func Defaults() *domain.Config {
	return &domain.Config{
		Mode:        DefaultMode,
		Format:      DefaultFormat,
		Root:        ".",
		Exclude:     defaultExclude,
		FailOn:      DefaultFailOn,
		MaxFindings: DefaultMaxFindings,
		Timeout:     DefaultTimeout,
		ConfigPath:  DefaultConfigFile,
		LLM: domain.LLMConfig{
			Temperature: 0.2,
			MaxTokens:   8000,
		},
	}
}

// Load reads configuration from the YAML file and merges CLI overrides.
func Load(cliCfg *domain.Config) (*domain.Config, error) {
	cfg := Defaults()

	// Determine config path
	configPath := DefaultConfigFile
	if cliCfg != nil && cliCfg.ConfigPath != "" {
		configPath = cliCfg.ConfigPath
	}

	// Try to load YAML config
	yamlCfg, err := loadYAML(configPath)
	if err != nil {
		if os.IsNotExist(err) && configPath == DefaultConfigFile {
			// Default config missing is fine
		} else {
			return nil, fmt.Errorf("config: %w", err)
		}
	}

	// Apply YAML values
	if yamlCfg != nil {
		mergeYAML(cfg, yamlCfg)
	}

	// Apply CLI overrides
	if cliCfg != nil {
		mergeCLI(cfg, cliCfg)
	}

	// Resolve root to absolute path
	if cfg.Root != "" {
		abs, err := filepath.Abs(cfg.Root)
		if err != nil {
			return nil, fmt.Errorf("config: resolve root: %w", err)
		}
		cfg.Root = abs
	}

	// Resolve spec/plan defaults if not overridden
	if len(cfg.SpecPaths) == 0 {
		cfg.SpecPaths = resolveDefaultPaths(cfg.Root, defaultSpecPaths)
	}
	if len(cfg.PlanPaths) == 0 {
		cfg.PlanPaths = resolveDefaultPaths(cfg.Root, defaultPlanPaths)
	}

	if err := Validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func resolveDefaultPaths(root string, candidates []string) []string {
	for _, p := range candidates {
		full := filepath.Join(root, p)
		if _, err := os.Stat(full); err == nil {
			return []string{full}
		}
	}
	return nil
}

func mergeYAML(dst *domain.Config, src *domain.Config) {
	if src.Mode != "" {
		dst.Mode = src.Mode
	}
	if src.Format != "" {
		dst.Format = src.Format
	}
	if len(src.SpecPaths) > 0 {
		dst.SpecPaths = src.SpecPaths
	}
	if len(src.PlanPaths) > 0 {
		dst.PlanPaths = src.PlanPaths
	}
	if len(src.Exclude) > 0 {
		dst.Exclude = src.Exclude
	}
	if len(src.Include) > 0 {
		dst.Include = src.Include
	}
	if src.CI.FailOn != "" {
		dst.FailOn = src.CI.FailOn
	}
	if src.LLM.Provider != "" {
		dst.LLM.Provider = src.LLM.Provider
	}
	if src.LLM.Model != "" {
		dst.LLM.Model = src.LLM.Model
	}
	if src.LLM.Temperature != 0 {
		dst.LLM.Temperature = src.LLM.Temperature
	}
	if src.LLM.MaxTokens != 0 {
		dst.LLM.MaxTokens = src.LLM.MaxTokens
	}
}

// mergeCLI applies CLI flag overrides. List-type flags replace (not merge).
func mergeCLI(dst *domain.Config, cli *domain.Config) {
	if cli.Mode != "" {
		dst.Mode = cli.Mode
	}
	if cli.Format != "" {
		dst.Format = cli.Format
	}
	if cli.Root != "" {
		dst.Root = cli.Root
	}
	if len(cli.SpecPaths) > 0 {
		dst.SpecPaths = cli.SpecPaths
	}
	if len(cli.PlanPaths) > 0 {
		dst.PlanPaths = cli.PlanPaths
	}
	if len(cli.Exclude) > 0 {
		dst.Exclude = cli.Exclude
	}
	if len(cli.Include) > 0 {
		dst.Include = cli.Include
	}
	if cli.FailOn != "" {
		dst.FailOn = cli.FailOn
	}
	if cli.Seed != nil {
		dst.Seed = cli.Seed
	}
	if cli.MaxFindings != 0 {
		dst.MaxFindings = cli.MaxFindings
	}
	if cli.Timeout != 0 {
		dst.Timeout = cli.Timeout
	}
	if cli.SpecCriticPath != "" {
		dst.SpecCriticPath = cli.SpecCriticPath
	}
	if cli.PlanCriticPath != "" {
		dst.PlanCriticPath = cli.PlanCriticPath
	}
	if cli.RealityCheckPath != "" {
		dst.RealityCheckPath = cli.RealityCheckPath
	}
	if cli.PrismPath != "" {
		dst.PrismPath = cli.PrismPath
	}
	if cli.LLM.Provider != "" {
		dst.LLM.Provider = cli.LLM.Provider
	}
	if cli.LLM.Model != "" {
		dst.LLM.Model = cli.LLM.Model
	}
}

// Validate checks the config for invalid values.
func Validate(cfg *domain.Config) error {
	// Mode
	switch cfg.Mode {
	case "offline", "llm":
	default:
		return fmt.Errorf("config: invalid mode %q (must be offline or llm)", cfg.Mode)
	}

	// Format
	switch cfg.Format {
	case "text", "md", "json":
	default:
		return fmt.Errorf("config: invalid format %q (must be text, md, or json)", cfg.Format)
	}

	// FailOn
	switch cfg.FailOn {
	case "none", "low", "medium", "high", "critical":
	default:
		return fmt.Errorf("config: invalid fail-on %q (must be none, low, medium, high, or critical)", cfg.FailOn)
	}

	// Seed range
	if cfg.Seed != nil {
		s := *cfg.Seed
		if s < 0 || s > math.MaxInt32 {
			return fmt.Errorf("config: seed %d out of range (must be 0 to %d)", s, math.MaxInt32)
		}
	}

	// Temperature
	if cfg.LLM.Temperature < 0 || cfg.LLM.Temperature > 2.0 {
		return fmt.Errorf("config: temperature %.1f out of range (must be 0.0 to 2.0)", cfg.LLM.Temperature)
	}

	// Timeout
	if cfg.Mode == "llm" && cfg.Timeout <= 0 {
		return fmt.Errorf("config: timeout must be positive for LLM mode")
	}

	// LLM mode requires provider + model
	if cfg.Mode == "llm" {
		if cfg.LLM.Provider == "" {
			return fmt.Errorf("config: --mode llm requires a provider (set llm.provider)")
		}
		if cfg.LLM.Model == "" {
			return fmt.Errorf("config: --mode llm requires a model (set llm.model)")
		}
	}

	// Validate spec/plan paths
	allPaths := append(cfg.SpecPaths, cfg.PlanPaths...)
	if err := validatePaths(allPaths); err != nil {
		return err
	}

	// Validate glob patterns
	for _, g := range cfg.Include {
		if _, err := filepath.Match(g, ""); err != nil {
			return fmt.Errorf("config: invalid include glob %q: %w", g, err)
		}
	}
	for _, g := range cfg.Exclude {
		if _, err := filepath.Match(g, ""); err != nil {
			return fmt.Errorf("config: invalid exclude glob %q: %w", g, err)
		}
	}

	return nil
}

func validatePaths(paths []string) error {
	seen := make(map[string]bool)
	for _, p := range paths {
		if p == "" {
			continue
		}
		// Dedup
		if seen[p] {
			continue
		}
		seen[p] = true

		info, err := os.Stat(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue // missing files handled at pipeline level
			}
			return fmt.Errorf("config: cannot read %q: %w", p, err)
		}
		if info.IsDir() {
			return fmt.Errorf("config: path %q is a directory, not a file", p)
		}
	}
	return nil
}

// DefaultYAMLContent returns the default .verifier.yaml content.
func DefaultYAMLContent() string {
	return strings.TrimSpace(`
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
`) + "\n"
}

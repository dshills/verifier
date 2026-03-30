package config

import (
	"fmt"
	"log/slog"
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

	// Apply environment variable overrides (between YAML and CLI)
	mergeEnv(cfg)

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

	// Resolve API key from environment for LLM mode (only if not already set)
	if cfg.Mode == "llm" && cfg.LLM.APIKey == "" {
		key, err := resolveAPIKey(cfg.LLM.Provider)
		if err != nil {
			return nil, err
		}
		cfg.LLM.APIKey = key
	}

	return cfg, nil
}

// mergeEnv applies environment variable overrides.
// Priority: defaults < YAML < env vars < CLI flags.
func mergeEnv(cfg *domain.Config) {
	if v := os.Getenv("VERIFIER_MODE"); v != "" {
		cfg.Mode = v
	}
	if v := os.Getenv("VERIFIER_FORMAT"); v != "" {
		cfg.Format = v
	}
	if v := os.Getenv("VERIFIER_ROOT"); v != "" {
		cfg.Root = v
	}
	if v := os.Getenv("VERIFIER_SPEC"); v != "" {
		cfg.SpecPaths = splitCSV(v)
	}
	if v := os.Getenv("VERIFIER_PLAN"); v != "" {
		cfg.PlanPaths = splitCSV(v)
	}
	if v := os.Getenv("VERIFIER_EXCLUDE"); v != "" {
		cfg.Exclude = splitCSV(v)
	}
	if v := os.Getenv("VERIFIER_INCLUDE"); v != "" {
		cfg.Include = splitCSV(v)
	}
	if v := os.Getenv("VERIFIER_FAIL_ON"); v != "" {
		cfg.FailOn = v
	}
	if v := os.Getenv("VERIFIER_SEED"); v != "" {
		var s int
		if _, err := fmt.Sscanf(v, "%d", &s); err != nil {
			slog.Warn("ignoring invalid VERIFIER_SEED", "value", v, "err", err)
		} else {
			cfg.Seed = &s
		}
	}
	if v := os.Getenv("VERIFIER_MAX_FINDINGS"); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
			slog.Warn("ignoring invalid VERIFIER_MAX_FINDINGS", "value", v, "err", err)
		} else {
			cfg.MaxFindings = n
		}
	}
	if v := os.Getenv("VERIFIER_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err != nil {
			slog.Warn("ignoring invalid VERIFIER_TIMEOUT", "value", v, "err", err)
		} else {
			cfg.Timeout = d
		}
	}
	if v := os.Getenv("VERIFIER_LLM_PROVIDER"); v != "" {
		cfg.LLM.Provider = v
	}
	if v := os.Getenv("VERIFIER_LLM_MODEL"); v != "" {
		cfg.LLM.Model = v
	}
	if v := os.Getenv("VERIFIER_LLM_TEMPERATURE"); v != "" {
		var t float64
		if _, err := fmt.Sscanf(v, "%f", &t); err != nil {
			slog.Warn("ignoring invalid VERIFIER_LLM_TEMPERATURE", "value", v, "err", err)
		} else {
			cfg.LLM.Temperature = t
		}
	}
	if v := os.Getenv("VERIFIER_LLM_MAX_TOKENS"); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
			slog.Warn("ignoring invalid VERIFIER_LLM_MAX_TOKENS", "value", v, "err", err)
		} else {
			cfg.LLM.MaxTokens = n
		}
	}
	if v := os.Getenv("VERIFIER_SPEC_CRITIC"); v != "" {
		cfg.SpecCriticPath = v
	}
	if v := os.Getenv("VERIFIER_PLAN_CRITIC"); v != "" {
		cfg.PlanCriticPath = v
	}
	if v := os.Getenv("VERIFIER_REALITY_CHECK"); v != "" {
		cfg.RealityCheckPath = v
	}
	if v := os.Getenv("VERIFIER_PRISM"); v != "" {
		cfg.PrismPath = v
	}
}

// splitCSV splits a comma-separated string into trimmed, non-empty parts.
func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// providerEnvVars maps provider names to their environment variable names.
var providerEnvVars = map[string]string{
	"openai":    "OPENAI_API_KEY",
	"anthropic": "ANTHROPIC_API_KEY",
	"gemini":    "GEMINI_API_KEY",
}

// resolveAPIKey reads the API key from the environment for the given provider.
func resolveAPIKey(provider string) (string, error) {
	envVar, ok := providerEnvVars[provider]
	if !ok {
		return "", fmt.Errorf("config: no known environment variable for provider %q", provider)
	}
	key := os.Getenv(envVar)
	if key == "" {
		return "", fmt.Errorf("config: %s not set (required for provider %q)", envVar, provider)
	}
	return key, nil
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

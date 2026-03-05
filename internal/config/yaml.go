package config

import (
	"fmt"
	"os"

	"github.com/dshills/verifier/internal/domain"
)

// loadYAML reads and parses a YAML config file.
// Returns nil, nil if the file doesn't exist and is the default path.
func loadYAML(path string) (*domain.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &domain.Config{}
	if err := parseYAML(data, cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	return cfg, nil
}

// parseYAML is a minimal YAML parser that handles the .verifier.yaml format.
// We use a simple line-based parser to avoid the yaml.v3 dependency for now,
// keeping the binary stdlib-only. If more complex YAML is needed, swap in yaml.v3.
func parseYAML(data []byte, cfg *domain.Config) error {
	// For now, use a simple key-value parser that handles our config format.
	// This avoids the yaml.v3 dependency per the plan's stdlib preference.
	p := &yamlParser{lines: splitLines(string(data))}
	return p.parse(cfg)
}

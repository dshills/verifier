package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dshills/verifier/internal/domain"
)

type yamlParser struct {
	lines []string
	pos   int
}

func splitLines(s string) []string {
	return strings.Split(s, "\n")
}

func (p *yamlParser) parse(cfg *domain.Config) error {
	var currentSection string
	var currentList *[]string

	for p.pos < len(p.lines) {
		line := p.lines[p.pos]
		p.pos++

		// Skip empty lines and comments
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check for list item
		if strings.HasPrefix(trimmed, "- ") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			val = unquoteYAML(val)
			if currentList != nil {
				*currentList = append(*currentList, val)
			}
			continue
		}

		// Key-value pair
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// Section headers (value is empty)
		if val == "" {
			currentSection = key
			currentList = nil
			switch currentSection {
			case "spec":
				currentList = &cfg.SpecPaths
			case "plan":
				currentList = &cfg.PlanPaths
			case "exclude":
				currentList = &cfg.Exclude
			case "include":
				currentList = &cfg.Include
			}
			continue
		}

		val = unquoteYAML(val)

		// Assign based on section context
		switch currentSection {
		case "":
			if err := p.setTopLevel(cfg, key, val); err != nil {
				return err
			}
		case "ci":
			p.setCI(cfg, key, val)
		case "llm":
			if err := p.setLLM(cfg, key, val); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *yamlParser) setTopLevel(cfg *domain.Config, key, val string) error {
	switch key {
	case "mode":
		cfg.Mode = val
	case "format":
		cfg.Format = val
	}
	return nil
}

func (p *yamlParser) setCI(cfg *domain.Config, key, val string) {
	switch key {
	case "fail_on":
		cfg.CI.FailOn = val
	}
}

func (p *yamlParser) setLLM(cfg *domain.Config, key, val string) error {
	switch key {
	case "provider":
		cfg.LLM.Provider = val
	case "model":
		cfg.LLM.Model = val
	case "temperature":
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return fmt.Errorf("invalid temperature %q: %w", val, err)
		}
		cfg.LLM.Temperature = f
	case "max_tokens":
		n, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid max_tokens %q: %w", val, err)
		}
		cfg.LLM.MaxTokens = n
	}
	return nil
}

func unquoteYAML(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

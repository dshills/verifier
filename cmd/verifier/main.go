package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/dshills/verifier/internal/config"
	"github.com/dshills/verifier/internal/domain"
	"github.com/dshills/verifier/internal/ecosystem"
	"github.com/dshills/verifier/internal/gaps"
	golangpkg "github.com/dshills/verifier/internal/golang"
	"github.com/dshills/verifier/internal/mapping"
	"github.com/dshills/verifier/internal/parse"
	"github.com/dshills/verifier/internal/pipeline"
	"github.com/dshills/verifier/internal/ranking"
	"github.com/dshills/verifier/internal/repo"
	"github.com/dshills/verifier/internal/report"
	"github.com/dshills/verifier/internal/scaffold"
	"github.com/dshills/verifier/internal/strategy"
)

var version = "dev"

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "analyze":
		os.Exit(runAnalyze(os.Args[2:]))
	case "init":
		os.Exit(runInit(os.Args[2:]))
	case "explain":
		os.Exit(runExplain(os.Args[2:]))
	case "scaffold":
		os.Exit(runScaffold(os.Args[2:]))
	case "version", "--version", "-v":
		fmt.Println(version)
		os.Exit(0)
	case "help", "--help", "-h":
		printUsage()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: verifier <command> [flags]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  analyze    Analyze repository for test gaps")
	fmt.Fprintln(os.Stderr, "  init       Create default .verifier.yaml")
	fmt.Fprintln(os.Stderr, "  explain    Explain a specific TESTREC recommendation")
	fmt.Fprintln(os.Stderr, "  scaffold   Generate skeleton test files (Phase 3)")
	fmt.Fprintln(os.Stderr, "  version    Print version")
	fmt.Fprintln(os.Stderr, "  help       Show this help")
}

func runAnalyze(args []string) int {
	fs := flag.NewFlagSet("analyze", flag.ContinueOnError)
	cliCfg := &domain.Config{}

	fs.StringVar(&cliCfg.Mode, "mode", "", "Analysis mode: offline or llm")
	fs.StringVar(&cliCfg.Format, "format", "", "Output format: json, md, or text")
	fs.StringVar(&cliCfg.Root, "root", "", "Repository root directory")
	fs.StringVar(&cliCfg.FailOn, "fail-on", "", "Fail if severity >= threshold (none, low, medium, high, critical)")
	fs.StringVar(&cliCfg.ConfigPath, "config", "", "Path to config file")
	fs.IntVar(&cliCfg.MaxFindings, "max-findings", 0, "Maximum findings to report (0 = unlimited)")

	var specPaths, planPaths, exclude, include string
	fs.StringVar(&specPaths, "spec", "", "Comma-separated spec file paths")
	fs.StringVar(&planPaths, "plan", "", "Comma-separated plan file paths")
	fs.StringVar(&exclude, "exclude", "", "Comma-separated exclude globs")
	fs.StringVar(&include, "include", "", "Comma-separated include globs")

	var seed int
	var hasSeed bool
	fs.Func("seed", "Deterministic seed value", func(s string) error {
		_, err := fmt.Sscanf(s, "%d", &seed)
		if err != nil {
			return fmt.Errorf("invalid seed: %w", err)
		}
		hasSeed = true
		return nil
	})

	var timeout string
	fs.StringVar(&timeout, "timeout", "", "Analysis timeout (e.g. 2m, 30s)")

	// LLM flags
	fs.StringVar(&cliCfg.LLM.Provider, "llm-provider", "", "LLM provider")
	fs.StringVar(&cliCfg.LLM.Model, "llm-model", "", "LLM model")

	// External tool inputs
	fs.StringVar(&cliCfg.SpecCriticPath, "spec-critic", "", "Path to SpecCritic JSON output")
	fs.StringVar(&cliCfg.PlanCriticPath, "plan-critic", "", "Path to PlanCritic JSON output")
	fs.StringVar(&cliCfg.RealityCheckPath, "reality-check", "", "Path to RealityCheck JSON output")
	fs.StringVar(&cliCfg.PrismPath, "prism", "", "Path to Prism JSON output")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if specPaths != "" {
		cliCfg.SpecPaths = splitCSV(specPaths)
	}
	if planPaths != "" {
		cliCfg.PlanPaths = splitCSV(planPaths)
	}
	if exclude != "" {
		cliCfg.Exclude = splitCSV(exclude)
	}
	if include != "" {
		cliCfg.Include = splitCSV(include)
	}
	if hasSeed {
		cliCfg.Seed = &seed
	}
	if timeout != "" {
		d, err := time.ParseDuration(timeout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: invalid timeout %q: %v\n", timeout, err)
			return 1
		}
		cliCfg.Timeout = d
	}

	cfg, err := config.Load(cliCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}

	ctx := context.Background()
	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	// Build pipeline with all stages
	p := pipeline.New(
		repo.Stage{},
		parse.Stage{},
		golangpkg.Stage{},
		mapping.Stage{},
		strategy.Stage{},
		gaps.Stage{},
		ranking.Stage{},
	)

	arts, err := p.Run(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}

	// Apply ecosystem tool integrations
	applyEcosystem(cfg, arts)

	// Build report
	rpt := report.Build(arts, cfg, version)

	// Write output
	switch cfg.Format {
	case "json":
		if err := report.WriteJSON(os.Stdout, rpt); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			return 1
		}
	case "md":
		if err := report.WriteMarkdown(os.Stdout, rpt); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			return 1
		}
	case "text":
		if err := report.WriteText(os.Stdout, rpt); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			return 1
		}
	}

	// Check fail-on threshold
	if ranking.CheckFailOn(rpt.Recommendations, cfg.FailOn) {
		return 2
	}

	return 0
}

func runInit(args []string) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	var force bool
	fs.BoolVar(&force, "force", false, "Overwrite existing config file")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	path := config.DefaultConfigFile

	if !force {
		if _, err := os.Stat(path); err == nil {
			fmt.Fprintf(os.Stderr, "ERROR: %s already exists (use --force to overwrite)\n", path)
			return 1
		}
	}

	content := config.DefaultYAMLContent()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to write %s: %v\n", path, err)
		return 1
	}

	fmt.Fprintf(os.Stderr, "created %s\n", path)
	return 0
}

func runExplain(args []string) int {
	fs := flag.NewFlagSet("explain", flag.ContinueOnError)
	var inputFile string
	fs.StringVar(&inputFile, "input", "", "Path to prior analysis JSON")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: verifier explain [--input <file>] <TESTREC-ID>")
		return 1
	}

	testrecID := fs.Arg(0)

	var rpt *domain.Report
	var err error

	// Load report from input file or stdin
	if inputFile != "" {
		rpt, err = report.LoadJSON(inputFile)
	} else {
		// Check if stdin has data
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			if inputFile != "" {
				slog.Info("both --input and stdin provided; using --input")
			}
			rpt, err = report.ReadJSON(os.Stdin)
		}
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}

	if rpt == nil {
		fmt.Fprintln(os.Stderr, "ERROR: no input provided. Use --input <file> or pipe JSON to stdin")
		return 1
	}

	// Find recommendation
	for _, rec := range rpt.Recommendations {
		if rec.ID == testrecID {
			report.ExplainRecommendation(os.Stdout, &rec)
			return 0
		}
	}

	fmt.Fprintf(os.Stderr, "ERROR: TESTREC ID %q not found in report\n", testrecID)
	return 1
}

func runScaffold(args []string) int {
	fs := flag.NewFlagSet("scaffold", flag.ContinueOnError)
	var inputFile, style string
	var limit int
	var dryRun, write bool
	fs.StringVar(&inputFile, "input", "", "Path to prior analysis JSON")
	fs.IntVar(&limit, "limit", 0, "Limit to top N critical recommendations")
	fs.BoolVar(&dryRun, "dry-run", true, "Print planned changes without writing")
	fs.BoolVar(&write, "write", false, "Execute file modifications")
	fs.StringVar(&style, "style", "std", "Test style: std or go-testify")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if write {
		dryRun = false
	}

	var rpt *domain.Report
	var err error

	if inputFile != "" {
		rpt, err = report.LoadJSON(inputFile)
	} else {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			rpt, err = report.ReadJSON(os.Stdin)
		}
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}

	if rpt == nil {
		fmt.Fprintln(os.Stderr, "ERROR: no input provided. Use --input <file> or pipe JSON to stdin")
		return 1
	}

	actions, selected := scaffold.Plan(rpt.Recommendations, limit)
	if len(actions) == 0 {
		fmt.Fprintln(os.Stderr, "NOTICE: no recommendations to scaffold")
		return 0
	}

	if err := scaffold.Execute(actions, selected, style, dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}

	return 0
}

func applyEcosystem(cfg *domain.Config, arts *domain.Artifacts) {
	if cfg.SpecCriticPath != "" {
		sc, err := ecosystem.LoadSpecCritic(cfg.SpecCriticPath)
		if err != nil {
			slog.Warn("failed to load spec-critic", "err", err)
		} else if sc != nil {
			ecosystem.ApplySpecCritic(&arts.Recommendations, sc)
		}
	}
	if cfg.PlanCriticPath != "" {
		pc, err := ecosystem.LoadPlanCritic(cfg.PlanCriticPath)
		if err != nil {
			slog.Warn("failed to load plan-critic", "err", err)
		} else if pc != nil {
			ecosystem.ApplyPlanCritic(&arts.Recommendations, pc)
		}
	}
	if cfg.RealityCheckPath != "" {
		rc, err := ecosystem.LoadRealityCheck(cfg.RealityCheckPath)
		if err != nil {
			slog.Warn("failed to load reality-check", "err", err)
		} else if rc != nil {
			ecosystem.ApplyRealityCheck(&arts.Recommendations, rc)
		}
	}
	if cfg.PrismPath != "" {
		pr, err := ecosystem.LoadPrism(cfg.PrismPath)
		if err != nil {
			slog.Warn("failed to load prism", "err", err)
		} else if pr != nil {
			ecosystem.ApplyPrism(&arts.Recommendations, pr)
		}
	}
}

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

package repo

import (
	"bufio"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/dshills/verifier/internal/domain"
)

const maxFileSize = 1 << 20 // 1MB

// ScanRepo walks the repository and builds the RepoGraph.
func ScanRepo(cfg *domain.Config) (*domain.RepoGraph, error) {
	graph := &domain.RepoGraph{ModuleRoot: cfg.Root}

	// Parse go.mod for module path
	gomodPath := filepath.Join(cfg.Root, "go.mod")
	if modPath, err := parseGoModPath(gomodPath); err == nil {
		graph.ModulePath = modPath
	}

	// Collect packages by walking the tree
	pkgMap := make(map[string]*domain.PackageInfo)

	err := filepath.WalkDir(cfg.Root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		// Skip excluded directories
		if d.IsDir() {
			rel, _ := filepath.Rel(cfg.Root, path)
			if shouldExclude(rel, cfg.Exclude) {
				return filepath.SkipDir
			}
			return nil
		}

		if filepath.Ext(path) != ".go" {
			return nil
		}

		rel, _ := filepath.Rel(cfg.Root, path)
		if shouldExclude(rel, cfg.Exclude) {
			return nil
		}
		if len(cfg.Include) > 0 && !shouldInclude(rel, cfg.Include) {
			return nil
		}

		// Skip files > 1MB
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.Size() > maxFileSize {
			slog.Warn("skipping large file", "path", rel, "size", info.Size())
			return nil
		}

		dir := filepath.Dir(path)
		relDir, _ := filepath.Rel(cfg.Root, dir)
		if relDir == "." {
			relDir = ""
		}

		pkg, ok := pkgMap[relDir]
		if !ok {
			pkgName := filepath.Base(dir)
			if relDir == "" {
				pkgName = filepath.Base(cfg.Root)
			}
			pkg = &domain.PackageInfo{
				Path:  buildImportPath(graph.ModulePath, relDir),
				Name:  pkgName,
				Dir:   dir,
				IsCmd: isCmdPackage(relDir),
			}
			pkgMap[relDir] = pkg
		}

		fileName := filepath.Base(path)
		if strings.HasSuffix(fileName, "_test.go") {
			pkg.TestFiles = append(pkg.TestFiles, path)
			pkg.HasTests = true
		} else {
			pkg.GoFiles = append(pkg.GoFiles, path)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	for _, pkg := range pkgMap {
		graph.Packages = append(graph.Packages, *pkg)
	}

	return graph, nil
}

func parseGoModPath(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", scanner.Err()
}

func buildImportPath(modulePath, relDir string) string {
	if modulePath == "" {
		return relDir
	}
	if relDir == "" {
		return modulePath
	}
	return modulePath + "/" + filepath.ToSlash(relDir)
}

func isCmdPackage(relDir string) bool {
	parts := strings.Split(filepath.ToSlash(relDir), "/")
	return len(parts) > 0 && parts[0] == "cmd"
}

func shouldExclude(rel string, patterns []string) bool {
	rel = filepath.ToSlash(rel)
	for _, p := range patterns {
		if matched, _ := filepath.Match(p, rel); matched {
			return true
		}
		// Also try matching against just the filename
		if matched, _ := filepath.Match(p, filepath.Base(rel)); matched {
			return true
		}
		// Check if any path segment matches
		if matchDoublestar(p, rel) {
			return true
		}
	}
	return false
}

func shouldInclude(rel string, patterns []string) bool {
	rel = filepath.ToSlash(rel)
	for _, p := range patterns {
		if matched, _ := filepath.Match(p, rel); matched {
			return true
		}
		if matched, _ := filepath.Match(p, filepath.Base(rel)); matched {
			return true
		}
		if matchDoublestar(p, rel) {
			return true
		}
	}
	return false
}

// matchDoublestar handles **/ glob patterns by checking if any path segment
// matches the inner pattern.
func matchDoublestar(pattern, path string) bool {
	if !strings.Contains(pattern, "**") {
		return false
	}

	// Extract the meaningful parts between ** markers
	// e.g. "**/vendor/**" → check if "vendor" is a path segment
	// e.g. "**/*.gen.go" → check if any segment matches "*.gen.go"
	cleaned := strings.ReplaceAll(pattern, "/**/", "/")
	cleaned = strings.TrimPrefix(cleaned, "**/")
	cleaned = strings.TrimSuffix(cleaned, "/**")

	if cleaned == "" || cleaned == "**" {
		return true
	}

	segments := strings.Split(path, "/")
	matchParts := strings.Split(cleaned, "/")

	// For single-part patterns, check each segment
	if len(matchParts) == 1 {
		for _, seg := range segments {
			if matched, _ := filepath.Match(matchParts[0], seg); matched {
				return true
			}
		}
		return false
	}

	// For multi-part patterns (e.g. "cmd/app"), check if they appear as consecutive segments
	for i := 0; i <= len(segments)-len(matchParts); i++ {
		allMatch := true
		for j, mp := range matchParts {
			if matched, _ := filepath.Match(mp, segments[i+j]); !matched {
				allMatch = false
				break
			}
		}
		if allMatch {
			return true
		}
	}
	return false
}

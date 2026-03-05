package repo

import (
	"bufio"
	"os"
	"regexp"
	"strings"

	"github.com/dshills/verifier/internal/domain"
)

var httpImports = map[string]bool{
	"net/http":                 true,
	"github.com/gorilla/mux":   true,
	"github.com/gin-gonic/gin": true,
	"github.com/go-chi/chi":    true,
	"github.com/go-chi/chi/v5": true,
}

var dbImports = map[string]bool{
	"database/sql": true,
}

var fsCallPatterns = []string{
	"os.Open", "os.Create", "os.ReadFile", "os.WriteFile",
	"ioutil.ReadFile", "ioutil.WriteFile", "ioutil.ReadAll",
}

var externalImports = map[string]bool{
	"github.com/go-resty/resty":    true,
	"github.com/go-resty/resty/v2": true,
}

var mqImports = map[string]bool{
	"github.com/streadway/amqp":      true,
	"github.com/segmentio/kafka-go":  true,
	"github.com/nats-io/nats.go":     true,
	"github.com/rabbitmq/amqp091-go": true,
}

var dbCallPattern = regexp.MustCompile(`\.(Query|QueryContext|QueryRow|QueryRowContext|Exec|ExecContext)\(`)
var handlerSigPattern = regexp.MustCompile(`func\s+\w*\s*\([^)]*http\.ResponseWriter\s*,\s*\*http\.Request`)

// DetectBoundaries scans Go files in the repo graph for external boundaries.
func DetectBoundaries(graph *domain.RepoGraph) *domain.BoundaryMap {
	bm := &domain.BoundaryMap{}

	for _, pkg := range graph.Packages {
		for _, goFile := range pkg.GoFiles {
			scanFileForBoundaries(goFile, pkg.Name, bm)
		}
	}

	return bm
}

func scanFileForBoundaries(path, pkgName string, bm *domain.BoundaryMap) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	lineNum := 0
	inImports := false

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Track import blocks
		if strings.HasPrefix(trimmed, "import (") || strings.HasPrefix(trimmed, "import(") {
			inImports = true
			continue
		}
		if inImports {
			if trimmed == ")" {
				inImports = false
				continue
			}
			importPath := extractImportPath(trimmed)
			if importPath == "" {
				continue
			}
			entry := domain.BoundaryEntry{
				File:   path,
				Symbol: pkgName,
				Line:   lineNum,
			}
			if httpImports[importPath] {
				entry.Kind = "http"
				bm.HTTP = append(bm.HTTP, entry)
			}
			if dbImports[importPath] {
				entry.Kind = "db"
				bm.DB = append(bm.DB, entry)
			}
			if externalImports[importPath] {
				entry.Kind = "external"
				bm.External = append(bm.External, entry)
			}
			if mqImports[importPath] {
				entry.Kind = "mq"
				bm.MQ = append(bm.MQ, entry)
			}
			continue
		}

		// Single-line imports
		if strings.HasPrefix(trimmed, "import ") && !strings.Contains(trimmed, "(") {
			importPath := extractImportPath(trimmed[7:])
			entry := domain.BoundaryEntry{
				File:   path,
				Symbol: pkgName,
				Line:   lineNum,
			}
			if httpImports[importPath] {
				entry.Kind = "http"
				bm.HTTP = append(bm.HTTP, entry)
			}
			if dbImports[importPath] {
				entry.Kind = "db"
				bm.DB = append(bm.DB, entry)
			}
		}

		// FS calls
		for _, pat := range fsCallPatterns {
			if strings.Contains(trimmed, pat) {
				bm.FS = append(bm.FS, domain.BoundaryEntry{
					File:   path,
					Symbol: pkgName,
					Line:   lineNum,
					Kind:   "fs",
				})
				break
			}
		}

		// DB calls
		if dbCallPattern.MatchString(trimmed) {
			bm.DB = append(bm.DB, domain.BoundaryEntry{
				File:   path,
				Symbol: pkgName,
				Line:   lineNum,
				Kind:   "db",
			})
		}

		// HTTP handler signatures
		if handlerSigPattern.MatchString(trimmed) {
			bm.HTTP = append(bm.HTTP, domain.BoundaryEntry{
				File:   path,
				Symbol: pkgName,
				Line:   lineNum,
				Kind:   "http",
			})
		}
	}
}

func extractImportPath(s string) string {
	s = strings.TrimSpace(s)
	// Handle aliased imports: alias "path"
	if idx := strings.Index(s, `"`); idx >= 0 {
		s = s[idx:]
	}
	s = strings.Trim(s, `"`)
	return s
}

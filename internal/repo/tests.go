package repo

import (
	"bufio"
	"os"
	"regexp"

	"github.com/dshills/verifier/internal/domain"
)

var testFuncPattern = regexp.MustCompile(`^func\s+(Test\w+)\s*\(`)

// CollectTests scans test files and extracts test function names.
func CollectTests(graph *domain.RepoGraph) *domain.TestInventory {
	inv := &domain.TestInventory{}

	for _, pkg := range graph.Packages {
		for _, testFile := range pkg.TestFiles {
			extractTestFuncs(testFile, pkg.Name, inv)
		}
	}

	return inv
}

func extractTestFuncs(path, pkgName string, inv *domain.TestInventory) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		matches := testFuncPattern.FindStringSubmatch(line)
		if len(matches) >= 2 {
			inv.Tests = append(inv.Tests, domain.TestInfo{
				File:     path,
				Package:  pkgName,
				FuncName: matches[1],
				Line:     lineNum,
			})
		}
	}
}

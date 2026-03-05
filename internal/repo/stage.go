package repo

import (
	"context"

	"github.com/dshills/verifier/internal/domain"
)

// Stage implements Stage A: Repository Inventory.
type Stage struct{}

func (Stage) Name() string { return "repo-inventory" }

func (Stage) Execute(ctx context.Context, cfg *domain.Config, arts *domain.Artifacts) error {
	graph, err := ScanRepo(cfg)
	if err != nil {
		return err
	}
	arts.RepoGraph = graph

	boundaries := DetectBoundaries(graph)
	arts.BoundaryMap = boundaries

	tests := CollectTests(graph)
	arts.TestInventory = tests

	return nil
}

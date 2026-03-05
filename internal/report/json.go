package report

import (
	"encoding/json"
	"io"

	"github.com/dshills/verifier/internal/domain"
)

// WriteJSON writes the report as indented JSON.
func WriteJSON(w io.Writer, report *domain.Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

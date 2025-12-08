// Package exporter provides functionality to export collected resources to various formats.
// Package exporter provides CSV export functionality.
package exporter

import (
	"encoding/csv"
	"fmt"
	"io"

	"github.com/y-miyazaki/arc/internal/aws/resources"
)

// WriteCSV writes the collected resources to a CSV writer
func WriteCSV(w io.Writer, res []resources.Resource, columns []resources.Column) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	// Write header
	var headers []string
	for _, col := range columns {
		headers = append(headers, col.Header)
	}
	if err := cw.Write(headers); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write rows
	for i := range res {
		r := res[i]
		var row []string
		for _, col := range columns {
			row = append(row, col.Value(r))
		}
		if err := cw.Write(row); err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}
	}

	return nil
}

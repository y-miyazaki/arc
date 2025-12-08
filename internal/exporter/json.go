// Package exporter provides functionality to export collected resources to various formats.
package exporter

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/y-miyazaki/arc/internal/aws/resources"
)

// WriteJSON writes the collected resources to a JSON writer
func WriteJSON(w io.Writer, res []resources.Resource, columns []resources.Column) error {
	// Create a map for each resource with column headers as keys
	var jsonData []map[string]string

	for i := range res {
		resource := &res[i]
		row := make(map[string]string)
		for _, col := range columns {
			row[col.Header] = col.Value(*resource)
		}
		jsonData = append(jsonData, row)
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ") // Pretty print
	if err := encoder.Encode(jsonData); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

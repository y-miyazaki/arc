// Package exporter provides functionality to export collected resources to various formats.
package exporter

import (
	"archive/zip"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

//go:embed html_template.html
var htmlTemplate string

// HTMLTemplateData represents the data structure for HTML template substitution
type HTMLTemplateData struct {
	Title       string
	Description string
	OutputFile  string
}

// FileManifestEntry represents a single entry in the files.json manifest
type FileManifestEntry struct {
	Path        string `json:"path"`
	DisplayName string `json:"display_name"` //nolint:tagliatelle // matches JavaScript naming convention
}

// GenerateHTML generates HTML index and manifest files for CSV outputs
func GenerateHTML(outputDir, accountID, outputFile string, categories []string) error {
	// Generate files.json manifest
	manifestPath := filepath.Join(outputDir, accountID, "files.json")
	if err := generateManifest(manifestPath, outputDir, accountID, categories); err != nil {
		return fmt.Errorf("failed to generate manifest: %w", err)
	}

	// Create ZIP file containing all CSV resources
	resourcesDir := filepath.Join(outputDir, accountID, "resources")
	zipPath := filepath.Join(outputDir, accountID, "resources.zip")
	if err := createResourcesZip(zipPath, resourcesDir); err != nil {
		return fmt.Errorf("failed to create resources.zip: %w", err)
	}

	// Generate index.html
	indexPath := filepath.Join(outputDir, accountID, "index.html")
	if err := generateIndexHTML(indexPath, accountID, outputFile); err != nil {
		return fmt.Errorf("failed to generate index.html: %w", err)
	}

	return nil
}

// generateManifest creates files.json with the list of CSV files
func generateManifest(manifestPath, outputDir, accountID string, categories []string) error {
	var entries []FileManifestEntry

	resourcesDir := filepath.Join(outputDir, accountID, "resources")
	for _, category := range categories {
		csvPath := filepath.Join(resourcesDir, category+".csv")
		// Check if file exists
		if _, statErr := os.Stat(csvPath); statErr == nil {
			// Create relative path from output directory
			relPath := filepath.Join("resources", category+".csv")
			entries = append(entries, FileManifestEntry{
				Path:        relPath,
				DisplayName: category,
			})
		}
	}

	// Write manifest file
	f, err := os.Create(manifestPath) //nolint:gosec // G304: Path is controlled and sanitized
	if err != nil {
		return fmt.Errorf("failed to create manifest file: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			// Log error but don't override return error
			_, _ = fmt.Fprintf(os.Stderr, "failed to close manifest file: %v\n", cerr)
		}
	}()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if encErr := encoder.Encode(entries); encErr != nil {
		return fmt.Errorf("failed to encode manifest: %w", encErr)
	}

	return nil
}

// createResourcesZip creates a ZIP archive of all CSV files in the resources directory
func createResourcesZip(zipPath, resourcesDir string) error {
	// Check if resources directory exists
	if _, statErr := os.Stat(resourcesDir); os.IsNotExist(statErr) {
		// If resources directory doesn't exist, create an empty ZIP
		zipFile, err := os.Create(zipPath) //nolint:gosec // G304: Path is controlled and sanitized
		if err != nil {
			return fmt.Errorf("failed to create ZIP file: %w", err)
		}
		defer func() {
			if cerr := zipFile.Close(); cerr != nil {
				_, _ = fmt.Fprintf(os.Stderr, "failed to close ZIP file: %v\n", cerr)
			}
		}()
		zipWriter := zip.NewWriter(zipFile)
		defer func() {
			if cerr := zipWriter.Close(); cerr != nil {
				_, _ = fmt.Fprintf(os.Stderr, "failed to close ZIP writer: %v\n", cerr)
			}
		}()
		return nil
	}

	// Create ZIP file
	zipFile, err := os.Create(zipPath) //nolint:gosec // G304: Path is controlled and sanitized
	if err != nil {
		return fmt.Errorf("failed to create ZIP file: %w", err)
	}
	defer func() {
		if cerr := zipFile.Close(); cerr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "failed to close ZIP file: %v\n", cerr)
		}
	}()

	zipWriter := zip.NewWriter(zipFile)
	defer func() {
		if cerr := zipWriter.Close(); cerr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "failed to close ZIP writer: %v\n", cerr)
		}
	}()

	// Walk through resources directory and add CSV files to ZIP
	if walkErr := filepath.Walk(resourcesDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only include CSV files
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".csv") {
			return nil
		}

		// Open source file
		srcFile, openErr := os.Open(path) //nolint:gosec // G304: Path comes from controlled Walk operation
		if openErr != nil {
			return fmt.Errorf("failed to open source file %s: %w", path, openErr)
		}
		defer func() {
			if cerr := srcFile.Close(); cerr != nil {
				_, _ = fmt.Fprintf(os.Stderr, "failed to close source file: %v\n", cerr)
			}
		}()

		// Get relative path for ZIP entry
		relPath, relErr := filepath.Rel(resourcesDir, path)
		if relErr != nil {
			return fmt.Errorf("failed to get relative path: %w", relErr)
		}

		// Create ZIP entry
		zipEntry, createErr := zipWriter.Create(relPath)
		if createErr != nil {
			return fmt.Errorf("failed to create ZIP entry: %w", createErr)
		}

		// Copy file content to ZIP
		if _, copyErr := io.Copy(zipEntry, srcFile); copyErr != nil {
			return fmt.Errorf("failed to copy file to ZIP: %w", copyErr)
		}

		return nil
	}); walkErr != nil {
		return fmt.Errorf("failed to walk resources directory: %w", walkErr)
	}

	return nil
}

// generateIndexHTML creates index.html with embedded template
func generateIndexHTML(indexPath, accountID, outputFile string) error {
	title := fmt.Sprintf("AWS Resources (%s)", accountID)
	description := "AWS resource inventory collected by arc"

	// Substitute placeholders in template
	html := htmlTemplate
	html = strings.ReplaceAll(html, "@@INDEX_TITLE@@", title)
	html = strings.ReplaceAll(html, "@@INDEX_DESCRIPTION@@", description)
	html = strings.ReplaceAll(html, "@@OUTPUT_FILE@@", outputFile)

	// Write HTML file
	f, err := os.Create(indexPath) //nolint:gosec // G304: Path is controlled and sanitized
	if err != nil {
		return fmt.Errorf("failed to create index.html: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "failed to close index.html: %v\n", cerr)
		}
	}()

	if _, writeErr := f.WriteString(html); writeErr != nil {
		return fmt.Errorf("failed to write index.html: %w", writeErr)
	}

	return nil
}

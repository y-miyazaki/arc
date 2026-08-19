// Package exporter provides functionality to export collected resources to various formats.
package exporter

import (
	"archive/zip"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"html"
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

func closeAndJoin(err error, c io.Closer, msg string) error {
	if c == nil {
		return err
	}
	if cerr := c.Close(); cerr != nil {
		return errors.Join(err, fmt.Errorf("%s: %w", msg, cerr))
	}
	return err
}

// GenerateHTML generates HTML index and manifest files for CSV outputs.
func GenerateHTML(ctx context.Context, outputDir, accountID, accountDisplay, outputFile string, categories []string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context canceled: %w", err)
	}
	// Generate files.json manifest
	manifestPath := filepath.Join(outputDir, accountID, "files.json")
	if err := generateManifest(manifestPath, outputDir, accountID, categories); err != nil {
		return fmt.Errorf("failed to generate manifest: %w", err)
	}

	// Create ZIP file containing all CSV resources
	resourcesDir := filepath.Join(outputDir, accountID, "resources")
	zipPath := filepath.Join(outputDir, accountID, "resources.zip")
	if err := createResourcesZip(ctx, zipPath, resourcesDir); err != nil {
		return fmt.Errorf("failed to create resources.zip: %w", err)
	}

	// Generate index.html
	indexPath := filepath.Join(outputDir, accountID, "index.html")
	if err := generateIndexHTML(ctx, indexPath, accountID, accountDisplay, outputFile); err != nil {
		return fmt.Errorf("failed to generate index.html: %w", err)
	}

	return nil
}

// generateManifest creates files.json with the list of CSV files
func generateManifest(manifestPath, outputDir, accountID string, categories []string) (err error) {
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
	var f *os.File
	f, err = os.Create(manifestPath) //nolint:gosec // G304: Path is controlled and sanitized
	if err != nil {
		return fmt.Errorf("failed to create manifest file: %w", err)
	}
	defer func() {
		err = closeAndJoin(err, f, "failed to close manifest file")
	}()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if encErr := encoder.Encode(entries); encErr != nil {
		return fmt.Errorf("failed to encode manifest: %w", encErr)
	}

	return nil
}

// createResourcesZip creates a ZIP archive of all CSV files in the resources directory
func createResourcesZip(ctx context.Context, zipPath, resourcesDir string) (err error) {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("context canceled: %w", ctxErr)
	}
	// Check if resources directory exists
	if _, statErr := os.Stat(resourcesDir); os.IsNotExist(statErr) {
		// If resources directory doesn't exist, create an empty ZIP
		var zipFile *os.File
		zipFile, err = os.Create(zipPath) //nolint:gosec // G304: Path is controlled and sanitized
		if err != nil {
			return fmt.Errorf("failed to create ZIP file: %w", err)
		}
		defer func() {
			err = closeAndJoin(err, zipFile, "failed to close ZIP file")
		}()
		zipWriter := zip.NewWriter(zipFile)
		defer func() {
			err = closeAndJoin(err, zipWriter, "failed to close ZIP writer")
		}()
		return nil
	}

	// Create ZIP file
	var zipFile *os.File
	zipFile, err = os.Create(zipPath) //nolint:gosec // G304: Path is controlled and sanitized
	if err != nil {
		return fmt.Errorf("failed to create ZIP file: %w", err)
	}
	defer func() {
		err = closeAndJoin(err, zipFile, "failed to close ZIP file")
	}()

	zipWriter := zip.NewWriter(zipFile)
	defer func() {
		err = closeAndJoin(err, zipWriter, "failed to close ZIP writer")
	}()

	// Walk through resources directory and add CSV files to ZIP
	if walkErr := filepath.Walk(resourcesDir, func(path string, info os.FileInfo, walkErr error) (walkRet error) {
		if walkErr != nil {
			return walkErr
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return fmt.Errorf("context canceled: %w", ctxErr)
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
			walkRet = closeAndJoin(walkRet, srcFile, "failed to close source file")
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

// generateIndexHTML creates index.html with embedded template.
func generateIndexHTML(ctx context.Context, indexPath, accountID, accountDisplay, outputFile string) (err error) {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("context canceled: %w", ctxErr)
	}
	displayAccount := accountID
	if accountDisplay != "" {
		displayAccount = accountDisplay
	}

	title := html.EscapeString(fmt.Sprintf("AWS Resources (%s)", displayAccount))
	description := html.EscapeString("AWS resource inventory collected by arc")
	safeOutputFile := html.EscapeString(outputFile)
	safeAccount := html.EscapeString(displayAccount)

	// Substitute placeholders in template
	page := htmlTemplate
	page = strings.ReplaceAll(page, "@@INDEX_TITLE@@", title)
	page = strings.ReplaceAll(page, "@@INDEX_DESCRIPTION@@", description)
	page = strings.ReplaceAll(page, "@@OUTPUT_FILE@@", safeOutputFile)
	page = strings.ReplaceAll(page, "@@ACCOUNT_ID@@", safeAccount)

	// Write HTML file
	var f *os.File
	f, err = os.Create(indexPath) //nolint:gosec // G304: Path is controlled and sanitized
	if err != nil {
		return fmt.Errorf("failed to create index.html: %w", err)
	}
	defer func() {
		err = closeAndJoin(err, f, "failed to close index.html")
	}()

	if _, writeErr := f.WriteString(page); writeErr != nil {
		return fmt.Errorf("failed to write index.html: %w", writeErr)
	}

	return nil
}

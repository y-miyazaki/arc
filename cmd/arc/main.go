// Package main is the entry point for the arc application.
package main

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/urfave/cli/v2"

	"github.com/y-miyazaki/arc/internal/aws"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
	"github.com/y-miyazaki/arc/internal/aws/resources"
	"github.com/y-miyazaki/arc/internal/exporter"
	"github.com/y-miyazaki/arc/internal/logger"
	"github.com/y-miyazaki/arc/internal/validation"
)

// Version information (set by GoReleaser during build)
const (
	// LogKeyCategory is the log key for category
	LogKeyCategory = "category"
	// LogKeyError is the log key for error
	LogKeyError = "error"
	// LogKeyFile is the log key for file
	LogKeyFile = "file"

	// DefaultDirPerm is the default directory permission
	DefaultDirPerm = 0750
	// GlobalServiceRegion is the region used for global services (IAM, S3, CloudFront, etc.)
	GlobalServiceRegion = "us-east-1"
	// DefaultMaxConcurrency is the default maximum number of concurrent AWS API requests
	DefaultMaxConcurrency = 5
)

var (
	version = "v1.0.6"
	commit  = "none"
	date    = "unknown"

	ErrInvalidOutputPath = errors.New("invalid output file path")
)

// CollectionOptions holds the configuration for resource collection
type CollectionOptions struct {
	Region         string
	Profile        string
	OutputDir      string
	Categories     string
	HTML           bool
	MaxConcurrency int
}

// collectionResult holds the result of collecting resources for a category
type collectionResult struct {
	err       error
	category  string
	resources []resources.Resource
}

// CollectionError wraps per-category errors so callers can inspect details while
// keeping the top-level error message relatively static for better error handling.
type CollectionError struct {
	Details map[string]error
}

func (ce CollectionError) Error() string {
	_ = len(ce.Details)
	return "failed to collect one or more categories"
}

// collectResources collects resources from all specified collectors and regions
// collectResources runs collectors across regions and returns a map of successful
// results per category and a map of per-category errors for collectors that failed.
// The caller can decide how to handle partial failures; this function will not
// stop on first error in order to try to gather as many successful results as
// possible.
//
// Note: Collectors must be initialized with AWS clients before calling this function.
func collectResources(ctx context.Context, l *logger.Logger, collectors map[string]resources.Collector, regionsToCheck []string, opts *CollectionOptions) (map[string]collectionResult, map[string]error) {
	// Collect resources in parallel using goroutines
	// For each region and collector combination
	var wg sync.WaitGroup
	resultsChan := make(chan collectionResult, len(collectors)*len(regionsToCheck))
	// Semaphore to limit concurrent requests
	concurrency := opts.MaxConcurrency
	if concurrency <= 0 {
		concurrency = DefaultMaxConcurrency
	}
	semaphore := make(chan struct{}, concurrency)

	for name, collector := range collectors {
		for _, regionToCheck := range regionsToCheck {
			wg.Add(1)
			go func(name string, collector resources.Collector, reg string) {
				defer wg.Done()
				// Acquire semaphore
				semaphore <- struct{}{}
				defer func() { <-semaphore }() // Release semaphore

				l.Info("Collecting resources", LogKeyCategory, name, "region", reg)
				res, collectErr := collector.Collect(ctx, reg)
				resultsChan <- collectionResult{
					category:  name,
					resources: res,
					err:       collectErr,
				}
			}(name, collector, regionToCheck)
		}
	}

	// Close channel when all goroutines are done
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect all results and merge resources from multiple regions
	categoryResults := make(map[string]collectionResult)
	failed := make(map[string]error)
	for result := range resultsChan {
		if result.err != nil {
			// track failures per category so caller can act on partial failures
			l.Error("Error collecting resources", "category", result.category, "error", result.err)
			failed[result.category] = result.err
			continue
		}
		// Merge resources from multiple regions
		if existing, ok := categoryResults[result.category]; ok {
			existing.resources = append(existing.resources, result.resources...)
			categoryResults[result.category] = existing
		} else {
			categoryResults[result.category] = result
		}
	}

	return categoryResults, failed
}

// runCollection executes the main resource collection logic
func runCollection(ctx context.Context, l *logger.Logger, opts *CollectionOptions) error {
	region := opts.Region
	profile := opts.Profile
	outputDir := opts.OutputDir
	categoryStr := opts.Categories
	html := opts.HTML

	// Parse regions (allow comma-separated list). The first region is used
	// to initialize the AWS config (primary region). The full list will be
	// expanded with GlobalServiceRegion when collecting.
	userRegions := parseCommaList(region)
	if len(userRegions) == 0 {
		userRegions = []string{"ap-northeast-1"}
	}

	// Initialize AWS Config with the primary region (first in the list) and profile
	primaryRegion := userRegions[0]
	cfg, err := aws.NewConfig(ctx, primaryRegion, profile)
	if err != nil {
		return fmt.Errorf("failed to load aws config: %w", err)
	}

	// Check AWS credentials before any AWS API usage
	l.Info("Checking AWS credentials...")
	identity, err := validation.CheckAWSCredentials(ctx, &cfg)
	if err != nil {
		return fmt.Errorf("aws credentials check failed: %w", err)
	}
	l.Info("AWS identity", "identity", identity)

	// Extract account ID from identity ARN
	accountID, err := helpers.ExtractAccountID(identity)
	if err != nil {
		return fmt.Errorf("failed to extract account ID from ARN: %w", err)
	}
	l.Info("Account ID", "accountID", accountID)

	// Create output directory structure: {outputDir}/{accountID}/resources
	resourcesDir := filepath.Join(outputDir, accountID, "resources")
	if mkdirErr := os.MkdirAll(resourcesDir, DefaultDirPerm); mkdirErr != nil {
		return fmt.Errorf("failed to create output directory: %w", mkdirErr)
	}

	// Initialize regions to check (support multiple regions and always include GlobalServiceRegion)
	regionsToCheck := initializeRegions(userRegions)
	l.Info("Regions to check", "regions", regionsToCheck)

	// Initialize collectors with AWS clients for all regions
	if initErr := resources.InitializeCollectors(&cfg, regionsToCheck); initErr != nil {
		return fmt.Errorf("failed to initialize collectors: %w", initErr)
	}

	// Iterate over registered collectors
	// Filter by categories if specified
	collectors := resources.GetCollectors()
	if categoryStr != "" {
		categoryList := strings.Split(categoryStr, ",")
		filteredCollectors := make(map[string]resources.Collector)
		for _, cat := range categoryList {
			cat = strings.TrimSpace(cat)
			if collector, exists := collectors[cat]; exists {
				filteredCollectors[cat] = collector
			} else {
				l.Warn("Unknown category specified", "category", cat)
			}
		}
		collectors = filteredCollectors
	}

	// Collect resources from all collectors and regions
	categoryResults, failedCategories := collectResources(ctx, l, collectors, regionsToCheck, opts)

	// Sort categories by name for deterministic output
	var categories []string
	for category := range categoryResults {
		categories = append(categories, category)
	}
	sort.Strings(categories)

	// Sort resources within each category if sorting is enabled
	for category := range categoryResults {
		result := categoryResults[category]
		collector := collectors[category]
		if collector.ShouldSort() {
			sort.Slice(result.resources, func(i, j int) bool {
				a, b := result.resources[i], result.resources[j]
				// Sort by: Region, SubCategory1, SubCategory2, Name
				if a.Region != b.Region {
					return a.Region < b.Region
				}
				if a.SubCategory1 != b.SubCategory1 {
					return a.SubCategory1 < b.SubCategory1
				}
				if a.SubCategory2 != b.SubCategory2 {
					return a.SubCategory2 < b.SubCategory2
				}
				if a.SubCategory3 != b.SubCategory3 {
					return a.SubCategory3 < b.SubCategory3
				}
				return a.Name < b.Name
			})
			categoryResults[category] = result
		}
	}

	// Write all.csv by directly combining results (no intermediate file read needed)
	allCSVPath := filepath.Join(resourcesDir, "all.csv")
	allCSVPath = filepath.Clean(allCSVPath)
	l.Info("Writing all results to file", LogKeyFile, allCSVPath)
	allFile, createAllErr := os.Create(allCSVPath) // #nosec G304 - Path is controlled and sanitized
	if createAllErr != nil {
		return fmt.Errorf("failed to create all.csv: %w", createAllErr)
	}
	defer func() {
		if closeErr := allFile.Close(); closeErr != nil {
			l.Error("Failed to close all.csv", LogKeyError, closeErr)
		}
	}()

	// First, write per-category CSV files (one file per collector/category).
	for _, category := range categories {
		result := categoryResults[category]
		if len(result.resources) == 0 {
			continue
		}
		cols := collectors[category].GetColumns()
		categoryPath := filepath.Join(resourcesDir, category+".csv")
		catFile, createCatErr := os.Create(categoryPath) // nolint:gosec // G304 - path is controlled and sanitized
		if createCatErr != nil {
			l.Error("Failed to create category csv", LogKeyError, createCatErr, LogKeyCategory, category)
			continue
		}
		if werr := exporter.WriteCSV(catFile, result.resources, cols); werr != nil {
			l.Error("Failed to write category csv", LogKeyError, werr, LogKeyCategory, category, LogKeyFile, categoryPath)
		}
		if cerr := catFile.Close(); cerr != nil {
			l.Error("Failed to close category csv", LogKeyError, cerr, LogKeyCategory, category, LogKeyFile, categoryPath)
		}
	}

	// Write all.csv by merging all category CSV files in A-Z order.
	// Insert one blank line between each category (matching aws_get_resources.sh behavior).
	cw := csv.NewWriter(allFile)
	for idx, category := range categories {
		result := categoryResults[category]
		if len(result.resources) == 0 {
			continue
		}
		cols := collectors[category].GetColumns()

		// Write header for each category
		var headers []string
		for _, col := range cols {
			headers = append(headers, col.Header)
		}
		if writeErr := cw.Write(headers); writeErr != nil {
			return fmt.Errorf("failed to write all.csv header for category %s: %w", category, writeErr)
		}

		// Write data rows for this category
		for i := range result.resources {
			r := result.resources[i]
			var row []string
			for _, col := range cols {
				row = append(row, col.Value(r))
			}
			if writeErr := cw.Write(row); writeErr != nil {
				return fmt.Errorf("failed to write all.csv row for category %s: %w", category, writeErr)
			}
		}

		// Insert blank line between categories (except after the last category)
		if idx < len(categories)-1 {
			if writeErr := cw.Write([]string{""}); writeErr != nil {
				return fmt.Errorf("failed to write blank line in all.csv: %w", writeErr)
			}
		}
	}
	cw.Flush()
	if flushErr := cw.Error(); flushErr != nil {
		return fmt.Errorf("failed to flush all.csv: %w", flushErr)
	}

	l.Info("Collection completed successfully", "outputDir", resourcesDir)
	if html {
		l.Info("Generating HTML index...")
		if htmlErr := exporter.GenerateHTML(outputDir, accountID, "all.csv", categories); htmlErr != nil {
			return fmt.Errorf("failed to generate HTML: %w", htmlErr)
		}
		l.Info("HTML index generated successfully", "indexPath", filepath.Join(outputDir, accountID, "index.html"))
	}
	// If there were per-category failures, return an aggregated error so the
	// caller and CLI can surface partial failure state while outputs may still
	// contain successful results.
	if len(failedCategories) > 0 {
		// Build a deterministic list of failures
		var keys []string
		for k := range failedCategories {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		// details are available in the returned error (CollectionError.Details)
		return fmt.Errorf("failed to collect categories: %w", CollectionError{Details: failedCategories})
	}

	return nil
}

// main initializes and runs the CLI application for collecting AWS resources.
func main() {
	// Create CLI app with basic configuration
	app := &cli.App{
		Name:    "arc",
		Usage:   "Collect AWS resources and output to CSV",
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "Enable verbose output",
			},
			&cli.StringFlag{
				Name:    "region",
				Aliases: []string{"r"},
				Usage:   "AWS region(s) to use (comma-separated list accepted). The first region is used as the primary region for API client initialization",
				EnvVars: []string{"AWS_DEFAULT_REGION"},
				Value:   "ap-northeast-1",
			},
			&cli.StringFlag{
				Name:    "profile",
				Usage:   "AWS profile to use",
				EnvVars: []string{"AWS_PROFILE"},
			},
			&cli.StringFlag{
				Name:    "output-dir",
				Aliases: []string{"D"},
				Usage:   "Base output directory",
				Value:   "./output",
			},
			&cli.StringFlag{
				Name:    "categories",
				Aliases: []string{"c"},
				Usage:   "Comma-separated list of categories to collect (e.g. 'acm,ec2,cloudfront')",
			},
			&cli.BoolFlag{
				Name:    "html",
				Aliases: []string{"H"},
				Usage:   "Generate HTML index",
			},
			&cli.IntFlag{
				Name:    "concurrency",
				Aliases: []string{"C"},
				Usage:   "Maximum number of concurrent AWS API requests",
				Value:   DefaultMaxConcurrency,
			},
		},
		Action: func(c *cli.Context) error {
			// Set up logger based on verbose flag
			logLevel := slog.LevelInfo
			if c.Bool("verbose") {
				logLevel = slog.LevelDebug
			}
			l := logger.NewText(logLevel)

			// Extract command-line arguments
			ctx := c.Context
			region := c.String("region")
			profile := c.String("profile")
			outputDir := c.String("output-dir")
			categories := c.String("categories")
			html := c.Bool("html")
			concurrency := c.Int("concurrency")

			// Create collection options
			opts := &CollectionOptions{
				Region:         region,
				Profile:        profile,
				OutputDir:      outputDir,
				Categories:     categories,
				HTML:           html,
				MaxConcurrency: concurrency,
			}

			// Run the collection logic
			return runCollection(ctx, l, opts)
		},
	}

	// Run the CLI app and handle any errors
	if err := app.Run(os.Args); err != nil {
		// Create a default logger for fatal errors
		defaultLogger := logger.NewDefault()
		defaultLogger.Error("Application failed", "error", err)
		os.Exit(1)
	}
}

// initializeRegions returns the list of regions to check based on the
// user-provided list. It preserves order, deduplicates and ensures the
// GlobalServiceRegion is present.
// Similar to shell script's initialize_regions function but accepts multiple regions.

// parseCommaList splits a comma-separated string and trims spaces, returning
// only non-empty elements in order.
func parseCommaList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var out []string
	seen := make(map[string]struct{})
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

// initializeRegionsFromList returns the list of regions to check based on the
// user-provided list. It preserves order, deduplicates and ensures the
// GlobalServiceRegion is present.
func initializeRegions(userRegions []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, r := range userRegions {
		if r == "" {
			continue
		}
		if _, ok := seen[r]; ok {
			continue
		}
		seen[r] = struct{}{}
		out = append(out, r)
	}
	if _, ok := seen[GlobalServiceRegion]; !ok {
		out = append(out, GlobalServiceRegion)
	}
	return out
}

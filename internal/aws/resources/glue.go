// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/glue"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// GlueCollector collects Glue Databases and Jobs.
// It uses dependency injection to manage Glue clients for multiple regions.
// It retrieves configuration details such as worker types and script locations.
// It also handles different job types including Python shell and Glue ETL.
// The collector paginates through databases and jobs to ensure all resources are captured.
type GlueCollector struct {
	clients      map[string]*glue.Client
	nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewGlueCollector creates a new Glue collector with clients for the specified regions.
// This constructor follows the standard naming convention for dependency injection:
// New<ServiceName>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*<ServiceName>Collector, error)
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create Glue clients for
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *GlueCollector: Initialized collector with regional clients and name resolver
//   - error: Error if client creation fails
func NewGlueCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*GlueCollector, error) {
	clients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *glue.Client {
		return glue.NewFromConfig(*c, func(o *glue.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Glue clients: %w", err)
	}

	return &GlueCollector{
		clients:      clients,
		nameResolver: nameResolver,
	}, nil
}

// Name returns the collector name.
func (*GlueCollector) Name() string {
	return "glue"
}

// ShouldSort returns true.
func (*GlueCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV column definitions for Glue.
func (*GlueCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory1", Value: func(r Resource) string { return r.SubCategory1 }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ID", Value: func(r Resource) string { return r.ARN }}, // Using ARN field for ID (Name in bash script)
		{Header: "Description", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Description") }},
		{Header: "RoleARN", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RoleARN") }},
		{Header: "Timeout", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Timeout") }},
		{Header: "WorkerType", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "WorkerType") }},
		{Header: "NumberOfWorkers", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "NumberOfWorkers") }},
		{Header: "MaxRetries", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "MaxRetries") }},
		{Header: "GlueVersion", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "GlueVersion") }},
		{Header: "Language", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Language") }},
		{Header: "ScriptLocation", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ScriptLocation") }},
	}
}

// Collect collects Glue resources for the specified region.
// The collector must have been initialized with a client for this region.
func (c *GlueCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	svc, ok := c.clients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

	var resources []Resource

	// Databases
	dbPaginator := glue.NewGetDatabasesPaginator(svc, &glue.GetDatabasesInput{})
	for dbPaginator.HasMorePages() {
		page, err := dbPaginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get databases: %w", err)
		}

		for i := range page.DatabaseList {
			db := &page.DatabaseList[i]
			resources = append(resources, NewResource(&ResourceInput{
				Category:     "glue",
				SubCategory1: "Database",
				Name:         db.Name,
				Region:       region,
				ARN:          db.Name, // ID column
				RawData: map[string]any{
					"Description": db.Description,
				},
			}))
		}
	}

	// Jobs
	jobPaginator := glue.NewGetJobsPaginator(svc, &glue.GetJobsInput{})
	for jobPaginator.HasMorePages() {
		page, err := jobPaginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get jobs: %w", err)
		}

		for i := range page.Jobs {
			job := &page.Jobs[i]

			var language string
			if job.Command != nil {
				if job.Command.PythonVersion != nil {
					language = "Python" + *job.Command.PythonVersion
				} else if job.Command.Name != nil {
					name := *job.Command.Name
					if name == "glueetl" || name == "pythonshell" {
						language = "Python3"
					} else {
						language = name
					}
				}
			}

			var scriptLoc string
			if job.Command != nil {
				scriptLoc = aws.ToString(job.Command.ScriptLocation)
			}

			resources = append(resources, NewResource(&ResourceInput{
				Category:     "glue",
				SubCategory1: "Job",
				Name:         job.Name,
				Region:       region,
				ARN:          job.Name, // ID column
				RawData: map[string]any{
					"Description":     job.Description,
					"RoleARN":         job.Role,
					"Timeout":         job.Timeout,
					"WorkerType":      job.WorkerType,
					"NumberOfWorkers": job.NumberOfWorkers,
					"MaxRetries":      job.MaxRetries,
					"GlueVersion":     job.GlueVersion,
					"Language":        language,
					"ScriptLocation":  scriptLoc,
				},
			}))
		}
	}

	return resources, nil
}

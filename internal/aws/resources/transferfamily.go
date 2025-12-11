// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/transfer"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// TransferFamilyCollector collects Transfer Family servers.
// It uses dependency injection to manage Transfer Family clients for multiple regions.
type TransferFamilyCollector struct {
	clients      map[string]*transfer.Client
	nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewTransferFamilyCollector creates a new Transfer Family collector with clients for the specified regions.
// This constructor follows the standard naming convention for dependency injection:
// New<ServiceName>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*<ServiceName>Collector, error)
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create Transfer Family clients for
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *TransferFamilyCollector: Initialized collector with regional clients and name resolver
//   - error: Error if client creation fails
func NewTransferFamilyCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*TransferFamilyCollector, error) {
	clients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *transfer.Client {
		return transfer.NewFromConfig(*c, func(o *transfer.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Transfer Family clients: %w", err)
	}

	return &TransferFamilyCollector{
		clients:      clients,
		nameResolver: nameResolver,
	}, nil
}

// Name returns the resource name of the collector.
func (*TransferFamilyCollector) Name() string {
	return "transferfamily"
}

// ShouldSort returns whether the collected resources should be sorted.
func (*TransferFamilyCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV columns for the collector.
func (*TransferFamilyCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory1", Value: func(r Resource) string { return r.SubCategory1 }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ServerID", Value: func(r Resource) string { return r.ARN }}, // Using ARN field for ServerID
		{Header: "Protocol", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Protocol") }},
		{Header: "State", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "State") }},
	}
}

// Collect collects Transfer Family resources for the specified region.
// The collector must have been initialized with a client for this region.
func (c *TransferFamilyCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	svc, ok := c.clients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

	var resources []Resource

	paginator := transfer.NewListServersPaginator(svc, &transfer.ListServersInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list servers: %w", err)
		}

		for i := range page.Servers {
			server := &page.Servers[i]

			var protocol string
			desc, descErr := svc.DescribeServer(ctx, &transfer.DescribeServerInput{
				ServerId: server.ServerId,
			})
			if descErr == nil && desc.Server != nil && len(desc.Server.Protocols) > 0 {
				protocol = string(desc.Server.Protocols[0])
			}

			r := NewResource(&ResourceInput{
				Category:     "transferfamily",
				SubCategory1: "Server",
				Name:         server.ServerId,
				Region:       region,
				ARN:          server.ServerId, // Using ServerID as ARN
				RawData: map[string]any{
					"Protocol": protocol,
					"State":    server.State,
				},
			})
			resources = append(resources, r)
		}
	}

	return resources, nil
}

package resources

import (
	"context"
	"fmt"

	"github.com/y-miyazaki/arc/internal/aws/helpers"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/transfer"
)

// TransferFamilyCollector collects Transfer Family servers.
// It retrieves server details including protocols and state.
// The collector uses the ListServers API to discover Transfer Family servers in the region.
// It extracts protocol information and current server state for reporting.
type TransferFamilyCollector struct{}

// Name returns the collector name.
func (*TransferFamilyCollector) Name() string {
	return "transferfamily"
}

// ShouldSort returns true.
func (*TransferFamilyCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV column definitions for Transfer Family.
func (*TransferFamilyCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ServerID", Value: func(r Resource) string { return r.ARN }}, // Using ARN field for ServerID
		{Header: "Protocol", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Protocol") }},
		{Header: "State", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "State") }},
	}
}

// Collect collects Transfer Family resources from the specified region.
func (*TransferFamilyCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	svc := transfer.NewFromConfig(*cfg, func(o *transfer.Options) {
		o.Region = region
	})

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

			resources = append(resources, NewResource(&ResourceInput{
				Category:    "transferfamily",
				SubCategory: "Server",
				Name:        server.ServerId,
				Region:      region,
				ARN:         server.ServerId,
				RawData: map[string]any{
					"Protocol": protocol,
					"State":    server.State,
				},
			}))
		}
	}

	return resources, nil
}

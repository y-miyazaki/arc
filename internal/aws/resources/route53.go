package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// Route53Collector collects Route53 resources.
type Route53Collector struct{}

// Name returns the resource name of the collector.
func (*Route53Collector) Name() string {
	return "route53"
}

// ShouldSort returns whether the collected resources should be sorted.
func (*Route53Collector) ShouldSort() bool {
	return false
}

// GetColumns returns the CSV columns for the collector.
func (*Route53Collector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ID", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ID") }},
		{Header: "Type", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Type") }},
		{Header: "Comment", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Comment") }},
		{Header: "TTL", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "TTL") }},
		{Header: "RecordType", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RecordType") }},
		{Header: "Value", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Value") }},
		{Header: "RecordCount", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RecordCount") }},
	}
}

// Collect collects Route53 resources.
func (*Route53Collector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// Route53 is a global service, only process from us-east-1 to avoid duplicates.
	if region != "us-east-1" {
		return nil, nil
	}

	svc := route53.NewFromConfig(*cfg)
	var resources []Resource

	// List Hosted Zones
	paginator := route53.NewListHostedZonesPaginator(svc, &route53.ListHostedZonesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list hosted zones: %w", err)
		}

		for i := range page.HostedZones {
			zone := &page.HostedZones[i]
			// Zone ID usually comes with /hostedzone/ prefix, remove it for cleaner output.
			zoneID := strings.TrimPrefix(helpers.StringValue(zone.Id), "/hostedzone/")
			zoneName := helpers.StringValue(zone.Name)
			zoneType := "Public"
			if zone.Config != nil && zone.Config.PrivateZone {
				zoneType = "Private"
			}
			// Zone comment
			var zoneComment *string
			if zone.Config != nil {
				zoneComment = zone.Config.Comment
			}
			// Add HostedZone resource
			resources = append(resources, NewResource(&ResourceInput{
				Category:    "route53",
				SubCategory: "HostedZone",
				Name:        zoneName,
				Region:      "Global",
				RawData: map[string]any{
					"ID":          zoneID,
					"Type":        zoneType,
					"Comment":     zoneComment,
					"RecordCount": zone.ResourceRecordSetCount,
				},
			}))

			// List Resource Record Sets for the zone
			recordPaginator := route53.NewListResourceRecordSetsPaginator(svc, &route53.ListResourceRecordSetsInput{
				HostedZoneId: zone.Id,
			})
			var recordPage *route53.ListResourceRecordSetsOutput
			for recordPaginator.HasMorePages() {
				recordPage, err = recordPaginator.NextPage(ctx)
				if err != nil {
					// Log error but continue with other zones to maximize data collection
					continue
				}

				for i := range recordPage.ResourceRecordSets {
					record := &recordPage.ResourceRecordSets[i]
					ttl := record.TTL

					// Handle Alias targets vs regular values
					var values []string
					if record.AliasTarget != nil {
						values = append(values, helpers.StringValue(record.AliasTarget.DNSName))
					} else {
						for j := range record.ResourceRecords {
							rr := &record.ResourceRecords[j]
							values = append(values, helpers.StringValue(rr.Value))
						}
					}
					valueStr := strings.Join(values, "\n")

					// Add RecordSet resource
					resources = append(resources, NewResource(&ResourceInput{
						Category:       "route53",
						SubCategory:    "",
						SubSubCategory: "RecordSet",
						Name:           record.Name,
						Region:         "Global",
						RawData: map[string]any{
							"ID":         zoneID, // Use ZoneID for grouping context
							"TTL":        ttl,
							"RecordType": record.Type,
							"Value":      valueStr,
						},
					}))
				}
			}
		}
	}

	return resources, nil
}

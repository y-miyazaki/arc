// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// DynamoDBCollector collects DynamoDB resources.
// It uses dependency injection to manage DynamoDB clients for multiple regions.
type DynamoDBCollector struct {
	clients      map[string]*dynamodb.Client
	nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewDynamoDBCollector creates a new DynamoDB collector with clients for the specified regions.
// This constructor follows the standard naming convention for dependency injection:
// New<ServiceName>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*<ServiceName>Collector, error)
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create DynamoDB clients for
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *DynamoDBCollector: Initialized collector with regional clients and name resolver
//   - error: Error if client creation fails
func NewDynamoDBCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*DynamoDBCollector, error) {
	clients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *dynamodb.Client {
		return dynamodb.NewFromConfig(*c, func(o *dynamodb.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create DynamoDB clients: %w", err)
	}

	return &DynamoDBCollector{
		clients:      clients,
		nameResolver: nameResolver,
	}, nil
}

// Name returns the resource name of the collector.
func (*DynamoDBCollector) Name() string {
	return "dynamodb"
}

// ShouldSort returns whether the collected resources should be sorted.
func (*DynamoDBCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV columns for the collector.
func (*DynamoDBCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory1", Value: func(r Resource) string { return r.SubCategory1 }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "AttributeDefinitions", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AttributeDefinitions") }},
		{Header: "BillingMode", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "BillingMode") }},
		{Header: "StreamEnabled", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "StreamEnabled") }},
		{Header: "GlobalTable", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "GlobalTable") }},
		{Header: "PointInTimeRecovery", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PointInTimeRecovery") }},
		{Header: "RecoveryPeriodInDays", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RecoveryPeriodInDays") }},
		{Header: "EarliestRestorableDateTime", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "EarliestRestorableDateTime") }},
		{Header: "LatestRestorableDateTime", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LatestRestorableDateTime") }},
		{Header: "DeletionProtection", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "DeletionProtection") }},
		{Header: "TTLAttribute", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "TTLAttribute") }},
		{Header: "SSE", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "SSE") }},
		{Header: "KmsKey", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "KmsKey") }},
		{Header: "ItemCount", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ItemCount") }},
		{Header: "TableSize(Bytes)", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "TableSize") }},
		{Header: "Status", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Status") }},
		{Header: "CreationDateTime", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreationDateTime") }},
	}
}

// Collect collects DynamoDB resources for the specified region.
// The collector must have been initialized with a client for this region.
func (c *DynamoDBCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	svc, ok := c.clients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

	var resources []Resource

	// Get all KMS keys to resolve names efficiently
	kmsMap, err := c.nameResolver.GetAllKMSKeys(ctx, region)
	if err != nil {
		return nil, fmt.Errorf("failed to get KMS keys: %w", err)
	}

	// List Tables
	paginator := dynamodb.NewListTablesPaginator(svc, &dynamodb.ListTablesInput{})
	for paginator.HasMorePages() {
		page, pageErr := paginator.NextPage(ctx)
		if pageErr != nil {
			return nil, fmt.Errorf("failed to list tables: %w", pageErr)
		}

		for _, tableName := range page.TableNames {
			// Describe Table
			tableOut, tableErr := svc.DescribeTable(ctx, &dynamodb.DescribeTableInput{
				TableName: aws.String(tableName),
			})
			if tableErr != nil {
				// If table is not found (deleted during collection), skip it
				continue
			}
			table := tableOut.Table

			// Describe Continuous Backups (PITR)
			pitrOut, _ := svc.DescribeContinuousBackups(ctx, &dynamodb.DescribeContinuousBackupsInput{
				TableName: aws.String(tableName),
			})

			// Describe TTL
			ttlOut, _ := svc.DescribeTimeToLive(ctx, &dynamodb.DescribeTimeToLiveInput{
				TableName: aws.String(tableName),
			})

			// Attribute Definitions
			var attrDefs []string
			for i := range table.AttributeDefinitions {
				attr := &table.AttributeDefinitions[i]
				attrDefs = append(attrDefs, fmt.Sprintf("%s (%s)", helpers.StringValue(attr.AttributeName), attr.AttributeType))
			}

			// Billing Mode
			var billingMode *string
			if table.BillingModeSummary != nil {
				billingMode = aws.String(string(table.BillingModeSummary.BillingMode))
			}

			// SSE & KMS
			var sseStatus *string
			kmsKey := ""
			if table.SSEDescription != nil {
				if table.SSEDescription.Status != "" {
					sseStatus = aws.String(string(table.SSEDescription.Status))
				}
				if table.SSEDescription.KMSMasterKeyArn != nil {
					kmsKey = helpers.ResolveNameFromMap(table.SSEDescription.KMSMasterKeyArn, kmsMap)
				}
			}

			// PITR
			var pitrEnabled *string
			var recoveryPeriod *int32
			var earliestRestorable *time.Time
			var latestRestorable *time.Time
			if pitrOut != nil && pitrOut.ContinuousBackupsDescription != nil && pitrOut.ContinuousBackupsDescription.PointInTimeRecoveryDescription != nil {
				pitrDesc := pitrOut.ContinuousBackupsDescription.PointInTimeRecoveryDescription
				if pitrDesc.PointInTimeRecoveryStatus != "" {
					pitrEnabled = aws.String(string(pitrDesc.PointInTimeRecoveryStatus))
				}
				recoveryPeriod = pitrDesc.RecoveryPeriodInDays
				earliestRestorable = pitrDesc.EarliestRestorableDateTime
				latestRestorable = pitrDesc.LatestRestorableDateTime
			}

			// TTL
			ttlAttribute := ""
			if ttlOut != nil && ttlOut.TimeToLiveDescription != nil && ttlOut.TimeToLiveDescription.AttributeName != nil {
				ttlAttribute = *ttlOut.TimeToLiveDescription.AttributeName
			}

			// Stream Enabled
			var streamEnabled *bool
			if table.StreamSpecification != nil {
				streamEnabled = table.StreamSpecification.StreamEnabled
			}
			resources = append(resources, NewResource(&ResourceInput{
				Category:     "dynamodb",
				SubCategory1: "Table",
				Name:         tableName,
				Region:       region,
				ARN:          table.TableArn,
				RawData: map[string]any{
					"AttributeDefinitions":       attrDefs,
					"BillingMode":                billingMode,
					"StreamEnabled":              streamEnabled,
					"GlobalTable":                table.GlobalTableVersion,
					"PointInTimeRecovery":        pitrEnabled,
					"RecoveryPeriodInDays":       recoveryPeriod,
					"EarliestRestorableDateTime": earliestRestorable,
					"LatestRestorableDateTime":   latestRestorable,
					"DeletionProtection":         table.DeletionProtectionEnabled,
					"TTLAttribute":               ttlAttribute,
					"SSE":                        sseStatus,
					"KmsKey":                     kmsKey,
					"ItemCount":                  table.ItemCount,
					"TableSize":                  table.TableSizeBytes,
					"Status":                     table.TableStatus,
					"CreationDateTime":           table.CreationDateTime,
				},
			}))
		}
	}

	return resources, nil
}

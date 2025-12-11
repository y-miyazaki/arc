// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/firehose"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// KinesisCollector collects Kinesis Streams and Firehose Delivery Streams.
// It uses dependency injection to manage Kinesis and Firehose clients for multiple regions.
// It gathers detailed information about streams including shards and retention.
// It also collects Firehose Delivery Streams and their destinations.
// The collector uses the Kinesis ListStreams and Firehose ListDeliveryStreams APIs
// to discover resources.
type KinesisCollector struct {
	kinesisClients  map[string]*kinesis.Client
	firehoseClients map[string]*firehose.Client
	nameResolver    *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewKinesisCollector creates a new Kinesis collector with clients for the specified regions.
// This constructor follows the standard naming convention for dependency injection:
// New<ServiceName>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*<ServiceName>Collector, error)
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create Kinesis clients for
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *KinesisCollector: Initialized collector with regional clients and name resolver
//   - error: Error if client creation fails
func NewKinesisCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*KinesisCollector, error) {
	kinesisClients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *kinesis.Client {
		return kinesis.NewFromConfig(*c, func(o *kinesis.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Kinesis clients: %w", err)
	}

	firehoseClients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *firehose.Client {
		return firehose.NewFromConfig(*c, func(o *firehose.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Firehose clients: %w", err)
	}

	return &KinesisCollector{
		kinesisClients:  kinesisClients,
		firehoseClients: firehoseClients,
		nameResolver:    nameResolver,
	}, nil
}

// Name returns the collector name.
func (*KinesisCollector) Name() string {
	return "kinesis"
}

// ShouldSort returns true.
func (*KinesisCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV column definitions for Kinesis.
func (*KinesisCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory1", Value: func(r Resource) string { return r.SubCategory1 }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "Status", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Status") }},
		{Header: "Shards", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Shards") }},
		{Header: "DestinationId", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "DestinationId") }},
		{Header: "RetentionPeriodHours", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RetentionPeriodHours") }},
		{Header: "EncryptionType", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "EncryptionType") }},
		{Header: "CreatedDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreatedDate") }},
		{Header: "LastUpdatedDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LastUpdatedDate") }},
	}
}

// Collect collects Kinesis resources for the specified region.
// The collector must have been initialized with a client for this region.
func (c *KinesisCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	kinesisSvc, ok := c.kinesisClients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

	firehoseSvc, ok := c.firehoseClients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s (Firehose)", ErrNoClientForRegion, region)
	}

	var resources []Resource

	// Kinesis Streams
	streamPaginator := kinesis.NewListStreamsPaginator(kinesisSvc, &kinesis.ListStreamsInput{})
	for streamPaginator.HasMorePages() {
		page, err := streamPaginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list streams: %w", err)
		}

		for i := range page.StreamNames {
			streamName := page.StreamNames[i]
			desc, descErr := kinesisSvc.DescribeStream(ctx, &kinesis.DescribeStreamInput{
				StreamName: &streamName,
			})
			if descErr != nil {
				continue
			}
			s := desc.StreamDescription

			resources = append(resources, NewResource(&ResourceInput{
				Category:     "kinesis",
				SubCategory1: "Stream",
				Name:         s.StreamName,
				Region:       region,
				ARN:          s.StreamARN,
				RawData: map[string]any{
					"Status":               s.StreamStatus,
					"Shards":               len(s.Shards),
					"RetentionPeriodHours": s.RetentionPeriodHours,
					"EncryptionType":       s.EncryptionType,
					"CreatedDate":          s.StreamCreationTimestamp,
				},
			}))
		}
	}

	// Firehose Delivery Streams
	var lastStreamName *string
	for {
		out, err := firehoseSvc.ListDeliveryStreams(ctx, &firehose.ListDeliveryStreamsInput{
			ExclusiveStartDeliveryStreamName: lastStreamName,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list delivery streams: %w", err)
		}

		for _, name := range out.DeliveryStreamNames {
			desc, descErr := firehoseSvc.DescribeDeliveryStream(ctx, &firehose.DescribeDeliveryStreamInput{
				DeliveryStreamName: aws.String(name),
			})
			if descErr != nil {
				continue
			}
			ds := desc.DeliveryStreamDescription

			var destID string
			if len(ds.Destinations) > 0 {
				destID = aws.ToString(ds.Destinations[0].DestinationId)
			}

			resources = append(resources, NewResource(&ResourceInput{
				Category:     "kinesis",
				SubCategory1: "Firehose",
				Name:         ds.DeliveryStreamName,
				Region:       region,
				ARN:          ds.DeliveryStreamARN,
				RawData: map[string]any{
					"Status":          ds.DeliveryStreamStatus,
					"DestinationId":   destID,
					"CreatedDate":     ds.CreateTimestamp,
					"LastUpdatedDate": ds.LastUpdateTimestamp,
				},
			}))
		}

		if out.HasMoreDeliveryStreams != nil && *out.HasMoreDeliveryStreams {
			if len(out.DeliveryStreamNames) > 0 {
				lastStreamName = aws.String(out.DeliveryStreamNames[len(out.DeliveryStreamNames)-1])
			} else {
				break
			}
		} else {
			break
		}
	}

	return resources, nil
}

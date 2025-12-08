package resources

import (
	"context"
	"fmt"

	"github.com/y-miyazaki/arc/internal/aws/helpers"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/firehose"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
)

// KinesisCollector collects Kinesis Streams and Firehose Delivery Streams.
// It gathers detailed information about streams including shards and retention.
// It also collects Firehose Delivery Streams and their destinations.
// The collector uses the Kinesis ListStreams and Firehose ListDeliveryStreams APIs
// to discover resources.
type KinesisCollector struct{}

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
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
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

// Collect collects Kinesis resources from the specified region.
func (*KinesisCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	kinesisSvc := kinesis.NewFromConfig(*cfg, func(o *kinesis.Options) {
		o.Region = region
	})
	firehoseSvc := firehose.NewFromConfig(*cfg, func(o *firehose.Options) {
		o.Region = region
	})

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
				Category:    "kinesis",
				SubCategory: "Stream",
				Name:        s.StreamName,
				Region:      region,
				ARN:         s.StreamARN,
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
				Category:    "kinesis",
				SubCategory: "Firehose",
				Name:        ds.DeliveryStreamName,
				Region:      region,
				ARN:         ds.DeliveryStreamARN,
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

// Package resources provides AWS resource collectors for different services.
package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// S3Collector collects S3 buckets.
type S3Collector struct{}

// Name returns the resource name of the collector.
func (*S3Collector) Name() string {
	return "s3"
}

// ShouldSort returns whether the collected resources should be sorted.
func (*S3Collector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV columns for the collector.
func (*S3Collector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "Encryption", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Encryption") }},
		{Header: "Versioning", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Versioning") }},
		{Header: "PABBlockPublicACLs", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PABBlockPublicACLs") }},
		{Header: "PABIgnorePublicACLs", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PABIgnorePublicACLs") }},
		{Header: "PABBlockPublicPolicy", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PABBlockPublicPolicy") }},
		{Header: "PABRestrictPublicBuckets", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PABRestrictPublicBuckets") }},
		{Header: "AccessLogARN", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AccessLogARN") }},
		{Header: "LifecycleRules", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LifecycleRules") }},
		{Header: "CreationDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreationDate") }},
	}
}

// Collect collects S3 resources.
// S3 is a global service - buckets exist in all regions but API calls must be made to us-east-1.
// Return empty if called with a different region to avoid duplicates.
func (*S3Collector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// S3 bucket listing only works from us-east-1 (global service).
	if region != "us-east-1" {
		return nil, nil
	}

	// Client for listing buckets (us-east-1).
	globalSvc := s3.NewFromConfig(*cfg, func(o *s3.Options) {
		o.Region = "us-east-1"
	})

	var resources []Resource

	// List all buckets.
	listBucketsOut, err := globalSvc.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	// Cache for region-specific clients.
	regionClients := make(map[string]*s3.Client)
	regionClients["us-east-1"] = globalSvc

	// Helper to get or create client for a region.
	getClient := func(r string) *s3.Client {
		if client, ok := regionClients[r]; ok {
			return client
		}
		client := s3.NewFromConfig(*cfg, func(o *s3.Options) {
			o.Region = r
		})
		regionClients[r] = client
		return client
	}

	for i := range listBucketsOut.Buckets {
		bucket := &listBucketsOut.Buckets[i]

		// Get bucket location using global client.
		bucketRegion := "us-east-1" // default
		locationOut, locErr := globalSvc.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
			Bucket: bucket.Name,
		})
		if locErr == nil && locationOut.LocationConstraint != "" {
			bucketRegion = string(locationOut.LocationConstraint)
			// Handle special case for EU (Ireland).
			if bucketRegion == "EU" {
				bucketRegion = "eu-west-1"
			}
		}

		// Use region-specific client for bucket operations.
		svc := getClient(bucketRegion)

		// Get encryption configuration.
		encryption := "None"
		encryptionOut, encErr := svc.GetBucketEncryption(ctx, &s3.GetBucketEncryptionInput{
			Bucket: bucket.Name,
		})
		if encErr == nil && encryptionOut.ServerSideEncryptionConfiguration != nil &&
			len(encryptionOut.ServerSideEncryptionConfiguration.Rules) > 0 {
			rule := encryptionOut.ServerSideEncryptionConfiguration.Rules[0]
			if rule.ApplyServerSideEncryptionByDefault != nil {
				encryption = string(rule.ApplyServerSideEncryptionByDefault.SSEAlgorithm)
			}
		}

		// Get versioning status.
		versioning := "Disabled"
		versioningOut, verErr := svc.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
			Bucket: bucket.Name,
		})
		if verErr == nil && versioningOut.Status != "" {
			versioning = string(versioningOut.Status)
		}

		// Public Access Block
		var pabBlockPublicACLs *bool
		var pabIgnorePublicACLs *bool
		var pabBlockPublicPolicy *bool
		var pabRestrictPublicBuckets *bool
		pabOut, pabErr := svc.GetPublicAccessBlock(ctx, &s3.GetPublicAccessBlockInput{
			Bucket: bucket.Name,
		})
		if pabErr == nil && pabOut.PublicAccessBlockConfiguration != nil {
			pab := pabOut.PublicAccessBlockConfiguration
			pabBlockPublicACLs = pab.BlockPublicAcls
			pabIgnorePublicACLs = pab.IgnorePublicAcls
			pabBlockPublicPolicy = pab.BlockPublicPolicy
			pabRestrictPublicBuckets = pab.RestrictPublicBuckets
		}

		// Get access logging configuration.
		accessLogARN := ""
		loggingOut, logErr := svc.GetBucketLogging(ctx, &s3.GetBucketLoggingInput{
			Bucket: bucket.Name,
		})
		if logErr == nil && loggingOut.LoggingEnabled != nil && loggingOut.LoggingEnabled.TargetBucket != nil {
			targetBucket := helpers.StringValue(loggingOut.LoggingEnabled.TargetBucket)
			if targetBucket != "" {
				accessLogARN = fmt.Sprintf("arn:aws:s3:::%s", targetBucket)
			}
		}

		// Get lifecycle configuration.
		lifecycleRules := ""
		lifecycleOut, lifecycleErr := svc.GetBucketLifecycleConfiguration(ctx, &s3.GetBucketLifecycleConfigurationInput{
			Bucket: bucket.Name,
		})
		if lifecycleErr == nil && lifecycleOut.Rules != nil && len(lifecycleOut.Rules) > 0 {
			// Convert each rule to formatted JSON string to match shell script behavior.
			// Shell script uses jq '.[]' which outputs each rule as separate JSON object.
			var ruleStrings []string
			for i := range lifecycleOut.Rules {
				ruleJSON, jsonErr := helpers.FormatJSONIndent(lifecycleOut.Rules[i])
				if jsonErr == nil {
					ruleStrings = append(ruleStrings, ruleJSON)
				}
			}
			if len(ruleStrings) > 0 {
				lifecycleRules = strings.Join(ruleStrings, "\n")
			}
		}

		resources = append(resources, NewResource(&ResourceInput{
			Category:    "s3",
			SubCategory: "Bucket",
			Name:        bucket.Name,
			Region:      bucketRegion,
			ARN:         fmt.Sprintf("arn:aws:s3:::%s", helpers.StringValue(bucket.Name)),
			RawData: map[string]any{
				"Encryption":               encryption,
				"Versioning":               versioning,
				"PABBlockPublicACLs":       pabBlockPublicACLs,
				"PABIgnorePublicACLs":      pabIgnorePublicACLs,
				"PABBlockPublicPolicy":     pabBlockPublicPolicy,
				"PABRestrictPublicBuckets": pabRestrictPublicBuckets,
				"AccessLogARN":             accessLogARN,
				"LifecycleRules":           lifecycleRules,
				"CreationDate":             bucket.CreationDate,
			},
		}))
	}

	return resources, nil
}

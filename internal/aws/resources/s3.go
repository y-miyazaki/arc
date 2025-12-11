// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

const (
	// statusDisabled represents a disabled status for S3 features.
	statusDisabled = "Disabled"
	// statusEnabled represents an enabled status for S3 features.
	statusEnabled = "Enabled"
)

// S3Collector collects S3 buckets.
// It uses dependency injection to manage S3 clients.
// S3 is a global service - only processes from us-east-1 to avoid duplicates.
type S3Collector struct {
	client       *s3.Client
	nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewS3Collector creates a new S3 collector with a global client.
// This constructor follows the standard naming convention for dependency injection:
// New<ServiceName>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*<ServiceName>Collector, error)
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions (only us-east-1 will be used for global service)
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *S3Collector: Initialized collector with global client and name resolver
//   - error: Error if client creation fails
func NewS3Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*S3Collector, error) {
	// S3 is a global service, create a single client for us-east-1
	_ = regions // unused parameter
	client := s3.NewFromConfig(*cfg, func(o *s3.Options) {
		o.Region = "us-east-1"
	})

	return &S3Collector{
		client:       client,
		nameResolver: nameResolver,
	}, nil
}

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
		{Header: "SubCategory1", Value: func(r Resource) string { return r.SubCategory1 }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "Versioning", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Versioning") }},
		{Header: "BucketABAC", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "BucketABAC") }},
		{Header: "Encryption", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Encryption") }},
		{Header: "KMSKey", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "KMSKey") }},
		{Header: "AccessLogARN", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AccessLogARN") }},
		{Header: "TransferAcceleration", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "TransferAcceleration") }},
		{Header: "ObjectLock", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ObjectLock") }},
		{Header: "RequesterPays", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RequesterPays") }},
		{Header: "StaticWebsiteHosting", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "StaticWebsiteHosting") }},
		{Header: "PABBlockPublicACLs", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PABBlockPublicACLs") }},
		{Header: "PABIgnorePublicACLs", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PABIgnorePublicACLs") }},
		{Header: "PABBlockPublicPolicy", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PABBlockPublicPolicy") }},
		{Header: "PABRestrictPublicBuckets", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PABRestrictPublicBuckets") }},
		{Header: "ACL", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ACL") }},
		{Header: "LifecycleRules", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LifecycleRules") }},
		{Header: "CreationDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreationDate") }},
	}
}

// Collect collects S3 resources for the specified region.
// S3 is a global service - only processes from us-east-1 to avoid duplicates.
func (c *S3Collector) Collect(ctx context.Context, region string) ([]Resource, error) {
	// S3 bucket listing only works from us-east-1 (global service).
	if region != "us-east-1" {
		return nil, nil
	}

	var resources []Resource

	// List all buckets.
	listBucketsOut, err := c.client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	// Cache for region-specific clients.
	regionClients := make(map[string]*s3.Client)
	regionClients["us-east-1"] = c.client

	// Helper to get or create client for a region.
	getClient := func(r string) *s3.Client {
		if client, ok := regionClients[r]; ok {
			return client
		}
		// Create new client with same credentials but different region
		cfg := c.client.Options().Copy()
		cfg.Region = r
		client := s3.New(cfg)
		regionClients[r] = client
		return client
	}

	for i := range listBucketsOut.Buckets {
		bucket := &listBucketsOut.Buckets[i]

		// Get bucket location using global client.
		bucketRegion := "us-east-1" // default
		locationOut, locErr := c.client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
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
		kmsKey := ""
		encryptionOut, encErr := svc.GetBucketEncryption(ctx, &s3.GetBucketEncryptionInput{
			Bucket: bucket.Name,
		})
		if encErr == nil && encryptionOut.ServerSideEncryptionConfiguration != nil &&
			len(encryptionOut.ServerSideEncryptionConfiguration.Rules) > 0 {
			rule := encryptionOut.ServerSideEncryptionConfiguration.Rules[0]
			if rule.ApplyServerSideEncryptionByDefault != nil {
				encryption = string(rule.ApplyServerSideEncryptionByDefault.SSEAlgorithm)
				if rule.ApplyServerSideEncryptionByDefault.KMSMasterKeyID != nil {
					kmsKey = helpers.StringValue(rule.ApplyServerSideEncryptionByDefault.KMSMasterKeyID)
				}
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

		// Get bucket tagging for ABAC.
		var bucketABAC []string
		taggingOut, taggingErr := svc.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
			Bucket: bucket.Name,
		})
		if taggingErr == nil && taggingOut.TagSet != nil && len(taggingOut.TagSet) > 0 {
			for _, tag := range taggingOut.TagSet {
				bucketABAC = append(bucketABAC, fmt.Sprintf("%s=%s", helpers.StringValue(tag.Key), helpers.StringValue(tag.Value)))
			}
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

		// Get bucket ACL.
		var acl []string
		aclOut, aclErr := svc.GetBucketAcl(ctx, &s3.GetBucketAclInput{
			Bucket: bucket.Name,
		})
		if aclErr == nil && aclOut.Grants != nil && len(aclOut.Grants) > 0 {
			for i := range aclOut.Grants {
				grant := &aclOut.Grants[i]
				granteeType := ""
				granteeID := ""
				if grant.Grantee != nil {
					granteeType = string(grant.Grantee.Type)
					if grant.Grantee.ID != nil {
						granteeID = helpers.StringValue(grant.Grantee.ID)
					} else if grant.Grantee.URI != nil {
						granteeID = helpers.StringValue(grant.Grantee.URI)
					}
				}
				permission := string(grant.Permission)
				acl = append(acl, fmt.Sprintf("%s:%s=%s", granteeType, granteeID, permission))
			}
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

		// Get transfer acceleration configuration.
		transferAcceleration := statusDisabled
		accelOut, accelErr := svc.GetBucketAccelerateConfiguration(ctx, &s3.GetBucketAccelerateConfigurationInput{
			Bucket: bucket.Name,
		})
		if accelErr == nil && accelOut.Status != "" {
			transferAcceleration = string(accelOut.Status)
		}

		// Get object lock configuration.
		objectLock := statusDisabled
		objectLockOut, objectLockErr := svc.GetObjectLockConfiguration(ctx, &s3.GetObjectLockConfigurationInput{
			Bucket: bucket.Name,
		})
		if objectLockErr == nil && objectLockOut.ObjectLockConfiguration != nil && objectLockOut.ObjectLockConfiguration.ObjectLockEnabled != "" {
			objectLock = string(objectLockOut.ObjectLockConfiguration.ObjectLockEnabled)
		}

		// Get requester pays configuration.
		requesterPays := statusDisabled
		reqPayOut, reqPayErr := svc.GetBucketRequestPayment(ctx, &s3.GetBucketRequestPaymentInput{
			Bucket: bucket.Name,
		})
		if reqPayErr == nil && reqPayOut.Payer != "" {
			requesterPays = string(reqPayOut.Payer)
		}

		// Get static website hosting configuration.
		staticWebsiteHosting := statusDisabled
		websiteOut, websiteErr := svc.GetBucketWebsite(ctx, &s3.GetBucketWebsiteInput{
			Bucket: bucket.Name,
		})
		if websiteErr == nil && websiteOut.IndexDocument != nil {
			staticWebsiteHosting = statusEnabled
		}

		// Get lifecycle configuration.
		var ruleStrings []string
		lifecycleOut, lifecycleErr := svc.GetBucketLifecycleConfiguration(ctx, &s3.GetBucketLifecycleConfigurationInput{
			Bucket: bucket.Name,
		})
		if lifecycleErr == nil && lifecycleOut.Rules != nil && len(lifecycleOut.Rules) > 0 {
			// Convert each rule to formatted JSON string to match shell script behavior.
			// Shell script uses jq '.[]' which outputs each rule as separate JSON object.
			for i := range lifecycleOut.Rules {
				ruleJSON, jsonErr := helpers.FormatJSONIndent(lifecycleOut.Rules[i])
				if jsonErr == nil {
					ruleStrings = append(ruleStrings, ruleJSON)
				}
			}
		}

		r := NewResource(&ResourceInput{
			Category:     "s3",
			SubCategory1: "Bucket",
			Name:         bucket.Name,
			Region:       bucketRegion,
			ARN:          fmt.Sprintf("arn:aws:s3:::%s", helpers.StringValue(bucket.Name)),
			RawData: map[string]any{
				"Versioning":               versioning,
				"BucketABAC":               bucketABAC,
				"Encryption":               encryption,
				"KMSKey":                   kmsKey,
				"AccessLogARN":             accessLogARN,
				"TransferAcceleration":     transferAcceleration,
				"ObjectLock":               objectLock,
				"RequesterPays":            requesterPays,
				"StaticWebsiteHosting":     staticWebsiteHosting,
				"PABBlockPublicACLs":       pabBlockPublicACLs,
				"PABIgnorePublicACLs":      pabIgnorePublicACLs,
				"PABBlockPublicPolicy":     pabBlockPublicPolicy,
				"PABRestrictPublicBuckets": pabRestrictPublicBuckets,
				"ACL":                      acl,
				"LifecycleRules":           ruleStrings,
				"CreationDate":             bucket.CreationDate,
			},
		})
		resources = append(resources, r)
	}

	return resources, nil
}

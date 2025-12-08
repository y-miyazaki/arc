// Package resources provides AWS resource collectors.
package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// Resource represents a single collected AWS resource
type Resource struct {
	RawData        map[string]any
	Category       string
	SubCategory    string
	SubSubCategory string
	Name           string
	Region         string
	ARN            string
}

// ResourceInput is the input for creating a new Resource.
// Fields are of type 'any' to allow passing pointers directly.
type ResourceInput struct {
	Category       any
	SubCategory    any
	SubSubCategory any
	Name           any
	Region         any
	ARN            any
	RawData        map[string]any
}

// NewResource creates a new Resource and normalizes its RawData.
// It automatically converts all input fields to strings using helpers.StringValue.
func NewResource(input *ResourceInput) Resource {
	return Resource{
		Category:       helpers.StringValue(input.Category),
		SubCategory:    helpers.StringValue(input.SubCategory, ""),
		SubSubCategory: helpers.StringValue(input.SubSubCategory, ""),
		Name:           helpers.StringValue(input.Name),
		Region:         helpers.StringValue(input.Region),
		ARN:            helpers.StringValue(input.ARN),
		RawData:        helpers.NormalizeRawData(input.RawData),
	}
}

// Column defines a CSV column with a header and a value extractor
type Column struct {
	Value  func(Resource) string
	Header string
}

// Collector is the interface that all resource collectors must implement
// nolint:unused
type Collector interface {
	// Name returns the resource name of the collector.
	Name() string
	// Collect collects resources
	Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error)
	// GetColumns returns the CSV columns for the collector.
	GetColumns() []Column
	// ShouldSort returns whether the collected resources should be sorted.
	ShouldSort() bool
}

var collectors = make(map[string]Collector)

// Register registers a collector to the global registry.
// This function is called during package initialization (init).
func Register(name string, c Collector) {
	collectors[name] = c
}

// GetCollectors returns all registered collectors.
// The returned map is safe for concurrent read access.
// All collectors are stateless and safe for concurrent execution across multiple goroutines.
func GetCollectors() map[string]Collector {
	return collectors
}

// init registers all collectors during package initialization.
// This pattern is acceptable because:
// 1. All collectors are stateless (empty structs)
// 2. Registration happens once at startup
// 3. The registry is read-only after initialization
func init() { // nolint:gochecknoinits
	Register("acm", &ACMCollector{})
	Register("apigateway", &APIGatewayCollector{})
	Register("batch", &BatchCollector{})
	Register("cloudformation", &CloudFormationCollector{})
	Register("cloudfront", &CloudFrontCollector{})
	Register("cloudwatch_alarms", &CloudWatchAlarmsCollector{})
	Register("cloudwatch_logs", &CloudWatchLogsCollector{})
	Register("cognito", &CognitoCollector{})
	Register("dynamodb", &DynamoDBCollector{})
	Register("ec2", &EC2Collector{})
	Register("ecr", &ECRCollector{})
	Register("ecs", &ECSCollector{})
	Register("efs", &EFSCollector{})
	Register("elasticache", &ElastiCacheCollector{})
	Register("elb", &ELBCollector{})
	Register("eventbridge", &EventBridgeCollector{})
	Register("glue", &GlueCollector{})
	Register("iam_role", &IAMRoleCollector{})
	Register("iam_policy", &IAMPolicyCollector{})
	Register("iam_user_group", &IAMUserGroupCollector{})
	Register("kinesis", &KinesisCollector{})
	Register("kms", &KMSCollector{})
	Register("lambda", &LambdaCollector{})
	Register("quicksight", &QuickSightCollector{})
	Register("rds", &RDSCollector{})
	Register("redshift", &RedshiftCollector{})
	Register("route53", &Route53Collector{})
	Register("s3", &S3Collector{})
	Register("secretsmanager", &SecretsManagerCollector{})
	Register("sns", &SNSCollector{})
	Register("sqs", &SQSCollector{})
	Register("transferfamily", &TransferFamilyCollector{})
	Register("vpc", &VPCCollector{})
	Register("waf", &WAFCollector{})
}

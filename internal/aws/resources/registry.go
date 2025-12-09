// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// Sentinel errors for registry operations (alphabetical order).
var (
	ErrInvalidCollectorType = errors.New("constructor returned invalid collector type")
	ErrInvalidErrorType     = errors.New("constructor returned invalid error type")
	ErrNoClientForRegion    = errors.New("no client found for region")
	ErrUnknownCollector     = errors.New("unknown collector")
)

var (
	collectors            = make(map[string]Collector)
	collectorConstructors = make(map[string]any)
)

// Collector is the interface that all resource collectors must implement.
// Collectors are initialized with AWS clients for specific regions via dependency injection,
// and the Collect method no longer requires the aws.Config parameter.
// nolint:unused
type Collector interface {
	// Collect collects resources for the specified region.
	// The collector must have been initialized with clients for this region.
	Collect(ctx context.Context, region string) ([]Resource, error)
	// GetColumns returns the CSV columns for the collector.
	GetColumns() []Column
	// Name returns the resource name of the collector.
	Name() string
	// ShouldSort returns whether the collected resources should be sorted.
	ShouldSort() bool
}

// Column defines a CSV column with a header and a value extractor
type Column struct {
	Value  func(Resource) string
	Header string
}

// Resource represents a single collected AWS resource
type Resource struct {
	ARN            string
	Category       string
	Name           string
	RawData        map[string]any
	Region         string
	SubCategory    string
	SubSubCategory string
}

// ResourceInput is the input for creating a new Resource.
// Fields are of type 'any' to allow passing pointers directly.
type ResourceInput struct {
	ARN            any
	Category       any
	Name           any
	RawData        map[string]any
	Region         any
	SubCategory    any
	SubSubCategory any
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
		ARN:            helpers.StringValue(input.ARN, ""),
		RawData:        helpers.NormalizeRawData(input.RawData),
	}
}

// GetCollectors returns all registered collectors.
// The returned map is safe for concurrent read access.
// Collectors must be initialized via InitializeCollectors before use.
func GetCollectors() map[string]Collector {
	return collectors
}

// InitializeCollectors initializes all supported collectors with AWS clients for the specified regions.
// This function uses reflection to dynamically call constructor functions following the naming convention:
// New<CollectorName>Collector(cfg *aws.Config, regions []string) (*<CollectorName>Collector, error)
//
// The function will fail fast if any collector initialization fails, returning a detailed error message.
//
// Parameters:
//   - cfg: AWS configuration with credentials and base settings
//   - regions: List of AWS regions to create clients for
//
// Returns:
//   - error: Error if any collector initialization fails
//
// Example:
//
//	err := resources.InitializeCollectors(&cfg, []string{"us-east-1", "eu-west-1"})
//	if err != nil {
//	    log.Fatal(err)
//	}
func InitializeCollectors(cfg *aws.Config, regions []string) error {
	// Create a single shared NameResolver instance for all collectors
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	if err != nil {
		return fmt.Errorf("failed to create NameResolver: %w", err)
	}

	// Register all collector constructors
	// Add new collectors here as they are migrated to the DI pattern
	RegisterConstructor("acm", NewACMCollector)
	RegisterConstructor("apigateway", NewAPIGatewayCollector)
	RegisterConstructor("batch", NewBatchCollector)
	RegisterConstructor("cloudformation", NewCloudFormationCollector)
	RegisterConstructor("cloudfront", NewCloudFrontCollector)
	RegisterConstructor("cloudwatch_alarms", NewCloudWatchAlarmsCollector)
	RegisterConstructor("cloudwatch_logs", NewCloudWatchLogsCollector)
	RegisterConstructor("cognito", NewCognitoCollector)
	RegisterConstructor("dynamodb", NewDynamoDBCollector)
	RegisterConstructor("ec2", NewEC2Collector)
	RegisterConstructor("ecr", NewECRCollector)
	RegisterConstructor("ecs", NewECSCollector)
	RegisterConstructor("efs", NewEFSCollector)
	RegisterConstructor("elasticache", NewElastiCacheCollector)
	RegisterConstructor("elb", NewELBCollector)
	RegisterConstructor("eventbridge", NewEventBridgeCollector)
	RegisterConstructor("glue", NewGlueCollector)
	RegisterConstructor("iam_policy", NewIAMPolicyCollector)
	RegisterConstructor("iam_role", NewIAMRoleCollector)
	RegisterConstructor("iam_user_group", NewIAMUserGroupCollector)
	RegisterConstructor("kinesis", NewKinesisCollector)
	RegisterConstructor("kms", NewKMSCollector)
	RegisterConstructor("lambda", NewLambdaCollector)
	RegisterConstructor("quicksight", NewQuickSightCollector)
	RegisterConstructor("rds", NewRDSCollector)
	RegisterConstructor("redshift", NewRedshiftCollector)
	RegisterConstructor("route53", NewRoute53Collector)
	RegisterConstructor("s3", NewS3Collector)
	RegisterConstructor("secretsmanager", NewSecretsManagerCollector)
	RegisterConstructor("ses", NewSESCollector)
	RegisterConstructor("sns", NewSNSCollector)
	RegisterConstructor("sqs", NewSQSCollector)
	RegisterConstructor("transferfamily", NewTransferFamilyCollector)
	RegisterConstructor("vpc", NewVPCCollector)
	RegisterConstructor("waf", NewWAFCollector)

	for name := range collectorConstructors {
		collector, collErr := createCollector(name, cfg, regions, nameResolver)
		if collErr != nil {
			return fmt.Errorf("failed to initialize %s collector: %w", name, collErr)
		}
		Register(name, collector)
	}

	return nil
}

// Register registers a collector to the global registry.
// This function is called during collector initialization.
func Register(name string, c Collector) {
	collectors[name] = c
}

// RegisterConstructor registers a collector constructor function.
// Constructor must follow the signature: func(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*XxxCollector, error)
func RegisterConstructor(name string, constructor any) {
	collectorConstructors[name] = constructor
}

// createCollector creates a collector instance using reflection to call its constructor.
// The constructor must be registered via RegisterConstructor before calling this function.
// Constructor signature: func(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*XxxCollector, error)
//
// Parameters:
//   - name: Collector name (e.g., "acm", "apigateway")
//   - cfg: AWS configuration
//   - regions: List of regions to create clients for
//   - nameResolver: Shared NameResolver instance for all collectors
//
// Returns:
//   - Collector: The initialized collector instance
//   - error: Error if constructor not found or initialization fails
func createCollector(name string, cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (Collector, error) {
	constructor, exists := collectorConstructors[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrUnknownCollector, name)
	}

	// Call constructor with reflection
	// Constructor signature: func(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*XxxCollector, error)
	result := reflect.ValueOf(constructor).Call([]reflect.Value{
		reflect.ValueOf(cfg),
		reflect.ValueOf(regions),
		reflect.ValueOf(nameResolver),
	})

	// Check for errors (second return value)
	if !result[1].IsNil() {
		err, ok := result[1].Interface().(error)
		if !ok {
			return nil, ErrInvalidErrorType
		}
		return nil, err
	}

	// Return the collector (first return value)
	collector, ok := result[0].Interface().(Collector)
	if !ok {
		return nil, ErrInvalidCollectorType
	}
	return collector, nil
}

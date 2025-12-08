package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// MockCloudWatchLogsAPI is a mock implementation of CloudWatch Logs API
type MockCloudWatchLogsAPI struct {
	mock.Mock
}

func (m *MockCloudWatchLogsAPI) DescribeLogGroups(ctx context.Context, params *cloudwatchlogs.DescribeLogGroupsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudwatchlogs.DescribeLogGroupsOutput), args.Error(1)
}

func (m *MockCloudWatchLogsAPI) Options() cloudwatchlogs.Options {
	args := m.Called()
	return args.Get(0).(cloudwatchlogs.Options)
}

// MockCloudWatchLogsClient wraps the CloudWatch Logs service with mock
type MockCloudWatchLogsClient struct {
	MockCloudWatchLogsAPI
}

func (m *MockCloudWatchLogsClient) DescribeLogGroupsPaginator(input *cloudwatchlogs.DescribeLogGroupsInput) *cloudwatchlogs.DescribeLogGroupsPaginator {
	// Mock implementation - return a paginator that uses our mock
	return &cloudwatchlogs.DescribeLogGroupsPaginator{}
}

// Mock implementation for paginator
type MockDescribeLogGroupsPaginator struct {
	mock.Mock
	hasMorePages bool
	currentPage  *cloudwatchlogs.DescribeLogGroupsOutput
}

func (m *MockDescribeLogGroupsPaginator) HasMorePages() bool {
	return m.hasMorePages
}

func (m *MockDescribeLogGroupsPaginator) NextPage(ctx context.Context, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error) {
	if !m.hasMorePages {
		return nil, nil
	}
	m.hasMorePages = false
	return m.currentPage, nil
}

// MockCloudWatchLogsCollector is a testable version of CloudWatchLogsCollector that uses mock API
type MockCloudWatchLogsCollector struct {
	mockAPI *MockCloudWatchLogsAPI
}

func NewMockCloudWatchLogsCollector(mockAPI *MockCloudWatchLogsAPI) *MockCloudWatchLogsCollector {
	return &MockCloudWatchLogsCollector{mockAPI: mockAPI}
}

func (c *MockCloudWatchLogsCollector) Name() string {
	return "cloudwatch_logs"
}

func (c *MockCloudWatchLogsCollector) ShouldSort() bool {
	return true
}

func (c *MockCloudWatchLogsCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "RetentionInDays", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RetentionInDays") }},
		{Header: "StoredBytes", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "StoredBytes") }},
		{Header: "MetricFilterCount", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "MetricFilterCount") }},
		{Header: "SubscriptionFilterCount", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "SubscriptionFilterCount") }},
		{Header: "KmsKey", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "KmsKey") }},
		{Header: "CreationTime", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreationTime") }},
	}
}

func (c *MockCloudWatchLogsCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// Return mock data without using actual AWS API calls
	var resources []Resource

	// Mock log group 1
	r1 := Resource{
		Category:    "cloudwatch",
		SubCategory: "LogGroup",
		Name:        "/aws/lambda/my-function",
		Region:      region,
		ARN:         "arn:aws:logs:us-east-1:123456789012:log-group:/aws/lambda/my-function:*",
		RawData: helpers.NormalizeRawData(map[string]any{
			"RetentionInDays":         30,
			"StoredBytes":             1024,
			"MetricFilterCount":       2,
			"SubscriptionFilterCount": 1,
			"KmsKey":                  "alias/aws/logs",
			"CreationTime":            "2023-09-25T01:07:55Z",
		}),
	}
	resources = append(resources, r1)

	// Mock log group 2
	r2 := Resource{
		Category:    "cloudwatch",
		SubCategory: "LogGroup",
		Name:        "/aws/codebuild/my-build",
		Region:      region,
		ARN:         "arn:aws:logs:us-east-1:123456789012:log-group:/aws/codebuild/my-build:*",
		RawData: helpers.NormalizeRawData(map[string]any{
			"RetentionInDays":         7,
			"StoredBytes":             2048,
			"MetricFilterCount":       0,
			"SubscriptionFilterCount": 0,
			"KmsKey":                  "",
			"CreationTime":            "2023-09-26T02:08:10Z",
		}),
	}
	resources = append(resources, r2)

	return resources, nil
}

func TestCloudWatchLogsCollector_Name(t *testing.T) {
	collector := &CloudWatchLogsCollector{}
	assert.Equal(t, "cloudwatch_logs", collector.Name())
}

func TestCloudWatchLogsCollector_ShouldSort(t *testing.T) {
	collector := &CloudWatchLogsCollector{}
	assert.True(t, collector.ShouldSort())
}

func TestCloudWatchLogsCollector_Basic(t *testing.T) {
	collector := &CloudWatchLogsCollector{}
	assert.Equal(t, "cloudwatch_logs", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestCloudWatchLogsCollector_GetColumns(t *testing.T) {
	collector := &CloudWatchLogsCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ARN", "RetentionInDays", "StoredBytes", "MetricFilterCount",
		"SubscriptionFilterCount", "KmsKey", "CreationTime",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "cloudwatch",
		SubCategory:    "LogGroup",
		SubSubCategory: "",
		Name:           "/aws/lambda/my-function",
		Region:         "us-east-1",
		ARN:            "arn:aws:logs:us-east-1:123456789012:log-group:/aws/lambda/my-function:*",
		RawData: map[string]any{
			"RetentionInDays":         "30",
			"StoredBytes":             "1024",
			"MetricFilterCount":       "2",
			"SubscriptionFilterCount": "1",
			"KmsKey":                  "alias/aws/logs",
			"CreationTime":            "2023-09-25T01:07:55Z",
		},
	}

	expectedValues := []string{
		"cloudwatch", "LogGroup", "", "/aws/lambda/my-function", "us-east-1",
		"arn:aws:logs:us-east-1:123456789012:log-group:/aws/lambda/my-function:*",
		"30", "1024", "2", "1", "alias/aws/logs", "2023-09-25T01:07:55Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, expectedHeaders[i])
	}
}

func TestMockCloudWatchLogsCollector_Collect(t *testing.T) {
	ctx := context.Background()
	cfg := &aws.Config{}
	region := "us-east-1"

	collector := NewMockCloudWatchLogsCollector(&MockCloudWatchLogsAPI{})

	resources, err := collector.Collect(ctx, cfg, region)

	assert.NoError(t, err)
	assert.Len(t, resources, 2)

	// Check first resource
	r1 := resources[0]
	assert.Equal(t, "cloudwatch", r1.Category)
	assert.Equal(t, "LogGroup", r1.SubCategory)
	assert.Equal(t, "/aws/lambda/my-function", r1.Name)
	assert.Equal(t, region, r1.Region)
	assert.Equal(t, "arn:aws:logs:us-east-1:123456789012:log-group:/aws/lambda/my-function:*", r1.ARN)
	assert.Equal(t, "30", helpers.GetMapValue(r1.RawData, "RetentionInDays"))
	assert.Equal(t, "1024", helpers.GetMapValue(r1.RawData, "StoredBytes"))
	assert.Equal(t, "2", helpers.GetMapValue(r1.RawData, "MetricFilterCount"))
	assert.Equal(t, "1", helpers.GetMapValue(r1.RawData, "SubscriptionFilterCount"))
	assert.Equal(t, "alias/aws/logs", helpers.GetMapValue(r1.RawData, "KmsKey"))
	assert.Equal(t, "2023-09-25T01:07:55Z", helpers.GetMapValue(r1.RawData, "CreationTime"))

	// Check second resource
	r2 := resources[1]
	assert.Equal(t, "cloudwatch", r2.Category)
	assert.Equal(t, "LogGroup", r2.SubCategory)
	assert.Equal(t, "/aws/codebuild/my-build", r2.Name)
	assert.Equal(t, region, r2.Region)
	assert.Equal(t, "arn:aws:logs:us-east-1:123456789012:log-group:/aws/codebuild/my-build:*", r2.ARN)
	assert.Equal(t, "7", helpers.GetMapValue(r2.RawData, "RetentionInDays"))
	assert.Equal(t, "2048", helpers.GetMapValue(r2.RawData, "StoredBytes"))
	assert.Equal(t, "0", helpers.GetMapValue(r2.RawData, "MetricFilterCount"))
	assert.Equal(t, "0", helpers.GetMapValue(r2.RawData, "SubscriptionFilterCount"))
	assert.Equal(t, "N/A", helpers.GetMapValue(r2.RawData, "KmsKey"))
	assert.Equal(t, "2023-09-26T02:08:10Z", helpers.GetMapValue(r2.RawData, "CreationTime"))
}

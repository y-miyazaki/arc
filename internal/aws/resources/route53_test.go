package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// MockRoute53Collector is a testable version of Route53Collector that uses mock data
type MockRoute53Collector struct{}

func NewMockRoute53Collector() *MockRoute53Collector {
	return &MockRoute53Collector{}
}

func (c *MockRoute53Collector) Name() string {
	return "route53"
}

func (c *MockRoute53Collector) ShouldSort() bool {
	return false
}

func (c *MockRoute53Collector) GetColumns() []Column {
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

func (c *MockRoute53Collector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// Route53 is a global service, only process from us-east-1 to avoid duplicates
	if region != "us-east-1" {
		return nil, nil
	}

	// Return mock data without using actual AWS API calls
	var resources []Resource

	// Mock Hosted Zone
	r1 := Resource{
		Category:    "route53",
		SubCategory: "HostedZone",
		Name:        "example.com.",
		Region:      "Global",
		RawData: helpers.NormalizeRawData(map[string]any{
			"ID":          "Z1234567890ABCDEF",
			"Type":        "Public",
			"Comment":     "Example domain",
			"RecordCount": 5,
		}),
	}
	resources = append(resources, r1)

	// Mock Record Set
	r2 := Resource{
		Category:       "route53",
		SubCategory:    "",
		SubSubCategory: "RecordSet",
		Name:           "www.example.com.",
		Region:         "Global",
		RawData: helpers.NormalizeRawData(map[string]any{
			"ID":         "Z1234567890ABCDEF",
			"TTL":        300,
			"RecordType": "A",
			"Value":      "192.168.1.1",
		}),
	}
	resources = append(resources, r2)

	return resources, nil
}

func TestRoute53Collector_Basic(t *testing.T) {
	collector := &Route53Collector{}
	assert.Equal(t, "route53", collector.Name())
	assert.False(t, collector.ShouldSort())
}

func TestRoute53Collector_GetColumns(t *testing.T) {
	collector := &Route53Collector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ID", "Type", "Comment", "TTL", "RecordType", "Value", "RecordCount",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i := range expectedHeaders {
		assert.Equal(t, expectedHeaders[i], columns[i].Header)
	}

	// Test Value functions with a sample resource (Hosted Zone)
	sampleResource := Resource{
		Category:       "Networking",
		SubCategory:    "Route53",
		SubSubCategory: "HostedZone",
		Name:           "example.com.",
		Region:         "Global",
		RawData: map[string]any{
			"ID":          "Z1234567890ABCDEF",
			"Type":        "Public",
			"Comment":     "Example domain",
			"TTL":         "300",
			"RecordType":  "A",
			"Value":       "192.168.1.1",
			"RecordCount": "5",
		},
	}

	expectedValues := []string{
		"Networking", "Route53", "HostedZone", "example.com.", "Global",
		"Z1234567890ABCDEF", "Public", "Example domain", "300", "A", "192.168.1.1", "5",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, expectedHeaders[i])
	}
}

func TestMockRoute53Collector_Collect(t *testing.T) {
	ctx := context.Background()
	cfg := &aws.Config{}
	region := "us-east-1"

	collector := NewMockRoute53Collector()

	resources, err := collector.Collect(ctx, cfg, region)

	assert.NoError(t, err)
	assert.Len(t, resources, 2)

	// Check first resource (Hosted Zone)
	r1 := resources[0]
	assert.Equal(t, "route53", r1.Category)
	assert.Equal(t, "HostedZone", r1.SubCategory)
	assert.Equal(t, "example.com.", r1.Name)
	assert.Equal(t, "Global", r1.Region)
	assert.Equal(t, "Z1234567890ABCDEF", helpers.GetMapValue(r1.RawData, "ID"))
	assert.Equal(t, "Public", helpers.GetMapValue(r1.RawData, "Type"))
	assert.Equal(t, "Example domain", helpers.GetMapValue(r1.RawData, "Comment"))
	assert.Equal(t, "5", helpers.GetMapValue(r1.RawData, "RecordCount"))

	// Check second resource (Record Set)
	r2 := resources[1]
	assert.Equal(t, "route53", r2.Category)
	assert.Equal(t, "", r2.SubCategory)
	assert.Equal(t, "RecordSet", r2.SubSubCategory)
	assert.Equal(t, "www.example.com.", r2.Name)
	assert.Equal(t, "Global", r2.Region)
	assert.Equal(t, "Z1234567890ABCDEF", helpers.GetMapValue(r2.RawData, "ID"))
	assert.Equal(t, "300", helpers.GetMapValue(r2.RawData, "TTL"))
	assert.Equal(t, "A", helpers.GetMapValue(r2.RawData, "RecordType"))
	assert.Equal(t, "192.168.1.1", helpers.GetMapValue(r2.RawData, "Value"))
}

func TestMockRoute53Collector_Collect_NonUSEast1(t *testing.T) {
	ctx := context.Background()
	cfg := &aws.Config{}
	region := "eu-west-1"

	collector := NewMockRoute53Collector()

	resources, err := collector.Collect(ctx, cfg, region)

	assert.NoError(t, err)
	assert.Len(t, resources, 0)
}

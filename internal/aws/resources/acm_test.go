package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestACMCollector_Basic(t *testing.T) {
	collector := &ACMCollector{}
	assert.Equal(t, "acm", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestACMCollector_GetColumns(t *testing.T) {
	collector := &ACMCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region", "ARN",
		"Type", "KeyAlgorithm", "InUse", "Status", "CreatedDate", "IssuedDate", "ExpirationDate",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "Security",
		SubCategory:    "ACM",
		SubSubCategory: "Certificate",
		Name:           "example.com",
		Region:         "us-east-1",
		ARN:            "arn:aws:acm:us-east-1:123456789012:certificate/test-cert",
		RawData: map[string]interface{}{
			"Status":         "ISSUED",
			"Type":           "AMAZON_ISSUED",
			"KeyAlgorithm":   "RSA_2048",
			"InUse":          "test-alb",
			"RequestDate":    "2023-09-25T01:07:55Z",
			"IssuedDate":     "2023-09-25T01:07:55Z",
			"ExpirationDate": "2024-09-25T01:07:55Z",
			"CreatedDate":    "2023-09-25T01:07:55Z",
		},
	}

	expectedValues := []string{
		"Security", "ACM", "Certificate", "example.com", "us-east-1", "arn:aws:acm:us-east-1:123456789012:certificate/test-cert",
		"AMAZON_ISSUED", "RSA_2048", "test-alb", "ISSUED", "2023-09-25T01:07:55Z", "2023-09-25T01:07:55Z", "2024-09-25T01:07:55Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}

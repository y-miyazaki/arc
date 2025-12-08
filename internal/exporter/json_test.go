package exporter

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/y-miyazaki/arc/internal/aws/resources"
)

func TestWriteJSON(t *testing.T) {
	// Create test resources
	testResources := []resources.Resource{
		{
			Category:    "test",
			SubCategory: "instance",
			Name:        "test-instance-1",
			Region:      "us-east-1",
			ARN:         "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0",
			RawData: map[string]any{
				"InstanceType": "t2.micro",
				"State":        "running",
			},
		},
		{
			Category:    "test",
			SubCategory: "bucket",
			Name:        "test-bucket-1",
			Region:      "us-east-1",
			ARN:         "arn:aws:s3:::test-bucket-1",
			RawData: map[string]any{
				"Versioning": "Enabled",
				"Size":       "1024",
			},
		},
	}

	// Define columns
	columns := []resources.Column{
		{Header: "Category", Value: func(r resources.Resource) string { return r.Category }},
		{Header: "Name", Value: func(r resources.Resource) string { return r.Name }},
		{Header: "Region", Value: func(r resources.Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r resources.Resource) string { return r.ARN }},
	}

	t.Run("valid resources", func(t *testing.T) {
		var buf bytes.Buffer
		err := WriteJSON(&buf, testResources, columns)
		if err != nil {
			t.Fatalf("WriteJSON() error = %v", err)
		}

		// Parse the JSON output
		var result []map[string]string
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v", err)
		}

		// Verify the structure
		if len(result) != 2 {
			t.Errorf("Expected 2 resources, got %d", len(result))
		}

		// Check first resource
		if result[0]["Category"] != "test" {
			t.Errorf("Expected Category 'test', got %s", result[0]["Category"])
		}
		if result[0]["Name"] != "test-instance-1" {
			t.Errorf("Expected Name 'test-instance-1', got %s", result[0]["Name"])
		}
		if result[0]["Region"] != "us-east-1" {
			t.Errorf("Expected Region 'us-east-1', got %s", result[0]["Region"])
		}

		// Check second resource
		if result[1]["Name"] != "test-bucket-1" {
			t.Errorf("Expected Name 'test-bucket-1', got %s", result[1]["Name"])
		}
	})

	t.Run("empty resources", func(t *testing.T) {
		var buf bytes.Buffer
		err := WriteJSON(&buf, []resources.Resource{}, columns)
		if err != nil {
			t.Fatalf("WriteJSON() error = %v", err)
		}

		// Should produce valid JSON array
		var result []map[string]string
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v", err)
		}

		if len(result) != 0 {
			t.Errorf("Expected empty array, got %d items", len(result))
		}
	})

	t.Run("empty columns", func(t *testing.T) {
		var buf bytes.Buffer
		err := WriteJSON(&buf, testResources, []resources.Column{})
		if err != nil {
			t.Fatalf("WriteJSON() error = %v", err)
		}

		// Should produce valid JSON with empty objects
		var result []map[string]string
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v", err)
		}

		if len(result) != 2 {
			t.Errorf("Expected 2 resources, got %d", len(result))
		}

		// Objects should be empty
		if len(result[0]) != 0 {
			t.Errorf("Expected empty object, got %d fields", len(result[0]))
		}
	})
}

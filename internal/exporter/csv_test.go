package exporter_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/resources"
	"github.com/y-miyazaki/arc/internal/exporter"
)

func TestWriteCSV(t *testing.T) {
	tests := []struct {
		name     string
		res      []resources.Resource
		columns  []resources.Column
		expected string
		wantErr  bool
	}{
		{
			name: "single resource",
			res: []resources.Resource{
				{
					Category: "TestCategory",
					Name:     "TestName",
					RawData: map[string]interface{}{
						"Extra": "Value",
					},
				},
			},
			columns: []resources.Column{
				{Header: "Category", Value: func(r resources.Resource) string { return r.Category }},
				{Header: "Name", Value: func(r resources.Resource) string { return r.Name }},
				{Header: "Extra", Value: func(r resources.Resource) string { return r.RawData["Extra"].(string) }},
			},
			expected: "Category,Name,Extra\nTestCategory,TestName,Value\n",
			wantErr:  false,
		},
		{
			name: "empty resource list",
			res:  []resources.Resource{},
			columns: []resources.Column{
				{Header: "Category", Value: func(r resources.Resource) string { return r.Category }},
			},
			expected: "Category\n",
			wantErr:  false,
		},
		{
			name: "multiple resources with all fields",
			res: []resources.Resource{
				{
					Category:     "EC2",
					SubCategory1: "Instance",
					SubCategory2: "t2.micro",
					Name:         "web-server-01",
					Region:       "us-east-1",
					ARN:          "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0",
					RawData: map[string]interface{}{
						"State": "running",
						"Type":  "t2.micro",
					},
				},
				{
					Category:     "S3",
					SubCategory1: "Bucket",
					SubCategory2: "",
					Name:         "my-bucket",
					Region:       "us-east-1",
					ARN:          "arn:aws:s3:::my-bucket",
					RawData: map[string]interface{}{
						"Versioning": "Enabled",
					},
				},
			},
			columns: []resources.Column{
				{Header: "Category", Value: func(r resources.Resource) string { return r.Category }},
				{Header: "SubCategory1", Value: func(r resources.Resource) string { return r.SubCategory1 }},
				{Header: "Name", Value: func(r resources.Resource) string { return r.Name }},
				{Header: "Region", Value: func(r resources.Resource) string { return r.Region }},
				{Header: "ARN", Value: func(r resources.Resource) string { return r.ARN }},
			},
			expected: "Category,SubCategory1,Name,Region,ARN\nEC2,Instance,web-server-01,us-east-1,arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0\nS3,Bucket,my-bucket,us-east-1,arn:aws:s3:::my-bucket\n",
			wantErr:  false,
		},
		{
			name: "resource with empty fields",
			res: []resources.Resource{
				{
					Category: "Test",
					Name:     "",
					Region:   "",
					ARN:      "",
					RawData:  nil,
				},
			},
			columns: []resources.Column{
				{Header: "Category", Value: func(r resources.Resource) string { return r.Category }},
				{Header: "Name", Value: func(r resources.Resource) string { return r.Name }},
				{Header: "Region", Value: func(r resources.Resource) string { return r.Region }},
			},
			expected: "Category,Name,Region\nTest,,\n",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := exporter.WriteCSV(&buf, tt.res, tt.columns)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, buf.String())
			}
		})
	}
}

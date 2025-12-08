package helpers_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/stretchr/testify/mock"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestGetTagValue(t *testing.T) {
	tests := []struct {
		name     string
		tags     []ec2types.Tag
		key      string
		expected string
	}{
		{
			name: "tag exists with exact case match",
			tags: []ec2types.Tag{
				{Key: aws.String("Name"), Value: aws.String("test-instance")},
				{Key: aws.String("Environment"), Value: aws.String("prod")},
			},
			key:      "Name",
			expected: "test-instance",
		},
		{
			name: "tag exists with different case",
			tags: []ec2types.Tag{
				{Key: aws.String("NAME"), Value: aws.String("test-instance")},
			},
			key:      "name",
			expected: "test-instance",
		},
		{
			name: "tag does not exist",
			tags: []ec2types.Tag{
				{Key: aws.String("Environment"), Value: aws.String("prod")},
			},
			key:      "Name",
			expected: "",
		},
		{
			name:     "empty tags slice",
			tags:     []ec2types.Tag{},
			key:      "Name",
			expected: "",
		},
		{
			name: "nil tag key",
			tags: []ec2types.Tag{
				{Key: nil, Value: aws.String("value")},
			},
			key:      "Name",
			expected: "",
		},
		{
			name: "nil tag value",
			tags: []ec2types.Tag{
				{Key: aws.String("Name"), Value: nil},
			},
			key:      "Name",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.GetTagValue(tt.tags, tt.key)
			if result != tt.expected {
				t.Errorf("GetTagValue() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseARN(t *testing.T) {
	tests := []struct {
		name        string
		arnStr      string
		expected    *helpers.ARN
		expectError bool
	}{
		{
			name:   "valid S3 bucket ARN",
			arnStr: "arn:aws:s3:::my-bucket",
			expected: &helpers.ARN{
				Partition:    "aws",
				Service:      "s3",
				Region:       "",
				AccountID:    "",
				ResourceType: "",
				Resource:     "my-bucket",
			},
			expectError: false,
		},
		{
			name:   "valid EC2 instance ARN",
			arnStr: "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0",
			expected: &helpers.ARN{
				Partition:    "aws",
				Service:      "ec2",
				Region:       "us-east-1",
				AccountID:    "123456789012",
				ResourceType: "instance",
				Resource:     "i-1234567890abcdef0",
			},
			expectError: false,
		},
		{
			name:   "valid IAM role ARN",
			arnStr: "arn:aws:iam::123456789012:role/MyRole",
			expected: &helpers.ARN{
				Partition:    "aws",
				Service:      "iam",
				Region:       "",
				AccountID:    "123456789012",
				ResourceType: "role",
				Resource:     "MyRole",
			},
			expectError: false,
		},
		{
			name:        "invalid ARN - not starting with arn:",
			arnStr:      "invalid-arn",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "invalid ARN - too few parts",
			arnStr:      "arn:aws:s3",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "empty string",
			arnStr:      "",
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := helpers.ParseARN(tt.arnStr)
			if tt.expectError {
				if err == nil {
					t.Errorf("ParseARN() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("ParseARN() unexpected error: %v", err)
				}
				if result == nil || tt.expected == nil {
					if result != tt.expected {
						t.Errorf("ParseARN() = %v, want %v", result, tt.expected)
					}
				} else if *result != *tt.expected {
					t.Errorf("ParseARN() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestGetResourceNameFromARN(t *testing.T) {
	tests := []struct {
		name     string
		arnStr   string
		expected string
	}{
		{
			name:     "S3 bucket ARN",
			arnStr:   "arn:aws:s3:::my-bucket",
			expected: "my-bucket",
		},
		{
			name:     "EC2 instance ARN",
			arnStr:   "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0",
			expected: "i-1234567890abcdef0",
		},
		{
			name:     "IAM role ARN",
			arnStr:   "arn:aws:iam::123456789012:role/MyRole",
			expected: "MyRole",
		},
		{
			name:     "invalid ARN",
			arnStr:   "invalid-arn",
			expected: "",
		},
		{
			name:     "empty string",
			arnStr:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.GetResourceNameFromARN(tt.arnStr)
			if result != tt.expected {
				t.Errorf("GetResourceNameFromARN() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestResolveNameFromMap(t *testing.T) {
	tests := []struct {
		name     string
		id       *string
		nameMap  map[string]string
		expected string
	}{
		{
			name:     "nil id",
			id:       nil,
			nameMap:  map[string]string{"test": "Test Name"},
			expected: "N/A",
		},
		{
			name:     "empty id",
			id:       aws.String(""),
			nameMap:  map[string]string{"test": "Test Name"},
			expected: "N/A",
		},
		{
			name:     "id found in map",
			id:       aws.String("test"),
			nameMap:  map[string]string{"test": "Test Name"},
			expected: "Test Name",
		},
		{
			name:     "id not found in map",
			id:       aws.String("unknown"),
			nameMap:  map[string]string{"test": "Test Name"},
			expected: "unknown",
		},
		{
			name:     "empty nameMap",
			id:       aws.String("test"),
			nameMap:  map[string]string{},
			expected: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.ResolveNameFromMap(tt.id, tt.nameMap)
			if result != tt.expected {
				t.Errorf("ResolveNameFromMap() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestResolveNamesFromMap(t *testing.T) {
	nameMap := map[string]string{
		"sg-12345": "web-sg",
		"sg-67890": "db-sg",
		"sg-11111": "app-sg",
	}

	tests := []struct {
		name     string
		ids      []*string
		nameMap  map[string]string
		expected []string
	}{
		{
			name:     "multiple IDs found in map",
			ids:      []*string{aws.String("sg-12345"), aws.String("sg-67890")},
			nameMap:  nameMap,
			expected: []string{"web-sg", "db-sg"},
		},
		{
			name:     "ID not found in map",
			ids:      []*string{aws.String("sg-99999")},
			nameMap:  nameMap,
			expected: []string{"sg-99999"},
		},
		{
			name:     "mixed found and not found IDs",
			ids:      []*string{aws.String("sg-12345"), aws.String("sg-99999"), aws.String("sg-67890")},
			nameMap:  nameMap,
			expected: []string{"web-sg", "sg-99999", "db-sg"},
		},
		{
			name:     "empty IDs slice",
			ids:      []*string{},
			nameMap:  nameMap,
			expected: []string{},
		},
		{
			name:     "nil ID in slice",
			ids:      []*string{nil, aws.String("sg-12345")},
			nameMap:  nameMap,
			expected: []string{"N/A", "web-sg"},
		},
		{
			name:     "empty map",
			ids:      []*string{aws.String("sg-12345")},
			nameMap:  map[string]string{},
			expected: []string{"sg-12345"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.ResolveNamesFromMap(tt.ids, tt.nameMap)
			if len(result) != len(tt.expected) {
				t.Errorf("ResolveNamesFromMap() returned %d items, want %d", len(result), len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("ResolveNamesFromMap()[%d] = %v, want %v", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

// MockKMSClient is a mock implementation of the KMS client
type MockKMSClient struct {
	mock.Mock
}

func (m *MockKMSClient) ListAliases(ctx context.Context, params *kms.ListAliasesInput, optFns ...func(*kms.Options)) (*kms.ListAliasesOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*kms.ListAliasesOutput), args.Error(1)
}

// MockEC2Client is a mock implementation of the EC2 client
type MockEC2Client struct {
	mock.Mock
}

func (m *MockEC2Client) DescribeSecurityGroups(ctx context.Context, params *ec2.DescribeSecurityGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*ec2.DescribeSecurityGroupsOutput), args.Error(1)
}

func (m *MockEC2Client) DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*ec2.DescribeSubnetsOutput), args.Error(1)
}

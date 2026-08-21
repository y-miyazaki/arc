package helpers

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kmstypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)
func TestStructToKeyValue(t *testing.T) {
	t.Parallel()

	type testStruct struct {
		Name       string
		Count      int
		Enabled    bool
		PtrStr     *string
		PtrInt     *int32
		PtrBool    *bool
		unexported string // should be ignored
	}

	name := "test-name"
	count := int32(42)
	enabled := true
	var nilStruct *testStruct

	tests := []struct {
		name string
		in   any
		want []string
	}{
		{
			name: "populated values",
			in: testStruct{
				Name:       "test-name",
				Count:      100,
				Enabled:    true,
				PtrStr:     &name,
				PtrInt:     &count,
				PtrBool:    &enabled,
				unexported: "ignored",
			},
			want: []string{"Name=test-name", "Count=100", "Enabled=true", "PtrStr=test-name", "PtrInt=42", "PtrBool=true"},
		},
		{
			name: "nil pointers and empty values",
			in: testStruct{
				Name:    "",
				Count:   0,
				Enabled: false,
			},
			want: []string{"Count=0"},
		},
		{name: "nil struct pointer", in: nilStruct, want: nil},
		{name: "non-struct", in: "not a struct", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, StructToKeyValue(tt.in))
		})
	}
}

func TestParseARN(t *testing.T) {
	tests := []struct {
		name        string
		arnStr      string
		expected    *ARN
		expectError bool
	}{
		{
			name:   "valid ARN with resource type",
			arnStr: "arn:aws:s3:::my-bucket",
			expected: &ARN{
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
			name:   "valid ARN with resource type and name",
			arnStr: "arn:aws:iam::123456789012:user/john",
			expected: &ARN{
				Partition:    "aws",
				Service:      "iam",
				Region:       "",
				AccountID:    "123456789012",
				ResourceType: "user",
				Resource:     "john",
			},
			expectError: false,
		},
		{
			name:   "valid ARN with colon separator",
			arnStr: "arn:aws:rds:us-east-1:123456789012:db:mysql-instance",
			expected: &ARN{
				Partition:    "aws",
				Service:      "rds",
				Region:       "us-east-1",
				AccountID:    "123456789012",
				ResourceType: "db",
				Resource:     "mysql-instance",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseARN(tt.arnStr)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
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
			name:     "valid ARN",
			arnStr:   "arn:aws:s3:::my-bucket",
			expected: "my-bucket",
		},
		{
			name:     "valid ARN with resource type",
			arnStr:   "arn:aws:iam::123456789012:user/john",
			expected: "john",
		},
		{
			name:     "invalid ARN",
			arnStr:   "invalid-arn",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetResourceNameFromARN(tt.arnStr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetTagValue(t *testing.T) {
	t.Parallel()

	tags := []ec2types.Tag{
		{Key: aws.String("Name"), Value: aws.String("my-instance")},
		{Key: aws.String("Environment"), Value: aws.String("prod")},
		{Key: aws.String("name"), Value: aws.String("lowercase-name")},
	}

	tests := []struct {
		name string
		tags []ec2types.Tag
		key  string
		want string
	}{
		{name: "exact match", tags: tags, key: "Name", want: "my-instance"},
		{name: "case insensitive match", tags: tags, key: "NAME", want: "my-instance"},
		{name: "lowercase key matches first occurrence", tags: tags, key: "name", want: "my-instance"},
		{name: "non-existent key", tags: tags, key: "Missing", want: ""},
		{name: "empty tags", tags: []ec2types.Tag{}, key: "Name", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, GetTagValue(tt.tags, tt.key))
		})
	}
}

func TestResolveNameFromMap(t *testing.T) {
	nameMap := map[string]string{
		"i-123": "web-server",
		"i-456": "db-server",
	}

	tests := []struct {
		name     string
		id       *string
		expected string
	}{
		{
			name:     "found in map",
			id:       aws.String("i-123"),
			expected: "web-server",
		},
		{
			name:     "not found in map",
			id:       aws.String("i-999"),
			expected: "i-999",
		},
		{
			name:     "nil id",
			id:       nil,
			expected: "N/A",
		},
		{
			name:     "empty id",
			id:       aws.String(""),
			expected: "N/A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveNameFromMap(tt.id, nameMap)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveNamesFromMap(t *testing.T) {
	nameMap := map[string]string{
		"i-123": "web-server",
		"i-456": "db-server",
	}

	tests := []struct {
		name     string
		ids      []*string
		expected []string
	}{
		{
			name:     "multiple ids",
			ids:      []*string{aws.String("i-123"), aws.String("i-456")},
			expected: []string{"web-server", "db-server"},
		},
		{
			name:     "mixed found and not found",
			ids:      []*string{aws.String("i-123"), aws.String("i-999")},
			expected: []string{"web-server", "i-999"},
		},
		{
			name:     "empty slice",
			ids:      []*string{},
			expected: []string{},
		},
		{
			name:     "nil slice",
			ids:      nil,
			expected: []string{},
		},
		{
			name:     "with nil elements",
			ids:      []*string{aws.String("i-123"), nil, aws.String("i-456")},
			expected: []string{"web-server", "N/A", "db-server"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveNamesFromMap(tt.ids, nameMap)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewNameResolver(t *testing.T) {
	t.Parallel()

	cfg := &aws.Config{
		Region: "us-east-1",
	}

	tests := []struct {
		name    string
		regions []string
		wantLen int
	}{
		{name: "creates clients for each region", regions: []string{"us-east-1", "eu-west-1"}, wantLen: 2},
		{name: "empty regions", regions: []string{}, wantLen: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver, err := NewNameResolver(cfg, tt.regions)
			require.NoError(t, err)
			require.NotNil(t, resolver)
			assert.Len(t, resolver.ec2Clients, tt.wantLen)
			assert.Len(t, resolver.kmsClients, tt.wantLen)
			assert.Len(t, resolver.cloudfrontClients, tt.wantLen)
			for _, region := range tt.regions {
				assert.Contains(t, resolver.ec2Clients, region)
				assert.Contains(t, resolver.kmsClients, region)
				assert.Contains(t, resolver.cloudfrontClients, region)
			}
			assert.NotNil(t, resolver.cache)
			assert.NotNil(t, resolver.cloudfrontCache)
		})
	}
}

func TestGetAllKMSKeysWithClient(t *testing.T) {
	mockClient := &ManualKMSClient{}

	// Mock data
	mockClient.keys = []*kms.ListKeysOutput{
		{
			Keys: []kmstypes.KeyListEntry{
				{KeyId: aws.String("key-1"), KeyArn: aws.String("arn:aws:kms:us-east-1:123456789012:key/key-1")},
				{KeyId: aws.String("key-2"), KeyArn: aws.String("arn:aws:kms:us-east-1:123456789012:key/key-2")},
			},
		},
	}
	mockClient.aliases = []*kms.ListAliasesOutput{
		{
			Aliases: []kmstypes.AliasListEntry{
				{AliasName: aws.String("alias/test-key-1"), TargetKeyId: aws.String("key-1")},
				{AliasName: aws.String("alias/test-key-2"), TargetKeyId: aws.String("key-2")},
			},
		},
	}

	ctx := context.Background()
	result, err := getAllKMSKeysWithClient(ctx, mockClient)

	require.NoError(t, err)
	assert.Contains(t, result, "key-1")
	assert.Equal(t, "alias/test-key-1", result["key-1"])
	assert.Contains(t, result, "arn:aws:kms:us-east-1:123456789012:key/key-1")
	assert.Equal(t, "alias/test-key-1", result["arn:aws:kms:us-east-1:123456789012:key/key-1"])
}

func TestNameResolver_GetAllKMSKeys_CacheHit(t *testing.T) {
	region := "ap-northeast-1"
	cached := map[string]string{
		"key-1": "alias/test-key-1",
	}

	resolver := &NameResolver{
		kmsClients: map[string]*kms.Client{},
		cache: map[string]map[string]map[string]string{
			region: {
				"kms": cached,
			},
		},
	}

	result, err := resolver.GetAllKMSKeys(context.Background(), region)

	require.NoError(t, err)
	assert.Equal(t, cached, result)
}

func TestNameResolver_GetAllKMSKeys_NoClientForRegion(t *testing.T) {
	resolver := &NameResolver{
		kmsClients: map[string]*kms.Client{},
		cache:      make(map[string]map[string]map[string]string),
	}

	_, err := resolver.GetAllKMSKeys(context.Background(), "ap-northeast-1")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoKMSClientForRegion)
	assert.ErrorContains(t, err, "ap-northeast-1")
}

func TestNameResolver_GetAllKMSKeys_GetAllKMSKeysWithClientError(t *testing.T) {
	cfg := aws.Config{
		Region:      "us-east-1",
		Credentials: aws.AnonymousCredentials{},
	}
	resolver := &NameResolver{
		kmsClients: map[string]*kms.Client{
			"us-east-1": kms.NewFromConfig(cfg),
		},
		cache: make(map[string]map[string]map[string]string),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := resolver.GetAllKMSKeys(ctx, "us-east-1")

	require.Error(t, err)
	assert.ErrorContains(t, err, "getAllKMSKeysWithClient")
}

func TestNameResolver_GetAllEC2Resources_ClientError(t *testing.T) {
	t.Parallel()

	cfg := &aws.Config{
		Region: "us-east-1",
	}
	resolver, err := NewNameResolver(cfg, []string{"us-east-1"})
	require.NoError(t, err)
	ctx := context.Background()

	tests := []struct {
		name string
		call func(*NameResolver) error
	}{
		{name: "images", call: func(r *NameResolver) error {
			_, err := r.GetAllImages(ctx, "us-east-1")
			return err
		}},
		{name: "network interfaces", call: func(r *NameResolver) error {
			_, err := r.GetAllNetworkInterfaces(ctx, "us-east-1")
			return err
		}},
		{name: "security groups", call: func(r *NameResolver) error {
			_, err := r.GetAllSecurityGroups(ctx, "us-east-1")
			return err
		}},
		{name: "snapshots", call: func(r *NameResolver) error {
			_, err := r.GetAllSnapshots(ctx, "us-east-1")
			return err
		}},
		{name: "subnets", call: func(r *NameResolver) error {
			_, err := r.GetAllSubnets(ctx, "us-east-1")
			return err
		}},
		{name: "volumes", call: func(r *NameResolver) error {
			_, err := r.GetAllVolumes(ctx, "us-east-1")
			return err
		}},
		{name: "vpcs", call: func(r *NameResolver) error {
			_, err := r.GetAllVPCs(ctx, "us-east-1")
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Shared NameResolver cache; omit t.Parallel() (TBL-06).
			assert.Error(t, tt.call(resolver))
		})
	}
}

func TestNameResolver_GetCloudFrontPolicyNames_WithoutUsableClient(t *testing.T) {
	t.Parallel()

	cfg := &aws.Config{
		Region: "us-east-1",
	}
	resolver, err := NewNameResolver(cfg, []string{"us-east-1"})
	require.NoError(t, err)
	ctx := context.Background()

	tests := []struct {
		name string
		call func(*NameResolver) string
	}{
		{name: "origin access control", call: func(r *NameResolver) string {
			return r.GetOriginAccessControlName(ctx, "test-oac-id")
		}},
		{name: "cache policy", call: func(r *NameResolver) string {
			return r.GetCachePolicyName(ctx, "test-policy-id")
		}},
		{name: "origin request policy", call: func(r *NameResolver) string {
			return r.GetOriginRequestPolicyName(ctx, "test-policy-id")
		}},
		{name: "response headers policy", call: func(r *NameResolver) string {
			return r.GetResponseHeadersPolicyName(ctx, "test-policy-id")
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Shared NameResolver cache; omit t.Parallel() (TBL-06).
			assert.Empty(t, tt.call(resolver))
		})
	}
}

func TestNameResolver_GetAllEC2Resources_CacheHit(t *testing.T) {
	region := "ap-northeast-1"
	cached := map[string]string{"id-1": "name-1"}

	tests := []struct {
		name     string
		cacheKey string
		call     func(*NameResolver) (map[string]string, error)
	}{
		{name: "images", cacheKey: "images", call: func(r *NameResolver) (map[string]string, error) { return r.GetAllImages(context.Background(), region) }},
		{name: "enis", cacheKey: "enis", call: func(r *NameResolver) (map[string]string, error) {
			return r.GetAllNetworkInterfaces(context.Background(), region)
		}},
		{name: "sgs", cacheKey: "sgs", call: func(r *NameResolver) (map[string]string, error) {
			return r.GetAllSecurityGroups(context.Background(), region)
		}},
		{name: "snapshots", cacheKey: "snapshots", call: func(r *NameResolver) (map[string]string, error) {
			return r.GetAllSnapshots(context.Background(), region)
		}},
		{name: "subnets", cacheKey: "subnets", call: func(r *NameResolver) (map[string]string, error) { return r.GetAllSubnets(context.Background(), region) }},
		{name: "volumes", cacheKey: "volumes", call: func(r *NameResolver) (map[string]string, error) { return r.GetAllVolumes(context.Background(), region) }},
		{name: "vpcs", cacheKey: "vpcs", call: func(r *NameResolver) (map[string]string, error) { return r.GetAllVPCs(context.Background(), region) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &NameResolver{
				ec2Clients: map[string]*ec2.Client{},
				cache: map[string]map[string]map[string]string{
					region: {
						tt.cacheKey: cached,
					},
				},
			}

			result, err := tt.call(resolver)

			require.NoError(t, err)
			assert.Equal(t, cached, result)
		})
	}
}

func TestNameResolver_GetAllEC2Resources_NoClientForRegion(t *testing.T) {
	tests := []struct {
		name string
		call func(*NameResolver) error
	}{
		{name: "images", call: func(r *NameResolver) error {
			_, err := r.GetAllImages(context.Background(), "ap-northeast-1")
			return err
		}},
		{name: "enis", call: func(r *NameResolver) error {
			_, err := r.GetAllNetworkInterfaces(context.Background(), "ap-northeast-1")
			return err
		}},
		{name: "sgs", call: func(r *NameResolver) error {
			_, err := r.GetAllSecurityGroups(context.Background(), "ap-northeast-1")
			return err
		}},
		{name: "snapshots", call: func(r *NameResolver) error {
			_, err := r.GetAllSnapshots(context.Background(), "ap-northeast-1")
			return err
		}},
		{name: "subnets", call: func(r *NameResolver) error {
			_, err := r.GetAllSubnets(context.Background(), "ap-northeast-1")
			return err
		}},
		{name: "volumes", call: func(r *NameResolver) error {
			_, err := r.GetAllVolumes(context.Background(), "ap-northeast-1")
			return err
		}},
		{name: "vpcs", call: func(r *NameResolver) error {
			_, err := r.GetAllVPCs(context.Background(), "ap-northeast-1")
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &NameResolver{
				ec2Clients: map[string]*ec2.Client{},
				cache:      make(map[string]map[string]map[string]string),
			}

			err := tt.call(resolver)

			require.Error(t, err)
			assert.ErrorIs(t, err, ErrNoEC2ClientForRegion)
			assert.ErrorContains(t, err, "ap-northeast-1")
		})
	}
}

func TestNameResolver_GetCloudFrontPolicies_CacheHit(t *testing.T) {
	tests := []struct {
		name     string
		cacheKey string
		call     func(*NameResolver) string
	}{
		{name: "origin access control", cacheKey: "oac:test-oac-id", call: func(r *NameResolver) string { return r.GetOriginAccessControlName(context.Background(), "test-oac-id") }},
		{name: "cache policy", cacheKey: "cachepolicy:test-policy-id", call: func(r *NameResolver) string { return r.GetCachePolicyName(context.Background(), "test-policy-id") }},
		{name: "origin request policy", cacheKey: "originrequestpolicy:test-policy-id", call: func(r *NameResolver) string {
			return r.GetOriginRequestPolicyName(context.Background(), "test-policy-id")
		}},
		{name: "response headers policy", cacheKey: "responseheaderspolicy:test-policy-id", call: func(r *NameResolver) string {
			return r.GetResponseHeadersPolicyName(context.Background(), "test-policy-id")
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &NameResolver{
				cloudfrontClients: map[string]*cloudfront.Client{},
				cloudfrontCache: map[string]string{
					tt.cacheKey: "cached-name",
				},
			}

			result := tt.call(resolver)

			assert.Equal(t, "cached-name", result)
		})
	}
}

func TestNameResolver_GetCloudFrontPolicies_NoClient(t *testing.T) {
	tests := []struct {
		name string
		call func(*NameResolver) string
	}{
		{name: "origin access control", call: func(r *NameResolver) string { return r.GetOriginAccessControlName(context.Background(), "test-oac-id") }},
		{name: "cache policy", call: func(r *NameResolver) string { return r.GetCachePolicyName(context.Background(), "test-policy-id") }},
		{name: "origin request policy", call: func(r *NameResolver) string {
			return r.GetOriginRequestPolicyName(context.Background(), "test-policy-id")
		}},
		{name: "response headers policy", call: func(r *NameResolver) string {
			return r.GetResponseHeadersPolicyName(context.Background(), "test-policy-id")
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &NameResolver{
				cloudfrontClients: map[string]*cloudfront.Client{},
				cloudfrontCache:   make(map[string]string),
			}

			result := tt.call(resolver)

			assert.Empty(t, result)
		})
	}
}

package helpers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kmstypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Test helpers/mocks for pagination tests follow.

// ManualKMSClient is a simple, deterministic mock used for pagination tests.
// It returns prepared ListKeys/ListAliases pages in sequence.
type ManualKMSClient struct {
	keys    []*kms.ListKeysOutput
	aliases []*kms.ListAliasesOutput
	ki, ai  int
}

func (m *ManualKMSClient) ListKeys(ctx context.Context, params *kms.ListKeysInput, optFns ...func(*kms.Options)) (*kms.ListKeysOutput, error) {
	if m.ki >= len(m.keys) {
		return &kms.ListKeysOutput{}, nil
	}
	out := m.keys[m.ki]
	m.ki++
	return out, nil
}

func (m *ManualKMSClient) ListAliases(ctx context.Context, params *kms.ListAliasesInput, optFns ...func(*kms.Options)) (*kms.ListAliasesOutput, error) {
	if m.ai >= len(m.aliases) {
		return &kms.ListAliasesOutput{}, nil
	}
	out := m.aliases[m.ai]
	m.ai++
	return out, nil
}

// MockEC2ClientForVPCs is a testify/mock-based mock for DescribeVpcs.
type MockEC2ClientForVPCs struct {
	mock.Mock
}

func (m *MockEC2ClientForVPCs) DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*ec2.DescribeVpcsOutput), args.Error(1)
}

func TestStringValue_PrimitivesAndPointers(t *testing.T) {
	// nil
	assert.Equal(t, "N/A", StringValue(nil))

	// string
	s := "hello"
	assert.Equal(t, "hello", StringValue(s))
	empty := ""
	assert.Equal(t, "N/A", StringValue(empty))

	// *string
	assert.Equal(t, "hello", StringValue(&s))
	assert.Equal(t, "N/A", StringValue(&empty))

	// int family
	assert.Equal(t, "42", StringValue(42))
	var i32 int32 = 7
	assert.Equal(t, "7", StringValue(i32))
	var i64 int64 = 900
	assert.Equal(t, "900", StringValue(i64))

	// floats
	var f32 float32 = 1.5
	var f64 float64 = 2.25
	assert.Equal(t, "1.5", StringValue(f32))
	assert.Equal(t, "2.25", StringValue(f64))

	// bool
	assert.Equal(t, "true", StringValue(true))
	var bptr *bool
	assert.Equal(t, "N/A", StringValue(bptr))

	// time
	z := time.Time{}
	assert.Equal(t, "N/A", StringValue(z))
	now := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	assert.Equal(t, "2020-01-02T03:04:05Z", StringValue(now))

	// slices
	arr := []string{"a", "b"}
	assert.Equal(t, "a\nb", StringValue(arr))
	emptyArr := []string{}
	assert.Equal(t, "N/A", StringValue(emptyArr))

	// []*string
	s1 := "x"
	s2 := ""
	ptrs := []*string{&s1, &s2, nil}
	assert.Equal(t, "x", StringValue(ptrs))
}

func TestStringValue_DoesNotMutateInputSlice(t *testing.T) {
	// ensure StringValue does not mutate the caller's slice order
	src := []string{"b", "a", "c"}
	// make a copy to compare after the call
	original := make([]string, len(src))
	copy(original, src)

	// call StringValue which sorts a copy internally
	out := StringValue(src)
	// output should be sorted
	assert.Equal(t, "a\nb\nc", out)
	// original slice must remain unchanged
	assert.Equal(t, original, src)
}

func TestStringValue_DefaultOverride(t *testing.T) {
	assert.Equal(t, "default", StringValue(nil, "default"))
}

func TestExtractAccountID(t *testing.T) {
	arn := "arn:aws:iam::123456789012:role/test"
	acct, err := ExtractAccountID(arn)
	assert.NoError(t, err)
	assert.Equal(t, "123456789012", acct)

	// invalid
	_, err = ExtractAccountID("invalid-arn")
	assert.Error(t, err)
}

func TestToString(t *testing.T) {
	var p *string
	assert.Equal(t, "", ToString(p))
	s := "ok"
	assert.Equal(t, "ok", ToString(&s))
}

func TestNormalizeRawDataAndGetMapValue(t *testing.T) {
	data := map[string]any{
		"a": nil,
		"b": 123,
		"c": "",
	}
	out := NormalizeRawData(data)
	// after normalization nil and empty string become "N/A"
	assert.Equal(t, "N/A", out["a"])
	assert.Equal(t, "123", out["b"])
	assert.Equal(t, "N/A", out["c"])

	// GetMapValue returns empty string when default is empty
	assert.Equal(t, "", GetMapValue(out, "missing"))
	// but existing keys return their string values
	assert.Equal(t, "N/A", GetMapValue(out, "a"))
}

func TestFormatJSONIndent(t *testing.T) {
	// nil
	s, err := FormatJSONIndent(nil)
	assert.NoError(t, err)
	assert.Equal(t, "", s)

	// empty string
	s2, err := FormatJSONIndent("")
	assert.NoError(t, err)
	assert.Equal(t, "", s2)

	// valid json string
	in := `{"k":1}`
	out, err := FormatJSONIndent(in)
	assert.NoError(t, err)
	assert.Contains(t, out, "\n  \"k\": 1")

	// invalid json string -> error
	_, err = FormatJSONIndent("not-json")
	assert.Error(t, err)

	// map input
	m := map[string]any{"x": "y"}
	out2, err := FormatJSONIndent(m)
	assert.NoError(t, err)
	assert.Contains(t, out2, "\"x\": \"y\"")
}

func TestParseTimestamp_EpochAndRFC3339(t *testing.T) {
	// epoch seconds -> *time.Time
	v := ParseTimestamp("1695601655")
	tptr, ok := v.(*time.Time)
	assert.True(t, ok, "expected *time.Time for epoch string")
	if ok {
		assert.Equal(t, time.Unix(1695601655, 0).UTC().Format(time.RFC3339), StringValue(tptr))
	}

	// RFC3339 -> *time.Time
	v2 := ParseTimestamp("2023-08-10T09:00:00Z")
	tptr2, ok2 := v2.(*time.Time)
	assert.True(t, ok2, "expected *time.Time for RFC3339 string")
	if ok2 {
		assert.Equal(t, "2023-08-10T09:00:00Z", StringValue(tptr2))
	}

	// invalid -> original string
	v3 := ParseTimestamp("not-a-timestamp")
	assert.IsType(t, "", v3)
	assert.Equal(t, "not-a-timestamp", v3)
}

func TestGetAllKMSKeys_Pagination(t *testing.T) {
	ctx := context.Background()

	// page 1 keys
	page1 := &kms.ListKeysOutput{
		Keys:       []kmstypes.KeyListEntry{{KeyId: aws.String("k1"), KeyArn: aws.String("arn1")}},
		Truncated:  true,
		NextMarker: aws.String("m1"),
	}
	page2 := &kms.ListKeysOutput{
		Keys:      []kmstypes.KeyListEntry{{KeyId: aws.String("k2"), KeyArn: aws.String("arn2")}},
		Truncated: false,
	}

	mk := &ManualKMSClient{keys: []*kms.ListKeysOutput{page1, page2}, aliases: []*kms.ListAliasesOutput{}}

	// aliases across two pages
	aliasP1 := &kms.ListAliasesOutput{
		Aliases:    []kmstypes.AliasListEntry{{AliasName: aws.String("alias/one"), TargetKeyId: aws.String("k1")}},
		Truncated:  true,
		NextMarker: aws.String("am1"),
	}
	aliasP2 := &kms.ListAliasesOutput{
		Aliases:   []kmstypes.AliasListEntry{{AliasName: aws.String("alias/two"), TargetKeyId: aws.String("k2")}},
		Truncated: false,
	}
	mk.aliases = []*kms.ListAliasesOutput{aliasP1, aliasP2}

	res, err := getAllKMSKeysWithClient(ctx, mk)
	assert.NoError(t, err)

	expected := map[string]string{
		"k1":        "alias/one",
		"k2":        "alias/two",
		"arn1":      "alias/one",
		"arn2":      "alias/two",
		"alias/one": "alias/one",
		"alias/two": "alias/two",
	}
	assert.Equal(t, expected, res)
}

func TestGetAllVPCs_Pagination(t *testing.T) {
	ctx := context.Background()
	me := &MockEC2ClientForVPCs{}

	// page 1
	v1 := ec2types.Vpc{VpcId: aws.String("vpc-1"), Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("first")}}}
	page1 := &ec2.DescribeVpcsOutput{Vpcs: []ec2types.Vpc{v1}, NextToken: aws.String("t1")}
	v2 := ec2types.Vpc{VpcId: aws.String("vpc-2"), Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("second")}}}
	page2 := &ec2.DescribeVpcsOutput{Vpcs: []ec2types.Vpc{v2}, NextToken: nil}

	me.On("DescribeVpcs", mock.Anything, mock.Anything, mock.Anything).Return(page1, nil).Once()
	me.On("DescribeVpcs", mock.Anything, mock.Anything, mock.Anything).Return(page2, nil).Once()

	res, err := getAllVPCsWithClient(ctx, me)
	assert.NoError(t, err)

	expected := map[string]string{
		"vpc-1": "first",
		"vpc-2": "second",
	}
	assert.Equal(t, expected, res)
	me.AssertExpectations(t)
}

func TestGetAllSecurityGroups_Pagination(t *testing.T) {
	ctx := context.Background()
	me := &MockEC2ClientForSGs{}

	// page 1
	sg1 := ec2types.SecurityGroup{GroupId: aws.String("sg-1"), GroupName: aws.String("first-sg")}
	page1 := &ec2.DescribeSecurityGroupsOutput{SecurityGroups: []ec2types.SecurityGroup{sg1}, NextToken: aws.String("t1")}
	sg2 := ec2types.SecurityGroup{GroupId: aws.String("sg-2"), GroupName: aws.String("second-sg")}
	page2 := &ec2.DescribeSecurityGroupsOutput{SecurityGroups: []ec2types.SecurityGroup{sg2}, NextToken: nil}

	me.On("DescribeSecurityGroups", mock.Anything, mock.Anything, mock.Anything).Return(page1, nil).Once()
	me.On("DescribeSecurityGroups", mock.Anything, mock.Anything, mock.Anything).Return(page2, nil).Once()

	res, err := getAllSecurityGroupsWithClient(ctx, me)
	assert.NoError(t, err)

	expected := map[string]string{
		"sg-1": "first-sg",
		"sg-2": "second-sg",
	}
	assert.Equal(t, expected, res)
	me.AssertExpectations(t)
}

func TestGetAllSubnets_Pagination(t *testing.T) {
	ctx := context.Background()
	me := &MockEC2ClientForSubnets{}

	// page 1
	subnet1 := ec2types.Subnet{SubnetId: aws.String("subnet-1"), Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("first-subnet")}}}
	page1 := &ec2.DescribeSubnetsOutput{Subnets: []ec2types.Subnet{subnet1}, NextToken: aws.String("t1")}
	subnet2 := ec2types.Subnet{SubnetId: aws.String("subnet-2"), Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("second-subnet")}}}
	page2 := &ec2.DescribeSubnetsOutput{Subnets: []ec2types.Subnet{subnet2}, NextToken: nil}

	me.On("DescribeSubnets", mock.Anything, mock.Anything, mock.Anything).Return(page1, nil).Once()
	me.On("DescribeSubnets", mock.Anything, mock.Anything, mock.Anything).Return(page2, nil).Once()

	res, err := getAllSubnetsWithClient(ctx, me)
	assert.NoError(t, err)

	expected := map[string]string{
		"subnet-1": "first-subnet",
		"subnet-2": "second-subnet",
	}
	assert.Equal(t, expected, res)
	me.AssertExpectations(t)
}

func TestGetAllImages_Pagination(t *testing.T) {
	ctx := context.Background()
	me := &MockEC2ClientForImages{}

	// page 1
	image1 := ec2types.Image{ImageId: aws.String("ami-1"), Name: aws.String("first-image")}
	page1 := &ec2.DescribeImagesOutput{Images: []ec2types.Image{image1}, NextToken: aws.String("t1")}
	image2 := ec2types.Image{ImageId: aws.String("ami-2"), Name: aws.String("second-image")}
	page2 := &ec2.DescribeImagesOutput{Images: []ec2types.Image{image2}, NextToken: nil}

	me.On("DescribeImages", mock.Anything, mock.Anything, mock.Anything).Return(page1, nil).Once()
	me.On("DescribeImages", mock.Anything, mock.Anything, mock.Anything).Return(page2, nil).Once()

	res, err := getAllImagesWithClient(ctx, me)
	assert.NoError(t, err)

	expected := map[string]string{
		"ami-1": "first-image",
		"ami-2": "second-image",
	}
	assert.Equal(t, expected, res)
	me.AssertExpectations(t)
}

func TestGetAllSnapshots_Pagination(t *testing.T) {
	ctx := context.Background()
	me := &MockEC2ClientForSnapshots{}

	// page 1
	snapshot1 := ec2types.Snapshot{SnapshotId: aws.String("snap-1"), Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("first-snapshot")}}}
	page1 := &ec2.DescribeSnapshotsOutput{Snapshots: []ec2types.Snapshot{snapshot1}, NextToken: aws.String("t1")}
	snapshot2 := ec2types.Snapshot{SnapshotId: aws.String("snap-2"), Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("second-snapshot")}}}
	page2 := &ec2.DescribeSnapshotsOutput{Snapshots: []ec2types.Snapshot{snapshot2}, NextToken: nil}

	me.On("DescribeSnapshots", mock.Anything, mock.Anything, mock.Anything).Return(page1, nil).Once()
	me.On("DescribeSnapshots", mock.Anything, mock.Anything, mock.Anything).Return(page2, nil).Once()

	res, err := getAllSnapshotsWithClient(ctx, me)
	assert.NoError(t, err)

	expected := map[string]string{
		"snap-1": "first-snapshot",
		"snap-2": "second-snapshot",
	}
	assert.Equal(t, expected, res)
	me.AssertExpectations(t)
}

func TestGetAllVolumes_Pagination(t *testing.T) {
	ctx := context.Background()
	me := &MockEC2ClientForVolumes{}

	// page 1
	volume1 := ec2types.Volume{VolumeId: aws.String("vol-1"), Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("first-volume")}}}
	page1 := &ec2.DescribeVolumesOutput{Volumes: []ec2types.Volume{volume1}, NextToken: aws.String("t1")}
	volume2 := ec2types.Volume{VolumeId: aws.String("vol-2"), Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("second-volume")}}}
	page2 := &ec2.DescribeVolumesOutput{Volumes: []ec2types.Volume{volume2}, NextToken: nil}

	me.On("DescribeVolumes", mock.Anything, mock.Anything, mock.Anything).Return(page1, nil).Once()
	me.On("DescribeVolumes", mock.Anything, mock.Anything, mock.Anything).Return(page2, nil).Once()

	res, err := getAllVolumesWithClient(ctx, me)
	assert.NoError(t, err)

	expected := map[string]string{
		"vol-1": "first-volume",
		"vol-2": "second-volume",
	}
	assert.Equal(t, expected, res)
	me.AssertExpectations(t)
}

func TestGetAllNetworkInterfaces_Pagination(t *testing.T) {
	ctx := context.Background()
	me := &MockEC2ClientForENIs{}

	// page 1
	eni1 := ec2types.NetworkInterface{NetworkInterfaceId: aws.String("eni-1"), TagSet: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("first-eni")}}}
	page1 := &ec2.DescribeNetworkInterfacesOutput{NetworkInterfaces: []ec2types.NetworkInterface{eni1}, NextToken: aws.String("t1")}
	eni2 := ec2types.NetworkInterface{NetworkInterfaceId: aws.String("eni-2"), TagSet: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("second-eni")}}}
	page2 := &ec2.DescribeNetworkInterfacesOutput{NetworkInterfaces: []ec2types.NetworkInterface{eni2}, NextToken: nil}

	me.On("DescribeNetworkInterfaces", mock.Anything, mock.Anything, mock.Anything).Return(page1, nil).Once()
	me.On("DescribeNetworkInterfaces", mock.Anything, mock.Anything, mock.Anything).Return(page2, nil).Once()

	res, err := getAllNetworkInterfacesWithClient(ctx, me)
	assert.NoError(t, err)

	expected := map[string]string{
		"eni-1": "first-eni",
		"eni-2": "second-eni",
	}
	assert.Equal(t, expected, res)
	me.AssertExpectations(t)
}

// Additional testify mock types for other EC2 Describe APIs used in tests.
type MockEC2ClientForSGs struct{ mock.Mock }

func (m *MockEC2ClientForSGs) DescribeSecurityGroups(ctx context.Context, params *ec2.DescribeSecurityGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*ec2.DescribeSecurityGroupsOutput), args.Error(1)
}

type MockEC2ClientForSubnets struct{ mock.Mock }

func (m *MockEC2ClientForSubnets) DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*ec2.DescribeSubnetsOutput), args.Error(1)
}

type MockEC2ClientForImages struct{ mock.Mock }

func (m *MockEC2ClientForImages) DescribeImages(ctx context.Context, params *ec2.DescribeImagesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*ec2.DescribeImagesOutput), args.Error(1)
}

type MockEC2ClientForSnapshots struct{ mock.Mock }

func (m *MockEC2ClientForSnapshots) DescribeSnapshots(ctx context.Context, params *ec2.DescribeSnapshotsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSnapshotsOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*ec2.DescribeSnapshotsOutput), args.Error(1)
}

type MockEC2ClientForVolumes struct{ mock.Mock }

func (m *MockEC2ClientForVolumes) DescribeVolumes(ctx context.Context, params *ec2.DescribeVolumesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVolumesOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*ec2.DescribeVolumesOutput), args.Error(1)
}

type MockEC2ClientForENIs struct{ mock.Mock }

func (m *MockEC2ClientForENIs) DescribeNetworkInterfaces(ctx context.Context, params *ec2.DescribeNetworkInterfacesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*ec2.DescribeNetworkInterfacesOutput), args.Error(1)
}

func TestGetAllSecurityGroups_WithClientPagination(t *testing.T) {
	ctx := context.Background()
	m := &MockEC2ClientForSGs{}

	sg1 := ec2types.SecurityGroup{GroupId: aws.String("sg-1"), GroupName: aws.String("one")}
	page1 := &ec2.DescribeSecurityGroupsOutput{SecurityGroups: []ec2types.SecurityGroup{sg1}, NextToken: aws.String("t1")}
	sg2 := ec2types.SecurityGroup{GroupId: aws.String("sg-2"), GroupName: aws.String("")}
	page2 := &ec2.DescribeSecurityGroupsOutput{SecurityGroups: []ec2types.SecurityGroup{sg2}, NextToken: nil}

	m.On("DescribeSecurityGroups", mock.Anything, mock.Anything).Return(page1, nil).Once()
	m.On("DescribeSecurityGroups", mock.Anything, mock.Anything).Return(page2, nil).Once()

	res, err := getAllSecurityGroupsWithClient(ctx, m)
	assert.NoError(t, err)
	expected := map[string]string{"sg-1": "one", "sg-2": "sg-2"}
	assert.Equal(t, expected, res)
	m.AssertExpectations(t)
}

func TestGetAllSubnets_WithClientPagination(t *testing.T) {
	ctx := context.Background()
	m := &MockEC2ClientForSubnets{}

	s1 := ec2types.Subnet{SubnetId: aws.String("sub-1"), Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("first")}}}
	page1 := &ec2.DescribeSubnetsOutput{Subnets: []ec2types.Subnet{s1}, NextToken: aws.String("t1")}
	s2 := ec2types.Subnet{SubnetId: aws.String("sub-2"), Tags: []ec2types.Tag{}}
	page2 := &ec2.DescribeSubnetsOutput{Subnets: []ec2types.Subnet{s2}, NextToken: nil}

	m.On("DescribeSubnets", mock.Anything, mock.Anything).Return(page1, nil).Once()
	m.On("DescribeSubnets", mock.Anything, mock.Anything).Return(page2, nil).Once()

	res, err := getAllSubnetsWithClient(ctx, m)
	assert.NoError(t, err)
	expected := map[string]string{"sub-1": "first", "sub-2": "sub-2"}
	assert.Equal(t, expected, res)
	m.AssertExpectations(t)
}

func TestGetAllImages_WithClientPagination(t *testing.T) {
	ctx := context.Background()
	m := &MockEC2ClientForImages{}

	img1 := ec2types.Image{ImageId: aws.String("ami-1"), Name: aws.String("img-one"), Tags: []ec2types.Tag{}}
	page1 := &ec2.DescribeImagesOutput{Images: []ec2types.Image{img1}, NextToken: aws.String("t1")}
	img2 := ec2types.Image{ImageId: aws.String("ami-2"), Name: nil, Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("img-two")}}}
	page2 := &ec2.DescribeImagesOutput{Images: []ec2types.Image{img2}, NextToken: nil}

	m.On("DescribeImages", mock.Anything, mock.Anything).Return(page1, nil).Once()
	m.On("DescribeImages", mock.Anything, mock.Anything).Return(page2, nil).Once()

	res, err := getAllImagesWithClient(ctx, m)
	assert.NoError(t, err)
	expected := map[string]string{"ami-1": "img-one", "ami-2": "img-two"}
	assert.Equal(t, expected, res)
	m.AssertExpectations(t)
}

func TestGetAllSnapshots_WithClientPagination(t *testing.T) {
	ctx := context.Background()
	m := &MockEC2ClientForSnapshots{}

	snap1 := ec2types.Snapshot{SnapshotId: aws.String("snap-1"), Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("snap-one")}}}
	page1 := &ec2.DescribeSnapshotsOutput{Snapshots: []ec2types.Snapshot{snap1}, NextToken: aws.String("t1")}
	snap2 := ec2types.Snapshot{SnapshotId: aws.String("snap-2"), Tags: []ec2types.Tag{}}
	page2 := &ec2.DescribeSnapshotsOutput{Snapshots: []ec2types.Snapshot{snap2}, NextToken: nil}

	m.On("DescribeSnapshots", mock.Anything, mock.Anything).Return(page1, nil).Once()
	m.On("DescribeSnapshots", mock.Anything, mock.Anything).Return(page2, nil).Once()

	res, err := getAllSnapshotsWithClient(ctx, m)
	assert.NoError(t, err)
	expected := map[string]string{"snap-1": "snap-one", "snap-2": "snap-2"}
	assert.Equal(t, expected, res)
	m.AssertExpectations(t)
}

func TestGetAllVolumes_WithClientPagination(t *testing.T) {
	ctx := context.Background()
	m := &MockEC2ClientForVolumes{}

	v1 := ec2types.Volume{VolumeId: aws.String("vol-1"), Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("vol-one")}}}
	page1 := &ec2.DescribeVolumesOutput{Volumes: []ec2types.Volume{v1}, NextToken: aws.String("t1")}
	v2 := ec2types.Volume{VolumeId: aws.String("vol-2"), Tags: []ec2types.Tag{}}
	page2 := &ec2.DescribeVolumesOutput{Volumes: []ec2types.Volume{v2}, NextToken: nil}

	m.On("DescribeVolumes", mock.Anything, mock.Anything).Return(page1, nil).Once()
	m.On("DescribeVolumes", mock.Anything, mock.Anything).Return(page2, nil).Once()

	res, err := getAllVolumesWithClient(ctx, m)
	assert.NoError(t, err)
	expected := map[string]string{"vol-1": "vol-one", "vol-2": "vol-2"}
	assert.Equal(t, expected, res)
	m.AssertExpectations(t)
}

func TestGetAllNetworkInterfaces_WithClientPagination(t *testing.T) {
	ctx := context.Background()
	m := &MockEC2ClientForENIs{}

	eni1 := ec2types.NetworkInterface{NetworkInterfaceId: aws.String("eni-1"), TagSet: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("eni-one")}}}
	page1 := &ec2.DescribeNetworkInterfacesOutput{NetworkInterfaces: []ec2types.NetworkInterface{eni1}, NextToken: aws.String("t1")}
	eni2 := ec2types.NetworkInterface{NetworkInterfaceId: aws.String("eni-2"), TagSet: []ec2types.Tag{}}
	page2 := &ec2.DescribeNetworkInterfacesOutput{NetworkInterfaces: []ec2types.NetworkInterface{eni2}, NextToken: nil}

	m.On("DescribeNetworkInterfaces", mock.Anything, mock.Anything).Return(page1, nil).Once()
	m.On("DescribeNetworkInterfaces", mock.Anything, mock.Anything).Return(page2, nil).Once()

	res, err := getAllNetworkInterfacesWithClient(ctx, m)
	assert.NoError(t, err)
	expected := map[string]string{"eni-1": "eni-one", "eni-2": "eni-2"}
	assert.Equal(t, expected, res)
	m.AssertExpectations(t)
}

func TestGetAllSecurityGroups_InvalidClient(t *testing.T) {
	ctx := context.Background()
	// Provide a client that does not implement DescribeSecurityGroups
	wrong := &MockEC2ClientForVPCs{}

	_, err := getAllSecurityGroupsWithClient(ctx, wrong)
	assert.ErrorIs(t, err, ErrClientNotDescribeSGs)
}

func TestGetAllVolumes_ClientError(t *testing.T) {
	ctx := context.Background()
	m := &MockEC2ClientForVolumes{}
	// simulate API error
	m.On("DescribeVolumes", mock.Anything, mock.Anything).Return((*ec2.DescribeVolumesOutput)(nil), assert.AnError).Once()

	_, err := getAllVolumesWithClient(ctx, m)
	assert.Error(t, err)
	m.AssertExpectations(t)
}

// Mock KMS client that returns an error to validate error handling in paginator loops
type MockKMSClientWithError struct{}

func (m *MockKMSClientWithError) ListKeys(ctx context.Context, params *kms.ListKeysInput, optFns ...func(*kms.Options)) (*kms.ListKeysOutput, error) {
	return nil, fmt.Errorf("kms failed")
}
func (m *MockKMSClientWithError) ListAliases(ctx context.Context, params *kms.ListAliasesInput, optFns ...func(*kms.Options)) (*kms.ListAliasesOutput, error) {
	return &kms.ListAliasesOutput{}, nil
}

func TestGetAllKMSKeys_ListKeysError(t *testing.T) {
	ctx := context.Background()
	mk := &MockKMSClientWithError{}
	_, err := getAllKMSKeysWithClient(ctx, mk)
	assert.Error(t, err)
}

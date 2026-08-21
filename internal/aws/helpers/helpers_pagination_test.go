package helpers

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kmstypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/y-miyazaki/arc/internal/aws/helpers/mocks"
)

var errTestDescribeVolumes = errors.New("describe volumes failed")

// ManualKMSClient is a simple, deterministic stub used for pagination tests.
// It returns prepared ListKeys/ListAliases pages in sequence.
type ManualKMSClient struct {
	keys    []*kms.ListKeysOutput
	aliases []*kms.ListAliasesOutput
	ki, ai  int
}

func (m *ManualKMSClient) ListKeys(_ context.Context, _ *kms.ListKeysInput, _ ...func(*kms.Options)) (*kms.ListKeysOutput, error) {
	if m.ki >= len(m.keys) {
		return &kms.ListKeysOutput{}, nil
	}
	out := m.keys[m.ki]
	m.ki++
	return out, nil
}

func (m *ManualKMSClient) ListAliases(_ context.Context, _ *kms.ListAliasesInput, _ ...func(*kms.Options)) (*kms.ListAliasesOutput, error) {
	if m.ai >= len(m.aliases) {
		return &kms.ListAliasesOutput{}, nil
	}
	out := m.aliases[m.ai]
	m.ai++
	return out, nil
}

func TestGetAllKMSKeys_Pagination(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	page1 := &kms.ListKeysOutput{
		Keys:       []kmstypes.KeyListEntry{{KeyId: aws.String("k1"), KeyArn: aws.String("arn1")}},
		NextMarker: aws.String("m1"),
	}
	page2 := &kms.ListKeysOutput{
		Keys:      []kmstypes.KeyListEntry{{KeyId: aws.String("k2"), KeyArn: aws.String("arn2")}},
		Truncated: false,
	}
	mk := &ManualKMSClient{keys: []*kms.ListKeysOutput{page1, page2}, aliases: []*kms.ListAliasesOutput{}}

	aliasP1 := &kms.ListAliasesOutput{
		Aliases:    []kmstypes.AliasListEntry{{AliasName: aws.String("alias/one"), TargetKeyId: aws.String("k1")}},
		NextMarker: aws.String("am1"),
	}
	aliasP2 := &kms.ListAliasesOutput{
		Aliases:   []kmstypes.AliasListEntry{{AliasName: aws.String("alias/two"), TargetKeyId: aws.String("k2")}},
		Truncated: false,
	}
	mk.aliases = []*kms.ListAliasesOutput{aliasP1, aliasP2}

	res, err := getAllKMSKeysWithClient(ctx, mk)
	require.NoError(t, err)

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
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockClient := mocks.NewMockEC2DescribeVpcsClientInterface(ctrl)

	v1 := ec2types.Vpc{VpcId: aws.String("vpc-1"), Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("first")}}}
	page1 := &ec2.DescribeVpcsOutput{Vpcs: []ec2types.Vpc{v1}, NextToken: aws.String("t1")}
	v2 := ec2types.Vpc{VpcId: aws.String("vpc-2"), Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("second")}}}
	page2 := &ec2.DescribeVpcsOutput{Vpcs: []ec2types.Vpc{v2}, NextToken: nil}

	mockClient.EXPECT().DescribeVpcs(gomock.Any(), gomock.Any(), gomock.Any()).Return(page1, nil)
	mockClient.EXPECT().DescribeVpcs(gomock.Any(), gomock.Any(), gomock.Any()).Return(page2, nil)

	res, err := getAllVPCsWithClient(context.Background(), mockClient)
	require.NoError(t, err)

	expected := map[string]string{
		"vpc-1": "first",
		"vpc-2": "second",
	}
	assert.Equal(t, expected, res)
}

func TestGetAllSecurityGroups_Pagination(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockClient := mocks.NewMockEC2DescribeSecurityGroupsClientInterface(ctrl)

	sg1 := ec2types.SecurityGroup{GroupId: aws.String("sg-1"), GroupName: aws.String("first-sg")}
	page1 := &ec2.DescribeSecurityGroupsOutput{SecurityGroups: []ec2types.SecurityGroup{sg1}, NextToken: aws.String("t1")}
	sg2 := ec2types.SecurityGroup{GroupId: aws.String("sg-2"), GroupName: aws.String("second-sg")}
	page2 := &ec2.DescribeSecurityGroupsOutput{SecurityGroups: []ec2types.SecurityGroup{sg2}, NextToken: nil}

	mockClient.EXPECT().DescribeSecurityGroups(gomock.Any(), gomock.Any(), gomock.Any()).Return(page1, nil)
	mockClient.EXPECT().DescribeSecurityGroups(gomock.Any(), gomock.Any(), gomock.Any()).Return(page2, nil)

	res, err := getAllSecurityGroupsWithClient(context.Background(), mockClient)
	require.NoError(t, err)

	expected := map[string]string{
		"sg-1": "first-sg",
		"sg-2": "second-sg",
	}
	assert.Equal(t, expected, res)
}

func TestGetAllSubnets_Pagination(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockClient := mocks.NewMockEC2DescribeSubnetsClientInterface(ctrl)

	subnet1 := ec2types.Subnet{SubnetId: aws.String("subnet-1"), Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("first-subnet")}}}
	page1 := &ec2.DescribeSubnetsOutput{Subnets: []ec2types.Subnet{subnet1}, NextToken: aws.String("t1")}
	subnet2 := ec2types.Subnet{SubnetId: aws.String("subnet-2"), Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("second-subnet")}}}
	page2 := &ec2.DescribeSubnetsOutput{Subnets: []ec2types.Subnet{subnet2}, NextToken: nil}

	mockClient.EXPECT().DescribeSubnets(gomock.Any(), gomock.Any(), gomock.Any()).Return(page1, nil)
	mockClient.EXPECT().DescribeSubnets(gomock.Any(), gomock.Any(), gomock.Any()).Return(page2, nil)

	res, err := getAllSubnetsWithClient(context.Background(), mockClient)
	require.NoError(t, err)

	expected := map[string]string{
		"subnet-1": "first-subnet",
		"subnet-2": "second-subnet",
	}
	assert.Equal(t, expected, res)
}

func TestGetAllImages_Pagination(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockClient := mocks.NewMockEC2DescribeImagesClientInterface(ctrl)

	image1 := ec2types.Image{ImageId: aws.String("ami-1"), Name: aws.String("first-image")}
	page1 := &ec2.DescribeImagesOutput{Images: []ec2types.Image{image1}, NextToken: aws.String("t1")}
	image2 := ec2types.Image{ImageId: aws.String("ami-2"), Name: aws.String("second-image")}
	page2 := &ec2.DescribeImagesOutput{Images: []ec2types.Image{image2}, NextToken: nil}

	mockClient.EXPECT().DescribeImages(gomock.Any(), gomock.Any(), gomock.Any()).Return(page1, nil)
	mockClient.EXPECT().DescribeImages(gomock.Any(), gomock.Any(), gomock.Any()).Return(page2, nil)

	res, err := getAllImagesWithClient(context.Background(), mockClient)
	require.NoError(t, err)

	expected := map[string]string{
		"ami-1": "first-image",
		"ami-2": "second-image",
	}
	assert.Equal(t, expected, res)
}

func TestGetAllSnapshots_Pagination(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockClient := mocks.NewMockEC2DescribeSnapshotsClientInterface(ctrl)

	snapshot1 := ec2types.Snapshot{SnapshotId: aws.String("snap-1"), Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("first-snapshot")}}}
	page1 := &ec2.DescribeSnapshotsOutput{Snapshots: []ec2types.Snapshot{snapshot1}, NextToken: aws.String("t1")}
	snapshot2 := ec2types.Snapshot{SnapshotId: aws.String("snap-2"), Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("second-snapshot")}}}
	page2 := &ec2.DescribeSnapshotsOutput{Snapshots: []ec2types.Snapshot{snapshot2}, NextToken: nil}

	mockClient.EXPECT().DescribeSnapshots(gomock.Any(), gomock.Any(), gomock.Any()).Return(page1, nil)
	mockClient.EXPECT().DescribeSnapshots(gomock.Any(), gomock.Any(), gomock.Any()).Return(page2, nil)

	res, err := getAllSnapshotsWithClient(context.Background(), mockClient)
	require.NoError(t, err)

	expected := map[string]string{
		"snap-1": "first-snapshot",
		"snap-2": "second-snapshot",
	}
	assert.Equal(t, expected, res)
}

func TestGetAllVolumes_Pagination(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockClient := mocks.NewMockEC2DescribeVolumesClientInterface(ctrl)

	volume1 := ec2types.Volume{VolumeId: aws.String("vol-1"), Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("first-volume")}}}
	page1 := &ec2.DescribeVolumesOutput{Volumes: []ec2types.Volume{volume1}, NextToken: aws.String("t1")}
	volume2 := ec2types.Volume{VolumeId: aws.String("vol-2"), Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("second-volume")}}}
	page2 := &ec2.DescribeVolumesOutput{Volumes: []ec2types.Volume{volume2}, NextToken: nil}

	mockClient.EXPECT().DescribeVolumes(gomock.Any(), gomock.Any(), gomock.Any()).Return(page1, nil)
	mockClient.EXPECT().DescribeVolumes(gomock.Any(), gomock.Any(), gomock.Any()).Return(page2, nil)

	res, err := getAllVolumesWithClient(context.Background(), mockClient)
	require.NoError(t, err)

	expected := map[string]string{
		"vol-1": "first-volume",
		"vol-2": "second-volume",
	}
	assert.Equal(t, expected, res)
}

func TestGetAllNetworkInterfaces_Pagination(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockClient := mocks.NewMockEC2DescribeNetworkInterfacesClientInterface(ctrl)

	eni1 := ec2types.NetworkInterface{NetworkInterfaceId: aws.String("eni-1"), TagSet: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("first-eni")}}}
	page1 := &ec2.DescribeNetworkInterfacesOutput{NetworkInterfaces: []ec2types.NetworkInterface{eni1}, NextToken: aws.String("t1")}
	eni2 := ec2types.NetworkInterface{NetworkInterfaceId: aws.String("eni-2"), TagSet: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("second-eni")}}}
	page2 := &ec2.DescribeNetworkInterfacesOutput{NetworkInterfaces: []ec2types.NetworkInterface{eni2}, NextToken: nil}

	mockClient.EXPECT().DescribeNetworkInterfaces(gomock.Any(), gomock.Any(), gomock.Any()).Return(page1, nil)
	mockClient.EXPECT().DescribeNetworkInterfaces(gomock.Any(), gomock.Any(), gomock.Any()).Return(page2, nil)

	res, err := getAllNetworkInterfacesWithClient(context.Background(), mockClient)
	require.NoError(t, err)

	expected := map[string]string{
		"eni-1": "first-eni",
		"eni-2": "second-eni",
	}
	assert.Equal(t, expected, res)
}

func TestGetAllSecurityGroups_WithClientPagination(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockClient := mocks.NewMockEC2DescribeSecurityGroupsClientInterface(ctrl)

	sg1 := ec2types.SecurityGroup{GroupId: aws.String("sg-1"), GroupName: aws.String("one")}
	page1 := &ec2.DescribeSecurityGroupsOutput{SecurityGroups: []ec2types.SecurityGroup{sg1}, NextToken: aws.String("t1")}
	sg2 := ec2types.SecurityGroup{GroupId: aws.String("sg-2"), GroupName: aws.String("")}
	page2 := &ec2.DescribeSecurityGroupsOutput{SecurityGroups: []ec2types.SecurityGroup{sg2}, NextToken: nil}

	mockClient.EXPECT().DescribeSecurityGroups(gomock.Any(), gomock.Any(), gomock.Any()).Return(page1, nil)
	mockClient.EXPECT().DescribeSecurityGroups(gomock.Any(), gomock.Any(), gomock.Any()).Return(page2, nil)

	res, err := getAllSecurityGroupsWithClient(context.Background(), mockClient)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"sg-1": "one", "sg-2": "sg-2"}, res)
}

func TestGetAllSubnets_WithClientPagination(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockClient := mocks.NewMockEC2DescribeSubnetsClientInterface(ctrl)

	s1 := ec2types.Subnet{SubnetId: aws.String("sub-1"), Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("first")}}}
	page1 := &ec2.DescribeSubnetsOutput{Subnets: []ec2types.Subnet{s1}, NextToken: aws.String("t1")}
	s2 := ec2types.Subnet{SubnetId: aws.String("sub-2"), Tags: []ec2types.Tag{}}
	page2 := &ec2.DescribeSubnetsOutput{Subnets: []ec2types.Subnet{s2}, NextToken: nil}

	mockClient.EXPECT().DescribeSubnets(gomock.Any(), gomock.Any(), gomock.Any()).Return(page1, nil)
	mockClient.EXPECT().DescribeSubnets(gomock.Any(), gomock.Any(), gomock.Any()).Return(page2, nil)

	res, err := getAllSubnetsWithClient(context.Background(), mockClient)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"sub-1": "first", "sub-2": "sub-2"}, res)
}

func TestGetAllImages_WithClientPagination(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockClient := mocks.NewMockEC2DescribeImagesClientInterface(ctrl)

	img1 := ec2types.Image{ImageId: aws.String("ami-1"), Name: aws.String("img-one"), Tags: []ec2types.Tag{}}
	page1 := &ec2.DescribeImagesOutput{Images: []ec2types.Image{img1}, NextToken: aws.String("t1")}
	img2 := ec2types.Image{ImageId: aws.String("ami-2"), Name: nil, Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("img-two")}}}
	page2 := &ec2.DescribeImagesOutput{Images: []ec2types.Image{img2}, NextToken: nil}

	mockClient.EXPECT().DescribeImages(gomock.Any(), gomock.Any(), gomock.Any()).Return(page1, nil)
	mockClient.EXPECT().DescribeImages(gomock.Any(), gomock.Any(), gomock.Any()).Return(page2, nil)

	res, err := getAllImagesWithClient(context.Background(), mockClient)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"ami-1": "img-one", "ami-2": "img-two"}, res)
}

func TestGetAllSnapshots_WithClientPagination(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockClient := mocks.NewMockEC2DescribeSnapshotsClientInterface(ctrl)

	snap1 := ec2types.Snapshot{SnapshotId: aws.String("snap-1"), Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("snap-one")}}}
	page1 := &ec2.DescribeSnapshotsOutput{Snapshots: []ec2types.Snapshot{snap1}, NextToken: aws.String("t1")}
	snap2 := ec2types.Snapshot{SnapshotId: aws.String("snap-2"), Tags: []ec2types.Tag{}}
	page2 := &ec2.DescribeSnapshotsOutput{Snapshots: []ec2types.Snapshot{snap2}, NextToken: nil}

	mockClient.EXPECT().DescribeSnapshots(gomock.Any(), gomock.Any(), gomock.Any()).Return(page1, nil)
	mockClient.EXPECT().DescribeSnapshots(gomock.Any(), gomock.Any(), gomock.Any()).Return(page2, nil)

	res, err := getAllSnapshotsWithClient(context.Background(), mockClient)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"snap-1": "snap-one", "snap-2": "snap-2"}, res)
}

func TestGetAllVolumes_WithClientPagination(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockClient := mocks.NewMockEC2DescribeVolumesClientInterface(ctrl)

	v1 := ec2types.Volume{VolumeId: aws.String("vol-1"), Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("vol-one")}}}
	page1 := &ec2.DescribeVolumesOutput{Volumes: []ec2types.Volume{v1}, NextToken: aws.String("t1")}
	v2 := ec2types.Volume{VolumeId: aws.String("vol-2"), Tags: []ec2types.Tag{}}
	page2 := &ec2.DescribeVolumesOutput{Volumes: []ec2types.Volume{v2}, NextToken: nil}

	mockClient.EXPECT().DescribeVolumes(gomock.Any(), gomock.Any(), gomock.Any()).Return(page1, nil)
	mockClient.EXPECT().DescribeVolumes(gomock.Any(), gomock.Any(), gomock.Any()).Return(page2, nil)

	res, err := getAllVolumesWithClient(context.Background(), mockClient)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"vol-1": "vol-one", "vol-2": "vol-2"}, res)
}

func TestGetAllNetworkInterfaces_WithClientPagination(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockClient := mocks.NewMockEC2DescribeNetworkInterfacesClientInterface(ctrl)

	eni1 := ec2types.NetworkInterface{NetworkInterfaceId: aws.String("eni-1"), TagSet: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("eni-one")}}}
	page1 := &ec2.DescribeNetworkInterfacesOutput{NetworkInterfaces: []ec2types.NetworkInterface{eni1}, NextToken: aws.String("t1")}
	eni2 := ec2types.NetworkInterface{NetworkInterfaceId: aws.String("eni-2"), TagSet: []ec2types.Tag{}}
	page2 := &ec2.DescribeNetworkInterfacesOutput{NetworkInterfaces: []ec2types.NetworkInterface{eni2}, NextToken: nil}

	mockClient.EXPECT().DescribeNetworkInterfaces(gomock.Any(), gomock.Any(), gomock.Any()).Return(page1, nil)
	mockClient.EXPECT().DescribeNetworkInterfaces(gomock.Any(), gomock.Any(), gomock.Any()).Return(page2, nil)

	res, err := getAllNetworkInterfacesWithClient(context.Background(), mockClient)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"eni-1": "eni-one", "eni-2": "eni-2"}, res)
}

func TestGetAllSecurityGroups_InvalidClient(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	wrong := mocks.NewMockEC2DescribeVpcsClientInterface(ctrl)

	_, err := getAllSecurityGroupsWithClient(context.Background(), wrong)
	assert.ErrorIs(t, err, ErrClientNotDescribeSGs)
}

func TestGetAllVolumes_ClientError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockClient := mocks.NewMockEC2DescribeVolumesClientInterface(ctrl)
	mockClient.EXPECT().DescribeVolumes(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errTestDescribeVolumes)

	_, err := getAllVolumesWithClient(context.Background(), mockClient)
	require.Error(t, err)
}

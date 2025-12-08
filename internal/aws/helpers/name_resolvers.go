// Package helpers provides helper functions for AWS resource collection.
package helpers

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/kms"
)

const (
	// NotAvailable represents a not available resource.
	NotAvailable = "N/A"
	// ARNPartCount is the expected number of parts in an ARN.
	ARNPartCount = 6
	// ARNPartitionIndex is the index of the partition in ARN parts.
	ARNPartitionIndex = 1
	// ARNServiceIndex is the index of the service in ARN parts.
	ARNServiceIndex = 2
	// ARNRegionIndex is the index of the region in ARN parts.
	ARNRegionIndex = 3
	// ARNAccountIndex is the index of the account in ARN parts.
	ARNAccountIndex = 4
	// ARNResourceIndex is the index of the resource in ARN parts.
	ARNResourceIndex = 5
	// ResourcePartCount is the number of parts when splitting resource.
	ResourcePartCount = 2
	// Colon is the colon character.
	Colon = ":"
	// TagNameKey is the standard AWS tag key for resource names.
	TagNameKey = "Name"
)

// GetTagValue retrieves the value of a tag by key (case-insensitive) from EC2 tags.
func GetTagValue(tags []ec2types.Tag, key string) string {
	lowerKey := strings.ToLower(key)
	for _, tag := range tags {
		if strings.ToLower(aws.ToString(tag.Key)) == lowerKey {
			return aws.ToString(tag.Value)
		}
	}
	return ""
}

// GetAllKMSKeys retrieves all KMS keys and their aliases in the region.
// Returns a map where both key ID and key ARN can be used as lookup keys to get the alias name.
// This allows lookups with either format: key ID (e.g., "12345678-1234-1234-1234-123456789012")
// or full ARN (e.g., "arn:aws:kms:region:account:key/12345678-1234-1234-1234-123456789012").
func GetAllKMSKeys(ctx context.Context, cfg *aws.Config, region string) (map[string]string, error) {
	svc := kms.NewFromConfig(*cfg, func(o *kms.Options) {
		o.Region = region
	})

	// Get all KMS keys to build ARNs
	keysResult, err := svc.ListKeys(ctx, &kms.ListKeysInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list KMS keys: %w", err)
	}

	// Get all aliases
	aliases, err := svc.ListAliases(ctx, &kms.ListAliasesInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list KMS aliases: %w", err)
	}

	keyMap := make(map[string]string)

	// Create a map with both key ID and ARN as keys
	for i := range aliases.Aliases {
		alias := &aliases.Aliases[i]
		if alias.TargetKeyId == nil || alias.AliasName == nil {
			continue
		}
		keyID := aws.ToString(alias.TargetKeyId)
		aliasName := aws.ToString(alias.AliasName)
		// Remove "alias/" prefix from alias name
		if after, ok := strings.CutPrefix(aliasName, "alias/"); ok {
			aliasName = after
		}

		// Add mapping for key ID
		keyMap[keyID] = aliasName

		// Find matching key ARN from ListKeys result
		for j := range keysResult.Keys {
			key := &keysResult.Keys[j]
			if key.KeyArn != nil && aws.ToString(key.KeyId) == keyID {
				keyARN := aws.ToString(key.KeyArn)
				keyMap[keyARN] = aliasName
				break
			}
		}
	}

	return keyMap, nil
}

// GetAllVPCs retrieves all VPCs in the region.
// Returns a map of VPC ID to VPC name.
func GetAllVPCs(ctx context.Context, cfg *aws.Config, region string) (map[string]string, error) {
	svc := ec2.NewFromConfig(*cfg, func(o *ec2.Options) {
		o.Region = region
	})

	result, err := svc.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to describe VPCs: %w", err)
	}

	vpcMap := make(map[string]string)
	for i := range result.Vpcs {
		vpc := &result.Vpcs[i]
		vpcID := aws.ToString(vpc.VpcId)
		name := GetTagValue(vpc.Tags, TagNameKey)
		if name == "" {
			name = vpcID
		}
		vpcMap[vpcID] = name
	}

	return vpcMap, nil
}

// GetAllSecurityGroups retrieves all security groups in the region.
// Returns a map of security group ID to security group name.
func GetAllSecurityGroups(ctx context.Context, cfg *aws.Config, region string) (map[string]string, error) {
	svc := ec2.NewFromConfig(*cfg, func(o *ec2.Options) {
		o.Region = region
	})

	result, err := svc.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to describe security groups: %w", err)
	}

	sgMap := make(map[string]string)
	for i := range result.SecurityGroups {
		sg := &result.SecurityGroups[i]
		sgID := aws.ToString(sg.GroupId)
		name := aws.ToString(sg.GroupName)
		if name == "" {
			name = sgID
		}
		sgMap[sgID] = name
	}

	return sgMap, nil
}

// GetAllSubnets retrieves all subnets in the region.
// Returns a map of subnet ID to subnet name.
func GetAllSubnets(ctx context.Context, cfg *aws.Config, region string) (map[string]string, error) {
	svc := ec2.NewFromConfig(*cfg, func(o *ec2.Options) {
		o.Region = region
	})

	result, err := svc.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to describe subnets: %w", err)
	}

	subnetMap := make(map[string]string)
	for i := range result.Subnets {
		subnet := &result.Subnets[i]
		subnetID := aws.ToString(subnet.SubnetId)
		name := GetTagValue(subnet.Tags, TagNameKey)
		if name == "" {
			name = subnetID
		}
		subnetMap[subnetID] = name
	}

	return subnetMap, nil
}

// GetAllImages retrieves all AMIs owned by the account in the region.
// Returns a map of image ID to image name.
func GetAllImages(ctx context.Context, cfg *aws.Config, region string) (map[string]string, error) {
	svc := ec2.NewFromConfig(*cfg, func(o *ec2.Options) {
		o.Region = region
	})

	result, err := svc.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Owners: []string{"self"}, // Only images owned by the account
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe images: %w", err)
	}

	imageMap := make(map[string]string)
	for i := range result.Images {
		image := &result.Images[i]
		imageID := aws.ToString(image.ImageId)
		name := aws.ToString(image.Name)
		if name == "" {
			name = GetTagValue(image.Tags, TagNameKey)
		}
		if name == "" {
			name = imageID
		}
		imageMap[imageID] = name
	}

	return imageMap, nil
}

// GetAllSnapshots retrieves all EBS snapshots owned by the account in the region.
// Returns a map of snapshot ID to snapshot name.
func GetAllSnapshots(ctx context.Context, cfg *aws.Config, region string) (map[string]string, error) {
	svc := ec2.NewFromConfig(*cfg, func(o *ec2.Options) {
		o.Region = region
	})

	result, err := svc.DescribeSnapshots(ctx, &ec2.DescribeSnapshotsInput{
		OwnerIds: []string{"self"}, // Only snapshots owned by the account
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe snapshots: %w", err)
	}

	snapshotMap := make(map[string]string)
	for i := range result.Snapshots {
		snapshot := &result.Snapshots[i]
		snapshotID := aws.ToString(snapshot.SnapshotId)
		name := GetTagValue(snapshot.Tags, TagNameKey)
		if name == "" {
			name = snapshotID
		}
		snapshotMap[snapshotID] = name
	}

	return snapshotMap, nil
}

// GetAllVolumes retrieves all EBS volumes in the region.
// Returns a map of volume ID to volume name.
func GetAllVolumes(ctx context.Context, cfg *aws.Config, region string) (map[string]string, error) {
	svc := ec2.NewFromConfig(*cfg, func(o *ec2.Options) {
		o.Region = region
	})

	result, err := svc.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to describe volumes: %w", err)
	}

	volumeMap := make(map[string]string)
	for i := range result.Volumes {
		volume := &result.Volumes[i]
		volumeID := aws.ToString(volume.VolumeId)
		name := GetTagValue(volume.Tags, TagNameKey)
		if name == "" {
			name = volumeID
		}
		volumeMap[volumeID] = name
	}

	return volumeMap, nil
}

// GetAllNetworkInterfaces retrieves all network interfaces in the region.
// Returns a map of network interface ID to network interface name.
func GetAllNetworkInterfaces(ctx context.Context, cfg *aws.Config, region string) (map[string]string, error) {
	svc := ec2.NewFromConfig(*cfg, func(o *ec2.Options) {
		o.Region = region
	})

	result, err := svc.DescribeNetworkInterfaces(ctx, &ec2.DescribeNetworkInterfacesInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to describe network interfaces: %w", err)
	}

	eniMap := make(map[string]string)
	for i := range result.NetworkInterfaces {
		eni := &result.NetworkInterfaces[i]
		eniID := aws.ToString(eni.NetworkInterfaceId)
		name := GetTagValue(eni.TagSet, TagNameKey)
		if name == "" {
			name = eniID
		}
		eniMap[eniID] = name
	}

	return eniMap, nil
}

// GetKMSName resolves a KMS key identifier to a human-friendly name.
// It handles both alias ARNs and key ARNs, attempting to find an alias for the key.
// This function is kept for backward compatibility but uses GetAllKMSKeys internally.
func GetKMSName(ctx context.Context, cfg *aws.Config, kmsIdentifier *string, region string) string {
	if kmsIdentifier == nil || aws.ToString(kmsIdentifier) == "" || aws.ToString(kmsIdentifier) == NotAvailable {
		return NotAvailable
	}

	identifier := aws.ToString(kmsIdentifier)

	// If this is an alias ARN, return the alias part
	if strings.Contains(identifier, ":alias/") {
		parts := strings.Split(identifier, ":")
		return parts[len(parts)-1]
	}

	// Extract key ID from ARN if needed
	keyID := identifier
	if strings.HasPrefix(identifier, "arn:aws:kms:") && strings.Contains(identifier, ":key/") {
		parts := strings.Split(identifier, "/")
		keyID = parts[len(parts)-1]
	}

	// Get all KMS keys and aliases
	keyMap, err := GetAllKMSKeys(ctx, cfg, region)
	if err != nil {
		return keyID
	}

	// Look up the key name
	if name, ok := keyMap[keyID]; ok {
		return name
	}

	// Return key ID as fallback
	return keyID
}

// GetSecurityGroupName resolves a security group ID to a human-friendly name.
func GetSecurityGroupName(ctx context.Context, cfg *aws.Config, sgID *string, region string) string {
	if sgID == nil || aws.ToString(sgID) == "" {
		return NotAvailable
	}

	sgIDStr := aws.ToString(sgID)

	// Only attempt to resolve well-formed SG IDs
	if !strings.HasPrefix(sgIDStr, "sg-") {
		return sgIDStr
	}

	// Get all security groups
	sgMap, err := GetAllSecurityGroups(ctx, cfg, region)
	if err != nil {
		return sgIDStr
	}

	// Look up the security group name
	if name, ok := sgMap[sgIDStr]; ok {
		return name
	}

	return sgIDStr
}

// GetSubnetName resolves a subnet ID to a human-friendly name.
func GetSubnetName(ctx context.Context, cfg *aws.Config, subnetID *string, region string) string {
	if subnetID == nil || aws.ToString(subnetID) == "" {
		return NotAvailable
	}

	subnetIDStr := aws.ToString(subnetID)

	if !strings.HasPrefix(subnetIDStr, "subnet-") {
		return subnetIDStr
	}

	// Get all subnets
	subnetMap, err := GetAllSubnets(ctx, cfg, region)
	if err != nil {
		return subnetIDStr
	}

	// Look up the subnet name
	if name, ok := subnetMap[subnetIDStr]; ok {
		return name
	}

	return subnetIDStr
}

// GetVPCName resolves a VPC ID to a human-friendly name.
func GetVPCName(ctx context.Context, cfg *aws.Config, vpcID *string, region string) string {
	if vpcID == nil || aws.ToString(vpcID) == "" {
		return NotAvailable
	}

	vpcIDStr := aws.ToString(vpcID)

	if !strings.HasPrefix(vpcIDStr, "vpc-") {
		return vpcIDStr
	}

	// Get all VPCs
	vpcMap, err := GetAllVPCs(ctx, cfg, region)
	if err != nil {
		return vpcIDStr
	}

	// Look up the VPC name
	if name, ok := vpcMap[vpcIDStr]; ok {
		return name
	}

	return vpcIDStr
}

// ResolveNameFromMap resolves an ID to a name using a pre-built map.
// If the ID is not found in the map, returns the ID itself.
func ResolveNameFromMap(id *string, nameMap map[string]string) string {
	idStr := StringValue(id)
	if name, ok := nameMap[idStr]; ok {
		return name
	}
	return idStr
}

// ResolveNamesFromMap resolves multiple IDs to names using a pre-built map.
// Returns a slice of resolved names.
// If an ID is not found in the map, uses the ID itself.
func ResolveNamesFromMap(ids []*string, nameMap map[string]string) []string {
	if len(ids) == 0 {
		return make([]string, 0)
	}

	names := make([]string, 0, len(ids))
	for _, id := range ids {
		idStr := StringValue(id)
		if name, ok := nameMap[idStr]; ok {
			names = append(names, name)
		} else {
			names = append(names, idStr)
		}
	}

	return names
}

// GetImageName resolves an image ID to a human-friendly name.
func GetImageName(ctx context.Context, cfg *aws.Config, imageID *string, region string) string {
	if imageID == nil || aws.ToString(imageID) == "" {
		return NotAvailable
	}

	imageIDStr := aws.ToString(imageID)

	if !strings.HasPrefix(imageIDStr, "ami-") {
		return imageIDStr
	}

	// Get all images
	imageMap, err := GetAllImages(ctx, cfg, region)
	if err != nil {
		return imageIDStr
	}

	// Look up the image name
	if name, ok := imageMap[imageIDStr]; ok {
		return name
	}

	return imageIDStr
}

// GetSnapshotName resolves a snapshot ID to a human-friendly name.
func GetSnapshotName(ctx context.Context, cfg *aws.Config, snapshotID *string, region string) string {
	if snapshotID == nil || aws.ToString(snapshotID) == "" {
		return NotAvailable
	}

	snapshotIDStr := aws.ToString(snapshotID)

	if !strings.HasPrefix(snapshotIDStr, "snap-") {
		return snapshotIDStr
	}

	// Get all snapshots
	snapshotMap, err := GetAllSnapshots(ctx, cfg, region)
	if err != nil {
		return snapshotIDStr
	}

	// Look up the snapshot name
	if name, ok := snapshotMap[snapshotIDStr]; ok {
		return name
	}

	return snapshotIDStr
}

// GetVolumeName resolves a volume ID to a human-friendly name.
func GetVolumeName(ctx context.Context, cfg *aws.Config, volumeID *string, region string) string {
	if volumeID == nil || aws.ToString(volumeID) == "" {
		return NotAvailable
	}

	volumeIDStr := aws.ToString(volumeID)

	if !strings.HasPrefix(volumeIDStr, "vol-") {
		return volumeIDStr
	}

	// Get all volumes
	volumeMap, err := GetAllVolumes(ctx, cfg, region)
	if err != nil {
		return volumeIDStr
	}

	// Look up the volume name
	if name, ok := volumeMap[volumeIDStr]; ok {
		return name
	}

	return volumeIDStr
}

// GetNetworkInterfaceName resolves a network interface ID to a human-friendly name.
func GetNetworkInterfaceName(ctx context.Context, cfg *aws.Config, eniID *string, region string) string {
	if eniID == nil || aws.ToString(eniID) == "" {
		return NotAvailable
	}

	eniIDStr := aws.ToString(eniID)

	if !strings.HasPrefix(eniIDStr, "eni-") {
		return eniIDStr
	}

	// Get all network interfaces
	eniMap, err := GetAllNetworkInterfaces(ctx, cfg, region)
	if err != nil {
		return eniIDStr
	}

	// Look up the network interface name
	if name, ok := eniMap[eniIDStr]; ok {
		return name
	}

	return eniIDStr
}

// ARN represents the components of an AWS ARN.
type ARN struct {
	Partition    string `json:"partition"`
	Service      string `json:"service"`
	Region       string `json:"region"`
	AccountID    string `json:"accountId"`
	ResourceType string `json:"resourceType"`
	Resource     string `json:"resource"`
}

// ParseARN parses an AWS ARN string into its components.
func ParseARN(arnStr string) (*ARN, error) {
	if !strings.HasPrefix(arnStr, "arn:") {
		return nil, ErrInvalidARNFormat
	}

	parts := strings.SplitN(arnStr, Colon, ARNPartCount)
	if len(parts) < ARNPartCount {
		return nil, ErrInvalidARNFormat
	}

	arn := &ARN{
		Partition: parts[ARNPartitionIndex],
		Service:   parts[ARNServiceIndex],
		Region:    parts[ARNRegionIndex],
		AccountID: parts[ARNAccountIndex],
		Resource:  parts[ARNResourceIndex],
	}

	// Parse resource type and resource name
	if strings.Contains(arn.Resource, "/") {
		resourceParts := strings.SplitN(arn.Resource, "/", ResourcePartCount)
		arn.ResourceType = resourceParts[0]
		arn.Resource = resourceParts[1]
	} else if strings.Contains(arn.Resource, Colon) {
		resourceParts := strings.SplitN(arn.Resource, Colon, ResourcePartCount)
		arn.ResourceType = resourceParts[0]
		arn.Resource = resourceParts[1]
	}

	return arn, nil
}

// GetResourceNameFromARN extracts the resource name from an ARN.
func GetResourceNameFromARN(arnStr string) string {
	arn, err := ParseARN(arnStr)
	if err != nil {
		return ""
	}

	return arn.Resource
}

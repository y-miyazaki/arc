// Package helpers provides helper functions for AWS resource collection.
package helpers

import (
	"context"
	"errors"
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

	keyMap, err := getAllKMSKeysWithClient(ctx, svc)
	if err != nil {
		return nil, fmt.Errorf("getAllKMSKeysWithClient: %w", err)
	}

	return keyMap, nil
}

// Package-level errors for client-type mismatches in test helpers.
var (
	ErrClientNotListKeys     = errors.New("client does not implement ListKeysAPIClient")
	ErrClientNotListAliases  = errors.New("client does not implement ListAliasesAPIClient")
	ErrClientNotDescribeVPCs = errors.New("client does not implement DescribeVpcsAPIClient")
)

// getAllKMSKeysWithClient collects KMS keys and aliases using the provided client.
// This helper exists so unit tests can inject a mock client that implements the
// KMS list APIs.
func getAllKMSKeysWithClient(ctx context.Context, client any) (map[string]string, error) {
	// We expect the client to implement both ListKeys and ListAliases APIs.
	// Use type assertion to pass the concrete client to the AWS paginator constructors.
	keysClient, ok := client.(kms.ListKeysAPIClient)
	if !ok {
		return nil, ErrClientNotListKeys
	}
	aliasesClient, ok := client.(kms.ListAliasesAPIClient)
	if !ok {
		return nil, ErrClientNotListAliases
	}

	// Collect all KMS keys to build ARNs
	keyARNs := make(map[string]string)
	keysPaginator := kms.NewListKeysPaginator(keysClient, &kms.ListKeysInput{})
	for keysPaginator.HasMorePages() {
		page, err := keysPaginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list KMS keys: %w", err)
		}
		for i := range page.Keys {
			k := &page.Keys[i]
			keyARNs[aws.ToString(k.KeyId)] = aws.ToString(k.KeyArn)
		}
	}

	// Collect aliases and build final mapping. We will keep alias names with the
	// "alias/" prefix for canonicalization and add mappings for keyID and keyARN.
	keyMap := make(map[string]string)
	aliasesPaginator := kms.NewListAliasesPaginator(aliasesClient, &kms.ListAliasesInput{})
	for aliasesPaginator.HasMorePages() {
		page, err := aliasesPaginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list KMS aliases: %w", err)
		}

		for j := range page.Aliases {
			alias := &page.Aliases[j]
			if alias.TargetKeyId == nil || alias.AliasName == nil {
				continue
			}
			keyID := aws.ToString(alias.TargetKeyId)
			aliasName := aws.ToString(alias.AliasName)

			// Ensure alias name includes the alias/ prefix (canonical form)
			if !strings.HasPrefix(aliasName, "alias/") {
				aliasName = "alias/" + aliasName
			}

			// Map key ID -> alias
			keyMap[keyID] = aliasName
			// Also map key ARN -> alias when available
			if arn, found := keyARNs[keyID]; found && arn != "" {
				keyMap[arn] = aliasName
			}

			// Make alias name resolvable directly to itself
			keyMap[aliasName] = aliasName
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

	vpcMap, err := getAllVPCsWithClient(ctx, svc)
	if err != nil {
		return nil, fmt.Errorf("getAllVPCsWithClient: %w", err)
	}

	return vpcMap, nil
}

// getAllVPCsWithClient collects VPCs using a provided EC2 client (testable helper).
func getAllVPCsWithClient(ctx context.Context, client any) (map[string]string, error) {
	cli, ok := client.(ec2.DescribeVpcsAPIClient)
	if !ok {
		return nil, ErrClientNotDescribeVPCs
	}

	paginator := ec2.NewDescribeVpcsPaginator(cli, &ec2.DescribeVpcsInput{})
	vpcMap := make(map[string]string)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe VPCs: %w", err)
		}
		for i := range page.Vpcs {
			vpc := &page.Vpcs[i]
			vpcID := aws.ToString(vpc.VpcId)
			name := GetTagValue(vpc.Tags, TagNameKey)
			if name == "" {
				name = vpcID
			}
			vpcMap[vpcID] = name
		}
	}

	return vpcMap, nil
}

// GetAllSecurityGroups retrieves all security groups in the region.
// Returns a map of security group ID to security group name.
func GetAllSecurityGroups(ctx context.Context, cfg *aws.Config, region string) (map[string]string, error) {
	svc := ec2.NewFromConfig(*cfg, func(o *ec2.Options) {
		o.Region = region
	})

	paginator := ec2.NewDescribeSecurityGroupsPaginator(svc, &ec2.DescribeSecurityGroupsInput{})
	sgMap := make(map[string]string)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe security groups: %w", err)
		}
		for i := range page.SecurityGroups {
			sg := &page.SecurityGroups[i]
			sgID := aws.ToString(sg.GroupId)
			name := aws.ToString(sg.GroupName)
			if name == "" {
				name = sgID
			}
			sgMap[sgID] = name
		}
	}

	return sgMap, nil
}

// GetAllSubnets retrieves all subnets in the region.
// Returns a map of subnet ID to subnet name.
func GetAllSubnets(ctx context.Context, cfg *aws.Config, region string) (map[string]string, error) {
	svc := ec2.NewFromConfig(*cfg, func(o *ec2.Options) {
		o.Region = region
	})

	paginator := ec2.NewDescribeSubnetsPaginator(svc, &ec2.DescribeSubnetsInput{})
	subnetMap := make(map[string]string)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe subnets: %w", err)
		}
		for i := range page.Subnets {
			subnet := &page.Subnets[i]
			subnetID := aws.ToString(subnet.SubnetId)
			name := GetTagValue(subnet.Tags, TagNameKey)
			if name == "" {
				name = subnetID
			}
			subnetMap[subnetID] = name
		}
	}

	return subnetMap, nil
}

// GetAllImages retrieves all AMIs owned by the account in the region.
// Returns a map of image ID to image name.
func GetAllImages(ctx context.Context, cfg *aws.Config, region string) (map[string]string, error) {
	svc := ec2.NewFromConfig(*cfg, func(o *ec2.Options) {
		o.Region = region
	})

	paginator := ec2.NewDescribeImagesPaginator(svc, &ec2.DescribeImagesInput{Owners: []string{"self"}})
	imageMap := make(map[string]string)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe images: %w", err)
		}
		for i := range page.Images {
			image := &page.Images[i]
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
	}

	return imageMap, nil
}

// GetAllSnapshots retrieves all EBS snapshots owned by the account in the region.
// Returns a map of snapshot ID to snapshot name.
func GetAllSnapshots(ctx context.Context, cfg *aws.Config, region string) (map[string]string, error) {
	svc := ec2.NewFromConfig(*cfg, func(o *ec2.Options) {
		o.Region = region
	})

	paginator := ec2.NewDescribeSnapshotsPaginator(svc, &ec2.DescribeSnapshotsInput{OwnerIds: []string{"self"}})
	snapshotMap := make(map[string]string)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe snapshots: %w", err)
		}
		for i := range page.Snapshots {
			snapshot := &page.Snapshots[i]
			snapshotID := aws.ToString(snapshot.SnapshotId)
			name := GetTagValue(snapshot.Tags, TagNameKey)
			if name == "" {
				name = snapshotID
			}
			snapshotMap[snapshotID] = name
		}
	}

	return snapshotMap, nil
}

// GetAllVolumes retrieves all EBS volumes in the region.
// Returns a map of volume ID to volume name.
func GetAllVolumes(ctx context.Context, cfg *aws.Config, region string) (map[string]string, error) {
	svc := ec2.NewFromConfig(*cfg, func(o *ec2.Options) {
		o.Region = region
	})

	paginator := ec2.NewDescribeVolumesPaginator(svc, &ec2.DescribeVolumesInput{})
	volumeMap := make(map[string]string)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe volumes: %w", err)
		}
		for i := range page.Volumes {
			volume := &page.Volumes[i]
			volumeID := aws.ToString(volume.VolumeId)
			name := GetTagValue(volume.Tags, TagNameKey)
			if name == "" {
				name = volumeID
			}
			volumeMap[volumeID] = name
		}
	}

	return volumeMap, nil
}

// GetAllNetworkInterfaces retrieves all network interfaces in the region.
// Returns a map of network interface ID to network interface name.
func GetAllNetworkInterfaces(ctx context.Context, cfg *aws.Config, region string) (map[string]string, error) {
	svc := ec2.NewFromConfig(*cfg, func(o *ec2.Options) {
		o.Region = region
	})

	paginator := ec2.NewDescribeNetworkInterfacesPaginator(svc, &ec2.DescribeNetworkInterfacesInput{})
	eniMap := make(map[string]string)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe network interfaces: %w", err)
		}
		for i := range page.NetworkInterfaces {
			eni := &page.NetworkInterfaces[i]
			eniID := aws.ToString(eni.NetworkInterfaceId)
			name := GetTagValue(eni.TagSet, TagNameKey)
			if name == "" {
				name = eniID
			}
			eniMap[eniID] = name
		}
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

	// If this is an alias ARN, return the alias part (keeps "alias/" prefix)
	if strings.Contains(identifier, ":alias/") {
		parts := strings.Split(identifier, ":")
		return parts[len(parts)-1]
	}

	// If the identifier is already an alias name (canonical form starting with "alias/"),
	// return it as-is to avoid making an AWS call.
	if strings.HasPrefix(identifier, "alias/") {
		return identifier
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

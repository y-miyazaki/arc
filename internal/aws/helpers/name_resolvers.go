// Package helpers provides helper functions for AWS resource collection.
package helpers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/kms"
)

const (
	// ARNPartitionIndex is the index of the partition in ARN parts.
	ARNPartitionIndex = 1
	// ARNRegionIndex is the index of the region in ARN parts.
	ARNRegionIndex = 3
	// ARNResourceIndex is the index of the resource in ARN parts.
	ARNResourceIndex = 5
	// ARNServiceIndex is the index of the service in ARN parts.
	ARNServiceIndex = 2
	// CloudFrontRegion is the AWS region for CloudFront global service.
	CloudFrontRegion = "us-east-1"
	// Colon is the colon character.
	Colon = ":"
	// ErrMsgClientRegionFmt is the error message format for missing region client.
	ErrMsgClientRegionFmt = "%w: %s"
	// NotAvailable represents a not available resource.
	NotAvailable = "N/A"
	// ResourcePartCount is the number of parts when splitting resource.
	ResourcePartCount = 2
	// TagNameKey is the standard AWS tag key for resource names.
	TagNameKey = "Name"
)

// Package-level errors for client-type mismatches in test helpers.
var (
	ErrClientNotDescribeImages            = errors.New("client does not implement DescribeImagesAPIClient")
	ErrClientNotDescribeNetworkInterfaces = errors.New("client does not implement DescribeNetworkInterfacesAPIClient")
	ErrClientNotDescribeSGs               = errors.New("client does not implement DescribeSecurityGroupsAPIClient")
	ErrClientNotDescribeSnapshots         = errors.New("client does not implement DescribeSnapshotsAPIClient")
	ErrClientNotDescribeSubnets           = errors.New("client does not implement DescribeSubnetsAPIClient")
	ErrClientNotDescribeVolumes           = errors.New("client does not implement DescribeVolumesAPIClient")
	ErrClientNotDescribeVPCs              = errors.New("client does not implement DescribeVpcsAPIClient")
	ErrClientNotListAliases               = errors.New("client does not implement ListAliasesAPIClient")
	ErrClientNotListKeys                  = errors.New("client does not implement ListKeysAPIClient")
	ErrNoEC2ClientForRegion               = errors.New("no EC2 client found for region")
	ErrNoKMSClientForRegion               = errors.New("no KMS client found for region")
	ErrNoCloudFrontClient                 = errors.New("no CloudFront client found")
)

// ARN represents the components of an AWS ARN.
type ARN struct {
	Partition    string `json:"partition"`
	Service      string `json:"service"`
	Region       string `json:"region"`
	AccountID    string `json:"accountId"`
	ResourceType string `json:"resourceType"`
	Resource     string `json:"resource"`
}

// NameResolver provides resource name resolution with caching.
// It holds pre-initialized AWS clients for multiple regions and caches resolved names
// to minimize API calls during resource collection.
type NameResolver struct {
	ec2Clients        map[string]*ec2.Client
	kmsClients        map[string]*kms.Client
	cloudfrontClients map[string]*cloudfront.Client
	cache             map[string]map[string]map[string]string // cache[region][resourceType] = map[id]name
	cloudfrontCache   map[string]string                       // cloudfrontCache[resourceType:id] = name
}

// NewNameResolver creates a new NameResolver with pre-initialized clients for all regions.
// This constructor follows dependency injection pattern by creating clients upfront.
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create clients for
//
// Returns:
//   - *NameResolver: Initialized resolver with regional clients and empty cache
//   - error: Error if client creation fails
func NewNameResolver(cfg *aws.Config, regions []string) (*NameResolver, error) {
	ec2Clients, err := CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *ec2.Client {
		return ec2.NewFromConfig(*c, func(o *ec2.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create EC2 clients: %w", err)
	}

	kmsClients, err := CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *kms.Client {
		return kms.NewFromConfig(*c, func(o *kms.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create KMS clients: %w", err)
	}

	// CloudFront is a global service, create clients for all regions (endpoint is us-east-1)
	//nolint:unused // region parameter unused as CloudFront is global (us-east-1 only)
	cloudfrontClients, err := CreateRegionalClients(cfg, regions, func(c *aws.Config, _ string) *cloudfront.Client {
		return cloudfront.NewFromConfig(*c, func(o *cloudfront.Options) {
			o.Region = CloudFrontRegion // CloudFront endpoints are in us-east-1
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudFront clients: %w", err)
	}

	return &NameResolver{
		ec2Clients:        ec2Clients,
		kmsClients:        kmsClients,
		cloudfrontClients: cloudfrontClients,
		cache:             make(map[string]map[string]map[string]string),
		cloudfrontCache:   make(map[string]string),
	}, nil
}

// GetAllKMSKeys retrieves all KMS keys and their aliases in the region with caching.
// Returns a map where both key ID and key ARN can be used as lookup keys to get the alias name.
// This allows lookups with either format: key ID (e.g., "12345678-1234-1234-1234-123456789012")
// or full ARN (e.g., "arn:aws:kms:region:account:key/12345678-1234-1234-1234-123456789012").
// Results are cached per region to minimize API calls.
func (nr *NameResolver) GetAllKMSKeys(ctx context.Context, region string) (map[string]string, error) {
	// Check cache first
	if nr.cache[region] != nil && nr.cache[region]["kms"] != nil {
		return nr.cache[region]["kms"], nil
	}

	svc, ok := nr.kmsClients[region]
	if !ok {
		return nil, fmt.Errorf(ErrMsgClientRegionFmt, ErrNoKMSClientForRegion, region)
	}

	keyMap, err := getAllKMSKeysWithClient(ctx, svc)
	if err != nil {
		return nil, fmt.Errorf("getAllKMSKeysWithClient: %w", err)
	}

	// Cache the result
	if nr.cache[region] == nil {
		nr.cache[region] = make(map[string]map[string]string)
	}
	nr.cache[region]["kms"] = keyMap

	return keyMap, nil
}

// GetAllImages retrieves all AMIs owned by the account in the region.
// Returns a map of image ID to image name.
// GetAllImages retrieves all AMIs owned by the account in the region with caching.
// Returns a map of image ID to image name.
// Results are cached per region to minimize API calls.
func (nr *NameResolver) GetAllImages(ctx context.Context, region string) (map[string]string, error) {
	// Check cache first
	if nr.cache[region] != nil && nr.cache[region]["images"] != nil {
		return nr.cache[region]["images"], nil
	}

	svc, ok := nr.ec2Clients[region]
	if !ok {
		return nil, fmt.Errorf(ErrMsgClientRegionFmt, ErrNoEC2ClientForRegion, region)
	}

	imageMap, err := getAllImagesWithClient(ctx, svc)
	if err != nil {
		return nil, fmt.Errorf("getAllImagesWithClient: %w", err)
	}

	// Cache the result
	if nr.cache[region] == nil {
		nr.cache[region] = make(map[string]map[string]string)
	}
	nr.cache[region]["images"] = imageMap

	return imageMap, nil
}

// GetAllNetworkInterfaces retrieves all network interfaces in the region with caching.
// Returns a map of network interface ID to network interface name.
// Results are cached per region to minimize API calls.
func (nr *NameResolver) GetAllNetworkInterfaces(ctx context.Context, region string) (map[string]string, error) {
	// Check cache first
	if nr.cache[region] != nil && nr.cache[region]["enis"] != nil {
		return nr.cache[region]["enis"], nil
	}

	svc, ok := nr.ec2Clients[region]
	if !ok {
		return nil, fmt.Errorf(ErrMsgClientRegionFmt, ErrNoEC2ClientForRegion, region)
	}

	eniMap, err := getAllNetworkInterfacesWithClient(ctx, svc)
	if err != nil {
		return nil, fmt.Errorf("getAllNetworkInterfacesWithClient: %w", err)
	}

	// Cache the result
	if nr.cache[region] == nil {
		nr.cache[region] = make(map[string]map[string]string)
	}
	nr.cache[region]["enis"] = eniMap

	return eniMap, nil
}

// GetAllSecurityGroups retrieves all security groups in the region with caching.
// Returns a map of security group ID to security group name.
// Results are cached per region to minimize API calls.
func (nr *NameResolver) GetAllSecurityGroups(ctx context.Context, region string) (map[string]string, error) {
	// Check cache first
	if nr.cache[region] != nil && nr.cache[region]["sgs"] != nil {
		return nr.cache[region]["sgs"], nil
	}

	svc, ok := nr.ec2Clients[region]
	if !ok {
		return nil, fmt.Errorf(ErrMsgClientRegionFmt, ErrNoEC2ClientForRegion, region)
	}

	sgMap, err := getAllSecurityGroupsWithClient(ctx, svc)
	if err != nil {
		return nil, fmt.Errorf("getAllSecurityGroupsWithClient: %w", err)
	}

	// Cache the result
	if nr.cache[region] == nil {
		nr.cache[region] = make(map[string]map[string]string)
	}
	nr.cache[region]["sgs"] = sgMap

	return sgMap, nil
}

// GetAllSnapshots retrieves all EBS snapshots owned by the account in the region with caching.
// Returns a map of snapshot ID to snapshot name.
// Results are cached per region to minimize API calls.
func (nr *NameResolver) GetAllSnapshots(ctx context.Context, region string) (map[string]string, error) {
	// Check cache first
	if nr.cache[region] != nil && nr.cache[region]["snapshots"] != nil {
		return nr.cache[region]["snapshots"], nil
	}

	svc, ok := nr.ec2Clients[region]
	if !ok {
		return nil, fmt.Errorf(ErrMsgClientRegionFmt, ErrNoEC2ClientForRegion, region)
	}

	snapshotMap, err := getAllSnapshotsWithClient(ctx, svc)
	if err != nil {
		return nil, fmt.Errorf("getAllSnapshotsWithClient: %w", err)
	}

	// Cache the result
	if nr.cache[region] == nil {
		nr.cache[region] = make(map[string]map[string]string)
	}
	nr.cache[region]["snapshots"] = snapshotMap

	return snapshotMap, nil
}

// GetAllSubnets retrieves all subnets in the region with caching.
// Returns a map of subnet ID to subnet name.
// Results are cached per region to minimize API calls.
func (nr *NameResolver) GetAllSubnets(ctx context.Context, region string) (map[string]string, error) {
	// Check cache first
	if nr.cache[region] != nil && nr.cache[region]["subnets"] != nil {
		return nr.cache[region]["subnets"], nil
	}

	svc, ok := nr.ec2Clients[region]
	if !ok {
		return nil, fmt.Errorf(ErrMsgClientRegionFmt, ErrNoEC2ClientForRegion, region)
	}

	subnetMap, err := getAllSubnetsWithClient(ctx, svc)
	if err != nil {
		return nil, fmt.Errorf("getAllSubnetsWithClient: %w", err)
	}

	// Cache the result
	if nr.cache[region] == nil {
		nr.cache[region] = make(map[string]map[string]string)
	}
	nr.cache[region]["subnets"] = subnetMap

	return subnetMap, nil
}

// GetAllVolumes retrieves all EBS volumes in the region with caching.
// Returns a map of volume ID to volume name.
// Results are cached per region to minimize API calls.
func (nr *NameResolver) GetAllVolumes(ctx context.Context, region string) (map[string]string, error) {
	// Check cache first
	if nr.cache[region] != nil && nr.cache[region]["volumes"] != nil {
		return nr.cache[region]["volumes"], nil
	}

	svc, ok := nr.ec2Clients[region]
	if !ok {
		return nil, fmt.Errorf(ErrMsgClientRegionFmt, ErrNoEC2ClientForRegion, region)
	}

	volumeMap, err := getAllVolumesWithClient(ctx, svc)
	if err != nil {
		return nil, fmt.Errorf("getAllVolumesWithClient: %w", err)
	}

	// Cache the result
	if nr.cache[region] == nil {
		nr.cache[region] = make(map[string]map[string]string)
	}
	nr.cache[region]["volumes"] = volumeMap

	return volumeMap, nil
}

// GetAllVPCs retrieves all VPCs in the region with caching.
// Returns a map of VPC ID to VPC name.
// Results are cached per region to minimize API calls.
func (nr *NameResolver) GetAllVPCs(ctx context.Context, region string) (map[string]string, error) {
	// Check cache first
	if nr.cache[region] != nil && nr.cache[region]["vpcs"] != nil {
		return nr.cache[region]["vpcs"], nil
	}

	svc, ok := nr.ec2Clients[region]
	if !ok {
		return nil, fmt.Errorf(ErrMsgClientRegionFmt, ErrNoEC2ClientForRegion, region)
	}

	vpcMap, err := getAllVPCsWithClient(ctx, svc)
	if err != nil {
		return nil, fmt.Errorf("getAllVPCsWithClient: %w", err)
	}

	// Cache the result
	if nr.cache[region] == nil {
		nr.cache[region] = make(map[string]map[string]string)
	}
	nr.cache[region]["vpcs"] = vpcMap

	return vpcMap, nil
}

// GetResourceNameFromARN extracts the resource name from an ARN.
func GetResourceNameFromARN(arnStr string) string {
	arn, err := ParseARN(arnStr)
	if err != nil {
		return ""
	}

	return arn.Resource
}

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

// ParseARN parses an AWS ARN string into its components.
func ParseARN(arnStr string) (*ARN, error) {
	if !strings.HasPrefix(arnStr, "arn:") {
		return nil, ErrInvalidARNFormat
	}

	parts := strings.SplitN(arnStr, Colon, ARNPartsCount)
	if len(parts) < ARNPartsCount {
		return nil, ErrInvalidARNFormat
	}

	arn := &ARN{
		Partition: parts[ARNPartitionIndex],
		Service:   parts[ARNServiceIndex],
		Region:    parts[ARNRegionIndex],
		AccountID: parts[ARNPartsAccountIndex],
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

// getAllImagesWithClient collects images via a provided EC2 client (testable helper).
func getAllImagesWithClient(ctx context.Context, client any) (map[string]string, error) {
	cli, ok := client.(ec2.DescribeImagesAPIClient)
	if !ok {
		return nil, ErrClientNotDescribeImages
	}

	paginator := ec2.NewDescribeImagesPaginator(cli, &ec2.DescribeImagesInput{Owners: []string{"self"}})
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

// (no-op) var block removed to avoid duplication.

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

// getAllNetworkInterfacesWithClient collects network interfaces via a provided EC2 client (testable helper).
func getAllNetworkInterfacesWithClient(ctx context.Context, client any) (map[string]string, error) {
	cli, ok := client.(ec2.DescribeNetworkInterfacesAPIClient)
	if !ok {
		return nil, ErrClientNotDescribeNetworkInterfaces
	}

	paginator := ec2.NewDescribeNetworkInterfacesPaginator(cli, &ec2.DescribeNetworkInterfacesInput{})
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

// getAllSecurityGroupsWithClient collects security groups via a provided EC2 client (testable helper).
func getAllSecurityGroupsWithClient(ctx context.Context, client any) (map[string]string, error) {
	cli, ok := client.(ec2.DescribeSecurityGroupsAPIClient)
	if !ok {
		return nil, ErrClientNotDescribeSGs
	}

	paginator := ec2.NewDescribeSecurityGroupsPaginator(cli, &ec2.DescribeSecurityGroupsInput{})
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

// getAllSnapshotsWithClient collects snapshots via a provided EC2 client (testable helper).
func getAllSnapshotsWithClient(ctx context.Context, client any) (map[string]string, error) {
	cli, ok := client.(ec2.DescribeSnapshotsAPIClient)
	if !ok {
		return nil, ErrClientNotDescribeSnapshots
	}

	paginator := ec2.NewDescribeSnapshotsPaginator(cli, &ec2.DescribeSnapshotsInput{OwnerIds: []string{"self"}})
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

// getAllSubnetsWithClient collects subnets via a provided EC2 client (testable helper).
func getAllSubnetsWithClient(ctx context.Context, client any) (map[string]string, error) {
	cli, ok := client.(ec2.DescribeSubnetsAPIClient)
	if !ok {
		return nil, ErrClientNotDescribeSubnets
	}

	paginator := ec2.NewDescribeSubnetsPaginator(cli, &ec2.DescribeSubnetsInput{})
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

// getAllVolumesWithClient collects volumes via a provided EC2 client (testable helper).
func getAllVolumesWithClient(ctx context.Context, client any) (map[string]string, error) {
	cli, ok := client.(ec2.DescribeVolumesAPIClient)
	if !ok {
		return nil, ErrClientNotDescribeVolumes
	}

	paginator := ec2.NewDescribeVolumesPaginator(cli, &ec2.DescribeVolumesInput{})
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

// GetOriginAccessControlName returns the name for a CloudFront Origin Access Control ID.
// Results are cached to minimize API calls. Caller must provide a context for cancellation/timeouts.
func (nr *NameResolver) GetOriginAccessControlName(ctx context.Context, oacID string) string {
	cacheKey := "oac:" + oacID
	if name, ok := nr.cloudfrontCache[cacheKey]; ok {
		return name
	}

	// CloudFront is a global service, use us-east-1 client
	client, ok := nr.cloudfrontClients[CloudFrontRegion]
	if !ok {
		return ""
	}

	output, err := client.GetOriginAccessControl(ctx, &cloudfront.GetOriginAccessControlInput{
		Id: aws.String(oacID),
	})
	if err != nil {
		return ""
	}

	name := ""
	if output.OriginAccessControl != nil && output.OriginAccessControl.OriginAccessControlConfig != nil {
		name = aws.ToString(output.OriginAccessControl.OriginAccessControlConfig.Name)
	}

	nr.cloudfrontCache[cacheKey] = name
	return name
}

// GetCachePolicyName returns the name for a CloudFront Cache Policy ID.
// Results are cached to minimize API calls. Caller must provide a context for cancellation/timeouts.
func (nr *NameResolver) GetCachePolicyName(ctx context.Context, policyID string) string {
	cacheKey := "cachepolicy:" + policyID
	if name, ok := nr.cloudfrontCache[cacheKey]; ok {
		return name
	}

	// CloudFront is a global service, use us-east-1 client
	client, ok := nr.cloudfrontClients[CloudFrontRegion]
	if !ok {
		return ""
	}

	output, err := client.GetCachePolicy(ctx, &cloudfront.GetCachePolicyInput{
		Id: aws.String(policyID),
	})
	if err != nil {
		return ""
	}

	name := ""
	if output.CachePolicy != nil && output.CachePolicy.CachePolicyConfig != nil {
		name = aws.ToString(output.CachePolicy.CachePolicyConfig.Name)
	}

	nr.cloudfrontCache[cacheKey] = name
	return name
}

// GetOriginRequestPolicyName returns the name for a CloudFront Origin Request Policy ID.
// Results are cached to minimize API calls. Caller must provide a context for cancellation/timeouts.
func (nr *NameResolver) GetOriginRequestPolicyName(ctx context.Context, policyID string) string {
	cacheKey := "originrequestpolicy:" + policyID
	if name, ok := nr.cloudfrontCache[cacheKey]; ok {
		return name
	}

	// CloudFront is a global service, use us-east-1 client
	client, ok := nr.cloudfrontClients[CloudFrontRegion]
	if !ok {
		return ""
	}

	output, err := client.GetOriginRequestPolicy(ctx, &cloudfront.GetOriginRequestPolicyInput{
		Id: aws.String(policyID),
	})
	if err != nil {
		return ""
	}

	name := ""
	if output.OriginRequestPolicy != nil && output.OriginRequestPolicy.OriginRequestPolicyConfig != nil {
		name = aws.ToString(output.OriginRequestPolicy.OriginRequestPolicyConfig.Name)
	}

	nr.cloudfrontCache[cacheKey] = name
	return name
}

// GetResponseHeadersPolicyName returns the name for a CloudFront Response Headers Policy ID.
// Results are cached to minimize API calls. Caller must provide a context for cancellation/timeouts.
func (nr *NameResolver) GetResponseHeadersPolicyName(ctx context.Context, policyID string) string {
	cacheKey := "responseheaderspolicy:" + policyID
	if name, ok := nr.cloudfrontCache[cacheKey]; ok {
		return name
	}

	// CloudFront is a global service, use us-east-1 client
	client, ok := nr.cloudfrontClients[CloudFrontRegion]
	if !ok {
		return ""
	}

	output, err := client.GetResponseHeadersPolicy(ctx, &cloudfront.GetResponseHeadersPolicyInput{
		Id: aws.String(policyID),
	})
	if err != nil {
		return ""
	}

	name := ""
	if output.ResponseHeadersPolicy != nil && output.ResponseHeadersPolicy.ResponseHeadersPolicyConfig != nil {
		name = aws.ToString(output.ResponseHeadersPolicy.ResponseHeadersPolicyConfig.Name)
	}

	nr.cloudfrontCache[cacheKey] = name
	return name
}

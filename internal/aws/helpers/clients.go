// Package helpers provides utility functions for AWS resource collectors.
package helpers

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/kms"
)

// KMSListKeysClientInterface wraps kms.ListKeys for pagination helpers.
type KMSListKeysClientInterface interface {
	// ListKeys lists KMS keys.
	ListKeys(_ context.Context, _ *kms.ListKeysInput, _ ...func(*kms.Options)) (*kms.ListKeysOutput, error)
}

// KMSListAliasesClientInterface wraps kms.ListAliases for pagination helpers.
type KMSListAliasesClientInterface interface {
	// ListAliases lists KMS aliases.
	ListAliases(_ context.Context, _ *kms.ListAliasesInput, _ ...func(*kms.Options)) (*kms.ListAliasesOutput, error)
}

// EC2DescribeImagesClientInterface wraps ec2.DescribeImages for pagination helpers.
type EC2DescribeImagesClientInterface interface {
	// DescribeImages describes EC2 images.
	DescribeImages(_ context.Context, _ *ec2.DescribeImagesInput, _ ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error)
}

// EC2DescribeNetworkInterfacesClientInterface wraps ec2.DescribeNetworkInterfaces for pagination helpers.
type EC2DescribeNetworkInterfacesClientInterface interface {
	// DescribeNetworkInterfaces describes EC2 network interfaces.
	DescribeNetworkInterfaces(_ context.Context, _ *ec2.DescribeNetworkInterfacesInput, _ ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error)
}

// EC2DescribeSecurityGroupsClientInterface wraps ec2.DescribeSecurityGroups for pagination helpers.
type EC2DescribeSecurityGroupsClientInterface interface {
	// DescribeSecurityGroups describes EC2 security groups.
	DescribeSecurityGroups(_ context.Context, _ *ec2.DescribeSecurityGroupsInput, _ ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error)
}

// EC2DescribeSnapshotsClientInterface wraps ec2.DescribeSnapshots for pagination helpers.
type EC2DescribeSnapshotsClientInterface interface {
	// DescribeSnapshots describes EC2 snapshots.
	DescribeSnapshots(_ context.Context, _ *ec2.DescribeSnapshotsInput, _ ...func(*ec2.Options)) (*ec2.DescribeSnapshotsOutput, error)
}

// EC2DescribeSubnetsClientInterface wraps ec2.DescribeSubnets for pagination helpers.
type EC2DescribeSubnetsClientInterface interface {
	// DescribeSubnets describes EC2 subnets.
	DescribeSubnets(_ context.Context, _ *ec2.DescribeSubnetsInput, _ ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error)
}

// EC2DescribeVolumesClientInterface wraps ec2.DescribeVolumes for pagination helpers.
type EC2DescribeVolumesClientInterface interface {
	// DescribeVolumes describes EC2 volumes.
	DescribeVolumes(_ context.Context, _ *ec2.DescribeVolumesInput, _ ...func(*ec2.Options)) (*ec2.DescribeVolumesOutput, error)
}

// EC2DescribeVpcsClientInterface wraps ec2.DescribeVpcs for pagination helpers.
type EC2DescribeVpcsClientInterface interface {
	// DescribeVpcs describes EC2 VPCs.
	DescribeVpcs(_ context.Context, _ *ec2.DescribeVpcsInput, _ ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)
}

// CreateRegionalClients creates AWS service clients for multiple regions using a factory function.
// This is a generic helper that works with any AWS service client type.
//
// Parameters:
//   - cfg: AWS configuration with credentials and base settings
//   - regions: List of AWS regions to create clients for
//   - factory: Function that creates a client for a specific region
//
// Returns:
//   - map[string]T: Map of region names to client instances
//   - error: Error if client creation fails for any region
//
// Example:
//
//	clients, err := helpers.CreateRegionalClients(cfg, regions, func(cfg *aws.Config, region string) *acm.Client {
//	    return acm.NewFromConfig(*cfg, func(o *acm.Options) {
//	        o.Region = region
//	    })
//	})
func CreateRegionalClients[T any](cfg *aws.Config, regions []string, factory func(*aws.Config, string) T) (map[string]T, error) {
	clients := make(map[string]T, len(regions))
	for _, region := range regions {
		client := factory(cfg, region)
		clients[region] = client
	}

	return clients, nil
}

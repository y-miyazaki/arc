// Package helpers provides utility functions for AWS resource collectors.
package helpers

import (
	"github.com/aws/aws-sdk-go-v2/aws"
)

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
// Example usage:
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

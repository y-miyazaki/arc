// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// SESCollector collects SES email identities and related resources.
// It uses dependency injection to manage SES clients for multiple regions.
type SESCollector struct {
	clients      map[string]*sesv2.Client
	nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewSESCollector creates a new SES collector with clients for the specified regions.
// Follows the same constructor pattern as other collectors in the package.
func NewSESCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*SESCollector, error) {
	clients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *sesv2.Client {
		return sesv2.NewFromConfig(*c, func(o *sesv2.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create SES clients: %w", err)
	}

	return &SESCollector{
		clients:      clients,
		nameResolver: nameResolver,
	}, nil
}

// Name returns the resource name of the collector.
func (*SESCollector) Name() string { return "ses" }

// ShouldSort returns whether the collected resources should be sorted.
func (*SESCollector) ShouldSort() bool { return false }

// GetColumns returns the CSV columns for the collector.
func (*SESCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory1", Value: func(r Resource) string { return r.SubCategory1 }},
		{Header: "SubCategory2", Value: func(r Resource) string { return r.SubCategory2 }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "IdentityType", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "IdentityType") }},
		{Header: "VerificationStatus", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "VerificationStatus") }},
		{Header: "DkimStatus", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "DkimStatus") }},
		{Header: "DkimTokens", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "DkimTokens") }},
		{Header: "MailFromDomain", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "MailFromDomain") }},
		{Header: "MailFromDomainStatus", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "MailFromDomainStatus") }},
		{Header: "BehaviorOnMXFailure", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "BehaviorOnMXFailure") }},
		{Header: "DefaultConfigurationSet", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "DefaultConfigurationSet") }},
		{Header: "SendingEnabled", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "SendingEnabled") }},
		{Header: "ReputationMetricsEnabled", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ReputationMetricsEnabled") }},
		{Header: "TrackingOptions", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "TrackingOptions") }},
		{Header: "DestinationType", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "DestinationType") }},
		{Header: "DestinationARN", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "DestinationARN") }},
		{Header: "EventTypes", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "EventTypes") }},
	}
}

// Collect collects SES resources for the specified region.
func (c *SESCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	svc, ok := c.clients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

	var resources []Resource

	// SES provides identities (email domains and addresses). We'll list identities
	// and attempt to fetch identity details for each identity. Use pagination.
	paginator := sesv2.NewListEmailIdentitiesPaginator(svc, &sesv2.ListEmailIdentitiesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list SES identities: %w", err)
		}

		for i := range page.EmailIdentities {
			identity := page.EmailIdentities[i]

			// Fetch identity details
			identityInput := &sesv2.GetEmailIdentityInput{EmailIdentity: identity.IdentityName}
			identityResp, identityErr := svc.GetEmailIdentity(ctx, identityInput)
			var verificationStatus, dkimStatus, mailFromDomain, mailFromDomainStatus, behaviorOnMXFailure, defaultConfigurationSet string
			var dkimTokens []string

			if identityErr == nil {
				// Verification status
				if identityResp.VerifiedForSendingStatus {
					verificationStatus = "Success"
				} else {
					verificationStatus = "Pending"
				}

				// DKIM attributes
				if identityResp.DkimAttributes != nil {
					dkimStatus = string(identityResp.DkimAttributes.Status)
					if len(identityResp.DkimAttributes.Tokens) > 0 {
						dkimTokens = identityResp.DkimAttributes.Tokens
					}
				}

				// Mail from attributes
				if identityResp.MailFromAttributes != nil {
					if identityResp.MailFromAttributes.MailFromDomain != nil {
						mailFromDomain = *identityResp.MailFromAttributes.MailFromDomain
					}
					mailFromDomainStatus = string(identityResp.MailFromAttributes.MailFromDomainStatus)
					behaviorOnMXFailure = string(identityResp.MailFromAttributes.BehaviorOnMxFailure)
				}

				// Configuration set
				if identityResp.ConfigurationSetName != nil {
					defaultConfigurationSet = *identityResp.ConfigurationSetName
				}
			}

			// Create identity resource
			identityResource := NewResource(&ResourceInput{
				Category:     "SES",
				SubCategory1: "Identity",
				Name:         *identity.IdentityName,
				Region:       region,
				ARN:          "",
				RawData: map[string]any{
					"IdentityType":            identity.IdentityType,
					"VerificationStatus":      verificationStatus,
					"DkimStatus":              dkimStatus,
					"DkimTokens":              dkimTokens,
					"MailFromDomain":          mailFromDomain,
					"MailFromDomainStatus":    mailFromDomainStatus,
					"BehaviorOnMXFailure":     behaviorOnMXFailure,
					"DefaultConfigurationSet": defaultConfigurationSet,
				},
			})
			resources = append(resources, identityResource)
		}
	}

	// Configuration Sets
	configResp, err := svc.ListConfigurationSets(ctx, &sesv2.ListConfigurationSetsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list SES configuration sets: %w", err)
	}

	for _, configSetName := range configResp.ConfigurationSets {
		// Get detailed configuration set information
		var trackingOptions string
		var sendingEnabled, reputationMetricsEnabled bool
		detailResp, detailErr := svc.GetConfigurationSet(ctx, &sesv2.GetConfigurationSetInput{
			ConfigurationSetName: &configSetName,
		})
		if detailErr == nil {
			if detailResp.SendingOptions != nil {
				sendingEnabled = detailResp.SendingOptions.SendingEnabled
			}
			if detailResp.ReputationOptions != nil {
				reputationMetricsEnabled = detailResp.ReputationOptions.ReputationMetricsEnabled
			}
			if detailResp.TrackingOptions != nil {
				trackingOptions = *detailResp.TrackingOptions.CustomRedirectDomain
			}

			// Add ConfigurationSet as a resource
			configSetResource := NewResource(&ResourceInput{
				Category:     "SES",
				SubCategory1: "ConfigurationSet",
				SubCategory2: "",
				Name:         configSetName,
				Region:       region,
				ARN:          "",
				RawData: map[string]any{
					"SendingEnabled":           sendingEnabled,
					"ReputationMetricsEnabled": reputationMetricsEnabled,
					"TrackingOptions":          trackingOptions,
				},
			})
			resources = append(resources, configSetResource)

			// Add EventDestinations as separate resources
			eventDestResp, eventDestErr := svc.GetConfigurationSetEventDestinations(ctx, &sesv2.GetConfigurationSetEventDestinationsInput{
				ConfigurationSetName: &configSetName,
			})
			if eventDestErr == nil && len(eventDestResp.EventDestinations) > 0 {
				for i := range eventDestResp.EventDestinations {
					dest := &eventDestResp.EventDestinations[i]
					if dest.Name != nil && dest.Enabled {
						var destinationType, destinationARN string

						// Determine destination type and ARN
						if dest.SnsDestination != nil && dest.SnsDestination.TopicArn != nil {
							destinationType = "SNS"
							destinationARN = *dest.SnsDestination.TopicArn
						} else if dest.CloudWatchDestination != nil {
							destinationType = "CloudWatch"
							destinationARN = "CloudWatch"
						} else if dest.KinesisFirehoseDestination != nil && dest.KinesisFirehoseDestination.DeliveryStreamArn != nil {
							destinationType = "KinesisFirehose"
							destinationARN = *dest.KinesisFirehoseDestination.DeliveryStreamArn
						}

						eventDestResource := NewResource(&ResourceInput{
							Category:     "SES",
							SubCategory1: "",
							SubCategory2: "EventDestination",
							Name:         dest.Name,
							Region:       region,
							ARN:          "",
							RawData: map[string]any{
								"ConfigurationSetName": configSetName,
								"DestinationType":      destinationType,
								"DestinationARN":       destinationARN,
								"EventTypes":           dest.MatchingEventTypes,
							},
						})
						resources = append(resources, eventDestResource)
					}
				}
			}
		}
	}

	return resources, nil
}

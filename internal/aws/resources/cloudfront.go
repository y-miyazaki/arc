// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

const (
	sepComma         = ","
	legacyCookiesFmt = "LegacyCookies=%s"
	kvFmt            = "%s=%s"
)

// CloudFrontCollector collects CloudFront resources.
// It uses dependency injection to manage CloudFront clients for multiple regions.
type CloudFrontCollector struct {
	clients      map[string]*cloudfront.Client
	nameResolver *helpers.NameResolver
}

// NewCloudFrontCollector creates a new CloudFront collector with clients for the specified regions.
// This constructor follows the standard naming convention for dependency injection:
// New<ServiceName>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*<ServiceName>Collector, error)
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create CloudFront clients for
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *CloudFrontCollector: Initialized collector with regional clients and name resolver
//   - error: Error if client creation fails
func NewCloudFrontCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*CloudFrontCollector, error) {
	clients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *cloudfront.Client {
		return cloudfront.NewFromConfig(*c, func(o *cloudfront.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudFront clients: %w", err)
	}

	return &CloudFrontCollector{
		clients:      clients,
		nameResolver: nameResolver,
	}, nil
}

// Name returns the resource name of the collector.
func (*CloudFrontCollector) Name() string {
	return "cloudfront"
}

// ShouldSort returns whether the collected resources should be sorted.
func (*CloudFrontCollector) ShouldSort() bool {
	return false
}

// GetColumns returns the CSV columns for the collector.
func (*CloudFrontCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory1", Value: func(r Resource) string { return r.SubCategory1 }},
		{Header: "SubCategory2", Value: func(r Resource) string { return r.SubCategory2 }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ID", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ID") }},
		{Header: "Description", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Description") }},
		{Header: "AlternateDomain", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AlternateDomain") }},
		{Header: "Origin", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Origin") }},
		{Header: "SSLCertificate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "SSLCertificate") }},
		{Header: "SecurityPolicy", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "SecurityPolicy") }},
		{Header: "SupportedHTTPVersions", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "SupportedHTTPVersions") }},
		{Header: "DefaultRootObject", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "DefaultRootObject") }},
		{Header: "PriceClass", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PriceClass") }},
		{Header: "WAF", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "WAF") }},
		{Header: "AccessLogDestinations", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AccessLogDestinations") }},
		{Header: "OriginId", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "OriginId") }},
		{Header: "DomainName", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "DomainName") }},
		{Header: "OriginPath", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "OriginPath") }},
		{Header: "OriginType", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "OriginType") }},
		{Header: "OriginAccessControlId", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "OriginAccessControlId") }},
		{Header: "OriginShield", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "OriginShield") }},
		{Header: "ConnectionTimeout", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ConnectionTimeout") }},
		{Header: "ResponseTimeout", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ResponseTimeout") }},
		{Header: "Config", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Config") }},
		{Header: "PathPattern", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PathPattern") }},
		{Header: "TargetOriginId", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "TargetOriginId") }},
		{Header: "ViewerProtocolPolicy", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ViewerProtocolPolicy") }},
		{Header: "CacheConfiguration", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CacheConfiguration") }},
		{Header: "SmoothStreaming", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "SmoothStreaming") }},
		{Header: "RealtimeLogConfig", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RealtimeLogConfig") }},
		{Header: "FunctionAssociations", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "FunctionAssociations") }},
		{Header: "Compress", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Compress") }},
		{Header: "HTTPErrorCode", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "HTTPErrorCode") }},
		{Header: "ErrorCachingMinTTL", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ErrorCachingMinTTL") }},
		{Header: "CustomizeErrorResponse", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CustomizeErrorResponse") }},
		{Header: "Status", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Status") }},
	}
}

// Collect collects CloudFront resources for the specified region.
// CloudFront is a global service, only process from us-east-1.
// The collector must have been initialized with a client for this region.
//
//nolint:revive // cyclomatic complexity is high due to many branches while collecting CloudFront resources
func (c *CloudFrontCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	// CloudFront is a global service, only process from us-east-1
	if region != "us-east-1" {
		return nil, nil
	}

	svc, ok := c.clients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

	var resources []Resource

	paginator := cloudfront.NewListDistributionsPaginator(svc, &cloudfront.ListDistributionsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list distributions: %w", err)
		}

		if page.DistributionList != nil {
			for i := range page.DistributionList.Items {
				dist := &page.DistributionList.Items[i]

				// Get full distribution details to access DistributionConfig
				getDistOutput, getDistErr := svc.GetDistribution(ctx, &cloudfront.GetDistributionInput{
					Id: dist.Id,
				})
				if getDistErr != nil {
					return nil, fmt.Errorf("failed to get distribution %s: %w", helpers.StringValue(dist.Id), getDistErr)
				}

				config := getDistOutput.Distribution.DistributionConfig

				// Aliases
				var aliases []string
				if config.Aliases != nil {
					aliases = config.Aliases.Items
				}

				// First Origin (for Distribution resource)
				var firstOrigin *string
				if config.Origins != nil && len(config.Origins.Items) > 0 {
					firstOrigin = config.Origins.Items[0].DomainName
				}

				// SSL Certificate
				var sslCert *string
				if config.ViewerCertificate != nil {
					if config.ViewerCertificate.ACMCertificateArn != nil {
						sslCert = config.ViewerCertificate.ACMCertificateArn
					} else if config.ViewerCertificate.IAMCertificateId != nil {
						sslCert = config.ViewerCertificate.IAMCertificateId
					} else if config.ViewerCertificate.CloudFrontDefaultCertificate != nil && *config.ViewerCertificate.CloudFrontDefaultCertificate {
						defaultCert := "CloudFront Default Certificate"
						sslCert = &defaultCert
					}
				}

				// Access Log Destinations
				var accessLogs *string
				if config.Logging != nil && config.Logging.Enabled != nil && *config.Logging.Enabled {
					logDest := helpers.StringValue(config.Logging.Bucket)
					if config.Logging.Prefix != nil && *config.Logging.Prefix != "" {
						logDest += "/" + *config.Logging.Prefix
					}
					accessLogs = &logDest
				}

				// WAF Name Resolution
				var waf *string
				if dist.WebACLId != nil && *dist.WebACLId != "" {
					wafARN := *dist.WebACLId
					// Try to extract the human-friendly name from WAFv2 ARN which contains "/webacl/<name>/"
					// Example: arn:aws:wafv2:us-east-1:123456789012:regional/webacl/MyWebACL/uuid
					if idx := strings.Index(wafARN, "/webacl/"); idx != -1 {
						// take the part after "/webacl/" and split by '/'
						sub := wafARN[idx+len("/webacl/"):]
						subParts := strings.SplitN(sub, "/", 2) //nolint:mnd
						if len(subParts) > 0 && subParts[0] != "" {
							wafName := subParts[0]
							waf = &wafName
						}
					}
					// Fallback: if we couldn't parse a name, try splitting on '/' and take the last segment
					if waf == nil {
						parts := strings.Split(wafARN, "/")
						if len(parts) > 0 {
							wafName := parts[len(parts)-1]
							waf = &wafName
						}
					}
				}

				// Add Distribution resource
				resources = append(resources, NewResource(&ResourceInput{
					Category:     "cloudfront",
					SubCategory1: "Distribution",
					SubCategory2: "",
					Name:         dist.DomainName,
					Region:       "Global",
					RawData: map[string]any{
						"ID":              dist.Id,
						"Description":     config.Comment,
						"AlternateDomain": aliases,
						"Origin":          firstOrigin,
						"SSLCertificate":  sslCert,
						"SecurityPolicy": func() any {
							if config.ViewerCertificate != nil {
								return config.ViewerCertificate.MinimumProtocolVersion
							}
							return nil
						}(),
						"SupportedHTTPVersions": config.HttpVersion,
						"DefaultRootObject":     config.DefaultRootObject,
						"PriceClass":            dist.PriceClass,
						"WAF":                   waf,
						"AccessLogDestinations": accessLogs,
						"Status":                dist.Status,
					},
				}))

				// Add Origin resources (SubCategory2="Origin")
				if config.Origins != nil {
					for i := range config.Origins.Items {
						origin := &config.Origins.Items[i]
						originType := "Custom"
						var configParts []string

						// Origin type and basic config
						if origin.S3OriginConfig != nil {
							originType = "S3"
							if origin.S3OriginConfig.OriginAccessIdentity != nil && *origin.S3OriginConfig.OriginAccessIdentity != "" {
								configParts = append(configParts, fmt.Sprintf("OAI=%s", *origin.S3OriginConfig.OriginAccessIdentity))
							}
						} else if origin.CustomOriginConfig != nil {
							configParts = append(configParts,
								fmt.Sprintf("HTTP=%d", aws.ToInt32(origin.CustomOriginConfig.HTTPPort)),
								fmt.Sprintf("HTTPS=%d", aws.ToInt32(origin.CustomOriginConfig.HTTPSPort)),
								fmt.Sprintf("Protocol=%s", origin.CustomOriginConfig.OriginProtocolPolicy),
							)
						}

						// Origin Access Control (OAC) - replaces OAI
						var originAccessControlID *string
						if origin.OriginAccessControlId != nil && *origin.OriginAccessControlId != "" {
							oacID := *origin.OriginAccessControlId
							// Try to resolve OAC name
							oacName := c.nameResolver.GetOriginAccessControlName(ctx, oacID)
							if oacName != "" {
								oacDisplay := fmt.Sprintf("%s (%s)", oacID, oacName)
								originAccessControlID = &oacDisplay
								configParts = append(configParts, fmt.Sprintf("OAC=%s(%s)", oacID, oacName))
							} else {
								originAccessControlID = &oacID
								configParts = append(configParts, fmt.Sprintf("OAC=%s", oacID))
							}
						}

						// Origin Shield
						var originShield *string
						if origin.OriginShield != nil && origin.OriginShield.Enabled != nil && *origin.OriginShield.Enabled {
							if origin.OriginShield.OriginShieldRegion != nil {
								shieldInfo := fmt.Sprintf("Enabled Region=%s", *origin.OriginShield.OriginShieldRegion)
								originShield = &shieldInfo
								configParts = append(configParts, fmt.Sprintf("OriginShield=%s", *origin.OriginShield.OriginShieldRegion))
							}
						}

						// Connection and Response Timeouts
						var connectionTimeout *int32
						var responseTimeout *int32
						if origin.CustomOriginConfig != nil {
							if origin.CustomOriginConfig.OriginReadTimeout != nil {
								responseTimeout = origin.CustomOriginConfig.OriginReadTimeout
								configParts = append(configParts, fmt.Sprintf("ResponseTimeout=%ds", *origin.CustomOriginConfig.OriginReadTimeout))
							}
							if origin.CustomOriginConfig.OriginKeepaliveTimeout != nil {
								connectionTimeout = origin.CustomOriginConfig.OriginKeepaliveTimeout
								configParts = append(configParts, fmt.Sprintf("ConnectionTimeout=%ds", *origin.CustomOriginConfig.OriginKeepaliveTimeout))
							}
						}

						originConfig := strings.Join(configParts, " ")

						resources = append(resources, NewResource(&ResourceInput{
							Category:     "cloudfront",
							SubCategory1: "",
							SubCategory2: "Origin",
							Name:         dist.DomainName,
							Region:       "Global",
							RawData: map[string]any{
								"ID":                    dist.Id,
								"OriginId":              origin.Id,
								"DomainName":            origin.DomainName,
								"OriginPath":            origin.OriginPath,
								"OriginType":            &originType,
								"OriginAccessControlId": originAccessControlID,
								"OriginShield":          originShield,
								"ConnectionTimeout":     connectionTimeout,
								"ResponseTimeout":       responseTimeout,
								"Config":                &originConfig,
							},
						}))
					}
				}

				// Add Behavior resources (SubCategory2="Behavior")
				// Default cache behavior
				if config.DefaultCacheBehavior != nil {
					behavior := config.DefaultCacheBehavior

					// Build cache configuration as key=value string array
					var cacheConfig []string

					// Cache policies (recommended)
					if behavior.CachePolicyId != nil {
						cachePolicyID := *behavior.CachePolicyId
						if cachePolicyName := c.nameResolver.GetCachePolicyName(ctx, cachePolicyID); cachePolicyName != "" {
							cacheConfig = append(cacheConfig, fmt.Sprintf("CachePolicy=%s", cachePolicyName))
						} else {
							cacheConfig = append(cacheConfig, fmt.Sprintf("CachePolicy=%s", cachePolicyID))
						}
					}
					if behavior.OriginRequestPolicyId != nil {
						originReqPolicyID := *behavior.OriginRequestPolicyId
						if originReqPolicyName := c.nameResolver.GetOriginRequestPolicyName(ctx, originReqPolicyID); originReqPolicyName != "" {
							cacheConfig = append(cacheConfig, fmt.Sprintf("OriginRequestPolicy=%s", originReqPolicyName))
						} else {
							cacheConfig = append(cacheConfig, fmt.Sprintf("OriginRequestPolicy=%s", originReqPolicyID))
						}
					}
					if behavior.ResponseHeadersPolicyId != nil {
						respHeadersPolicyID := *behavior.ResponseHeadersPolicyId
						if respHeadersPolicyName := c.nameResolver.GetResponseHeadersPolicyName(ctx, respHeadersPolicyID); respHeadersPolicyName != "" {
							cacheConfig = append(cacheConfig, fmt.Sprintf("ResponseHeadersPolicy=%s", respHeadersPolicyName))
						} else {
							cacheConfig = append(cacheConfig, fmt.Sprintf("ResponseHeadersPolicy=%s", respHeadersPolicyID))
						}
					}

					//nolint:staticcheck // Legacy cache settings (ForwardedValues is deprecated but still supported)
					if behavior.ForwardedValues != nil {
						// Headers
						if behavior.ForwardedValues.Headers != nil && len(behavior.ForwardedValues.Headers.Items) > 0 {
							headersList := strings.Join(behavior.ForwardedValues.Headers.Items, sepComma)
							cacheConfig = append(cacheConfig, fmt.Sprintf("LegacyHeaders=%s", headersList))
						}
						// Query Strings
						if behavior.ForwardedValues.QueryString != nil && *behavior.ForwardedValues.QueryString {
							if behavior.ForwardedValues.QueryStringCacheKeys != nil && len(behavior.ForwardedValues.QueryStringCacheKeys.Items) > 0 {
								qsList := strings.Join(behavior.ForwardedValues.QueryStringCacheKeys.Items, sepComma)
								cacheConfig = append(cacheConfig, fmt.Sprintf("LegacyQueryStrings=%s", qsList))
							} else {
								cacheConfig = append(cacheConfig, "LegacyQueryStrings=all")
							}
						}
						// Cookies
						if behavior.ForwardedValues.Cookies != nil && behavior.ForwardedValues.Cookies.Forward != "" {
							if behavior.ForwardedValues.Cookies.Forward == "whitelist" && behavior.ForwardedValues.Cookies.WhitelistedNames != nil && len(behavior.ForwardedValues.Cookies.WhitelistedNames.Items) > 0 {
								cookiesList := strings.Join(behavior.ForwardedValues.Cookies.WhitelistedNames.Items, sepComma)
								cacheConfig = append(cacheConfig, fmt.Sprintf(legacyCookiesFmt, cookiesList))
							} else {
								cacheConfig = append(cacheConfig, fmt.Sprintf(legacyCookiesFmt, string(behavior.ForwardedValues.Cookies.Forward)))
							}
						}
					}

					// Function Associations (key=value format)
					var functionAssociations []string
					// CloudFront Functions
					if behavior.FunctionAssociations != nil && len(behavior.FunctionAssociations.Items) > 0 {
						for i := range behavior.FunctionAssociations.Items {
							fa := &behavior.FunctionAssociations.Items[i]
							if fa.FunctionARN != nil && fa.EventType != "" {
								functionAssociations = append(functionAssociations, fmt.Sprintf(kvFmt, fa.EventType, *fa.FunctionARN))
							}
						}
					}
					// Lambda@Edge
					if behavior.LambdaFunctionAssociations != nil && len(behavior.LambdaFunctionAssociations.Items) > 0 {
						for i := range behavior.LambdaFunctionAssociations.Items {
							lfa := &behavior.LambdaFunctionAssociations.Items[i]
							if lfa.LambdaFunctionARN != nil && lfa.EventType != "" {
								functionAssociations = append(functionAssociations, fmt.Sprintf(kvFmt, lfa.EventType, *lfa.LambdaFunctionARN))
							}
						}
					}

					resources = append(resources, NewResource(&ResourceInput{
						Category:     "cloudfront",
						SubCategory1: "",
						SubCategory2: "Behavior",
						Name:         dist.DomainName,
						Region:       "Global",
						RawData: map[string]any{
							"ID":                   dist.Id,
							"PathPattern":          "Default (*)",
							"TargetOriginId":       behavior.TargetOriginId,
							"ViewerProtocolPolicy": behavior.ViewerProtocolPolicy,
							"CacheConfiguration":   cacheConfig,
							"SmoothStreaming":      behavior.SmoothStreaming,
							"RealtimeLogConfig":    behavior.RealtimeLogConfigArn,
							"FunctionAssociations": functionAssociations,
							"Compress":             behavior.Compress,
						},
					}))
				}

				// Additional cache behaviors
				if config.CacheBehaviors != nil {
					for i := range config.CacheBehaviors.Items {
						behavior := &config.CacheBehaviors.Items[i]

						// Build cache configuration as key=value string array
						var cacheConfig []string

						// Cache policies (recommended)
						if behavior.CachePolicyId != nil {
							cachePolicyID := *behavior.CachePolicyId
							if cachePolicyName := c.nameResolver.GetCachePolicyName(ctx, cachePolicyID); cachePolicyName != "" {
								cacheConfig = append(cacheConfig, fmt.Sprintf("CachePolicy=%s(%s)", cachePolicyID, cachePolicyName))
							} else {
								cacheConfig = append(cacheConfig, fmt.Sprintf("CachePolicy=%s", cachePolicyID))
							}
						}
						if behavior.OriginRequestPolicyId != nil {
							originReqPolicyID := *behavior.OriginRequestPolicyId
							if originReqPolicyName := c.nameResolver.GetOriginRequestPolicyName(ctx, originReqPolicyID); originReqPolicyName != "" {
								cacheConfig = append(cacheConfig, fmt.Sprintf("OriginRequestPolicy=%s(%s)", originReqPolicyID, originReqPolicyName))
							} else {
								cacheConfig = append(cacheConfig, fmt.Sprintf("OriginRequestPolicy=%s", originReqPolicyID))
							}
						}
						if behavior.ResponseHeadersPolicyId != nil {
							respHeadersPolicyID := *behavior.ResponseHeadersPolicyId
							if respHeadersPolicyName := c.nameResolver.GetResponseHeadersPolicyName(ctx, respHeadersPolicyID); respHeadersPolicyName != "" {
								cacheConfig = append(cacheConfig, fmt.Sprintf("ResponseHeadersPolicy=%s(%s)", respHeadersPolicyID, respHeadersPolicyName))
							} else {
								cacheConfig = append(cacheConfig, fmt.Sprintf("ResponseHeadersPolicy=%s", respHeadersPolicyID))
							}
						}

						//nolint:staticcheck // Legacy cache settings (ForwardedValues is deprecated but still supported)
						if behavior.ForwardedValues != nil {
							// Headers
							if behavior.ForwardedValues.Headers != nil && len(behavior.ForwardedValues.Headers.Items) > 0 {
								headersList := strings.Join(behavior.ForwardedValues.Headers.Items, sepComma)
								cacheConfig = append(cacheConfig, fmt.Sprintf("LegacyHeaders=%s", headersList))
							}
							// Query Strings
							if behavior.ForwardedValues.QueryString != nil && *behavior.ForwardedValues.QueryString {
								if behavior.ForwardedValues.QueryStringCacheKeys != nil && len(behavior.ForwardedValues.QueryStringCacheKeys.Items) > 0 {
									qsList := strings.Join(behavior.ForwardedValues.QueryStringCacheKeys.Items, sepComma)
									cacheConfig = append(cacheConfig, fmt.Sprintf("LegacyQueryStrings=%s", qsList))
								} else {
									cacheConfig = append(cacheConfig, "LegacyQueryStrings=all")
								}
							}
							// Cookies
							if behavior.ForwardedValues.Cookies != nil && behavior.ForwardedValues.Cookies.Forward != "" {
								if behavior.ForwardedValues.Cookies.Forward == "whitelist" && behavior.ForwardedValues.Cookies.WhitelistedNames != nil && len(behavior.ForwardedValues.Cookies.WhitelistedNames.Items) > 0 {
									cookiesList := strings.Join(behavior.ForwardedValues.Cookies.WhitelistedNames.Items, sepComma)
									cacheConfig = append(cacheConfig, fmt.Sprintf(legacyCookiesFmt, cookiesList))
								} else {
									cacheConfig = append(cacheConfig, fmt.Sprintf("LegacyCookies=%s", string(behavior.ForwardedValues.Cookies.Forward)))
								}
							}
						}

						// Function Associations (key=value format)
						var functionAssociations []string
						// CloudFront Functions
						if behavior.FunctionAssociations != nil && len(behavior.FunctionAssociations.Items) > 0 {
							for i := range behavior.FunctionAssociations.Items {
								fa := &behavior.FunctionAssociations.Items[i]
								if fa.FunctionARN != nil && fa.EventType != "" {
									functionAssociations = append(functionAssociations, fmt.Sprintf(kvFmt, fa.EventType, *fa.FunctionARN))
								}
							}
						}
						// Lambda@Edge
						if behavior.LambdaFunctionAssociations != nil && len(behavior.LambdaFunctionAssociations.Items) > 0 {
							for i := range behavior.LambdaFunctionAssociations.Items {
								lfa := &behavior.LambdaFunctionAssociations.Items[i]
								if lfa.LambdaFunctionARN != nil && lfa.EventType != "" {
									functionAssociations = append(functionAssociations, fmt.Sprintf(kvFmt, lfa.EventType, *lfa.LambdaFunctionARN))
								}
							}
						}

						resources = append(resources, NewResource(&ResourceInput{
							Category:     "cloudfront",
							SubCategory1: "",
							SubCategory2: "Behavior",
							Name:         dist.DomainName,
							Region:       "Global",
							RawData: map[string]any{
								"ID":                   dist.Id,
								"PathPattern":          behavior.PathPattern,
								"TargetOriginId":       behavior.TargetOriginId,
								"ViewerProtocolPolicy": behavior.ViewerProtocolPolicy,
								"CacheConfiguration":   cacheConfig,
								"SmoothStreaming":      behavior.SmoothStreaming,
								"RealtimeLogConfig":    behavior.RealtimeLogConfigArn,
								"FunctionAssociations": functionAssociations,
								"Compress":             behavior.Compress,
							},
						}))
					}
				}

				// Add ErrorPage resources (SubCategory2="ErrorPage")
				if config.CustomErrorResponses != nil && len(config.CustomErrorResponses.Items) > 0 {
					for i := range config.CustomErrorResponses.Items {
						errorResponse := &config.CustomErrorResponses.Items[i]

						// Customize error response info
						var customizeErrorResponse *string
						if errorResponse.ResponseCode != nil && *errorResponse.ResponseCode != "" {
							customInfo := fmt.Sprintf("ResponseCode=%s", *errorResponse.ResponseCode)
							if errorResponse.ResponsePagePath != nil && *errorResponse.ResponsePagePath != "" {
								customInfo += fmt.Sprintf(" ResponsePagePath=%s", *errorResponse.ResponsePagePath)
							}
							customizeErrorResponse = &customInfo
						}

						resources = append(resources, NewResource(&ResourceInput{
							Category:     "cloudfront",
							SubCategory1: "",
							SubCategory2: "ErrorPage",
							Name:         dist.DomainName,
							Region:       "Global",
							RawData: map[string]any{
								"ID":                     dist.Id,
								"HTTPErrorCode":          errorResponse.ErrorCode,
								"ErrorCachingMinTTL":     errorResponse.ErrorCachingMinTTL,
								"CustomizeErrorResponse": customizeErrorResponse,
							},
						}))
					}
				}
			}
		}
	}

	return resources, nil
}

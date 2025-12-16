// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/batch"
	"github.com/aws/aws-sdk-go-v2/service/batch/types"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// Sentinel errors for Batch operations.
var (
	ErrNoBatchClient = errors.New("no Batch client found for region")
)

// BatchCollector collects AWS Batch resources (Job Queues, Compute Environments, Job Definitions).
type BatchCollector struct {
	clients      map[string]*batch.Client
	nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewBatchCollector creates a new Batch collector with regional clients.
func NewBatchCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*BatchCollector, error) {
	clients, err := helpers.CreateRegionalClients(cfg, regions,
		func(cfg *aws.Config, region string) *batch.Client {
			return batch.NewFromConfig(*cfg, func(o *batch.Options) {
				o.Region = region
			})
		})
	if err != nil {
		return nil, fmt.Errorf("failed to create Batch clients: %w", err)
	}

	return &BatchCollector{
		clients:      clients,
		nameResolver: nameResolver,
	}, nil
}

// Name returns the collector name.
func (*BatchCollector) Name() string {
	return "batch"
}

// ShouldSort returns true.
func (*BatchCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV column definitions for Batch.
func (*BatchCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory1", Value: func(r Resource) string { return r.SubCategory1 }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "Priority", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Priority") }},
		{Header: "Type", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Type") }},
		{Header: "JobRoleArn", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "JobRoleArn") }},
		{Header: "ExecutionRoleArn", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ExecutionRoleArn") }},
		{Header: "Image", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Image") }},
		{Header: "vCPU", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "vCPU") }},
		{Header: "Memory", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Memory") }},
		{Header: "CpuArchitecture", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CpuArchitecture") }},
		{Header: "OperatingSystemFamily", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "OperatingSystemFamily") }},
		{Header: "Timeout", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Timeout") }},
		{Header: "JSON", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "JSON") }},
		{Header: "Status", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Status") }},
	}
}

// Collect collects Batch resources from the specified region.
func (c *BatchCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	svc, ok := c.clients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoBatchClient, region)
	}

	var resources []Resource

	// Describe Job Queues
	jqPaginator := batch.NewDescribeJobQueuesPaginator(svc, &batch.DescribeJobQueuesInput{})
	for jqPaginator.HasMorePages() {
		page, err := jqPaginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe job queues: %w", err)
		}

		for i := range page.JobQueues {
			jq := &page.JobQueues[i]

			resources = append(resources, NewResource(&ResourceInput{
				Category:     "batch",
				SubCategory1: "JobQueue",
				Name:         jq.JobQueueName,
				Region:       region,
				ARN:          jq.JobQueueArn,
				RawData: map[string]any{
					"Priority": jq.Priority,
					"Status":   jq.State,
					"JSON":     helpers.FormatJSONIndentOrRaw(jq),
				},
			}))
		}
	}

	// Describe Compute Environments
	cePaginator := batch.NewDescribeComputeEnvironmentsPaginator(svc, &batch.DescribeComputeEnvironmentsInput{})
	for cePaginator.HasMorePages() {
		page, err := cePaginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe compute environments: %w", err)
		}

		for i := range page.ComputeEnvironments {
			ce := &page.ComputeEnvironments[i]

			resources = append(resources, NewResource(&ResourceInput{
				Category:     "batch",
				SubCategory1: "ComputeEnvironment",
				Name:         ce.ComputeEnvironmentName,
				Region:       region,
				ARN:          ce.ComputeEnvironmentArn,
				RawData: map[string]any{
					"Type":   ce.Type,
					"Status": ce.State,
					"JSON":   helpers.FormatJSONIndentOrRaw(ce),
				},
			}))
		}
	}

	// Describe Job Definitions
	jdPaginator := batch.NewDescribeJobDefinitionsPaginator(svc, &batch.DescribeJobDefinitionsInput{
		Status: aws.String("ACTIVE"),
	})

	// Map to store latest revision per job definition name
	latestRevisions := make(map[string]*types.JobDefinition)

	for jdPaginator.HasMorePages() {
		page, err := jdPaginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe job definitions: %w", err)
		}

		for i := range page.JobDefinitions {
			jd := &page.JobDefinitions[i]
			name := aws.ToString(jd.JobDefinitionName)
			if existing, ok2 := latestRevisions[name]; ok2 {
				if aws.ToInt32(jd.Revision) > aws.ToInt32(existing.Revision) {
					latestRevisions[name] = jd
				}
			} else {
				latestRevisions[name] = jd
			}
		}
	}

	for _, jd := range latestRevisions {
		var image, vcpu, memory, cpuArch, osFamily, timeout string
		if jd.ContainerProperties != nil {
			image = aws.ToString(jd.ContainerProperties.Image)

			// Try ResourceRequirements first
			for i := range jd.ContainerProperties.ResourceRequirements {
				req := &jd.ContainerProperties.ResourceRequirements[i]
				if req.Type == types.ResourceTypeVcpu {
					vcpu = aws.ToString(req.Value)
				}
				if req.Type == types.ResourceTypeMemory {
					memory = aws.ToString(req.Value)
				}
			}

			// Fallback to legacy fields
			if vcpu == "" && jd.ContainerProperties.Vcpus != nil { //nolint:staticcheck
				vcpu = strconv.Itoa(int(*jd.ContainerProperties.Vcpus)) //nolint:staticcheck
			}
			if memory == "" && jd.ContainerProperties.Memory != nil { //nolint:staticcheck
				memory = strconv.Itoa(int(*jd.ContainerProperties.Memory)) //nolint:staticcheck
			}

			// New fields
			if jd.ContainerProperties.RuntimePlatform != nil {
				if jd.ContainerProperties.RuntimePlatform.CpuArchitecture != nil {
					cpuArch = aws.ToString(jd.ContainerProperties.RuntimePlatform.CpuArchitecture)
				}
				if jd.ContainerProperties.RuntimePlatform.OperatingSystemFamily != nil {
					osFamily = aws.ToString(jd.ContainerProperties.RuntimePlatform.OperatingSystemFamily)
				}
			}
		}

		// Timeout
		if jd.Timeout != nil && jd.Timeout.AttemptDurationSeconds != nil {
			timeout = strconv.Itoa(int(*jd.Timeout.AttemptDurationSeconds))
		}

		// JSON representation

		nameWithRev := fmt.Sprintf("%s:%d", aws.ToString(jd.JobDefinitionName), jd.Revision)
		resources = append(resources, NewResource(&ResourceInput{
			Category:     "batch",
			SubCategory1: "JobDefinition",
			Name:         nameWithRev,
			Region:       region,
			ARN:          jd.JobDefinitionArn,
			RawData: map[string]any{
				"Type":                  jd.Type,
				"JobRoleArn":            jd.ContainerProperties.JobRoleArn,
				"ExecutionRoleArn":      jd.ContainerProperties.ExecutionRoleArn,
				"Image":                 image,
				"vCPU":                  vcpu,
				"Memory":                memory,
				"CpuArchitecture":       cpuArch,
				"OperatingSystemFamily": osFamily,
				"Timeout":               timeout,
				"JSON":                  helpers.FormatJSONIndentOrRaw(jd),
				"Status":                jd.Status,
			},
		}))
	}

	return resources, nil
}

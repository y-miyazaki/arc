// Package resources provides AWS resource collectors for different services.
// This package contains collectors that gather information about various AWS resources
// such as ECS clusters, services, task definitions, and scheduled tasks.
package resources

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	ebtypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

const (
	// MaxServicesPerDescribe is the maximum number of services to describe in a single call
	MaxServicesPerDescribe = 10
	// MinARNParts is the minimum number of parts in a task definition ARN
	MinARNParts = 2
	// MinNameParts is the minimum number of parts in a task definition name
	MinNameParts = 2

	// EstimatedResourcesPerCluster is the estimated number of resources per cluster for capacity pre-allocation
	EstimatedResourcesPerCluster = 5
	// EstimatedServicesPerCluster is the estimated number of services per cluster for capacity pre-allocation
	EstimatedServicesPerCluster = 10
	// EstimatedPortMappingsPerContainer is the estimated number of port mappings per container
	EstimatedPortMappingsPerContainer = 2
	// EstimatedEnvVarsPerContainer is the estimated number of environment variables per container
	EstimatedEnvVarsPerContainer = 5
)

var (
	// ErrTaskDefinitionNotFound is returned when a task definition is not found
	ErrTaskDefinitionNotFound = errors.New("task definition not found")
)

// ECSCollector collects ECS resources.
// This collector is stateless and safe for concurrent use across multiple goroutines.
// Each Collect() call creates its own clients and cache, ensuring thread safety.
type ECSCollector struct{}

// Name returns the resource name of the collector.
func (*ECSCollector) Name() string {
	return "ecs"
}

// ShouldSort returns whether the collected resources should be sorted.
func (*ECSCollector) ShouldSort() bool {
	return false
}

// GetColumns returns the CSV columns for the collector.
func (*ECSCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "RoleARN", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RoleARN") }},
		{Header: "TaskDefinition", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "TaskDefinition") }},
		{Header: "LaunchType", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LaunchType") }},
		{Header: "Status", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Status") }},
		{Header: "CronSchedule", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CronSchedule") }},
		{Header: "Spec", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Spec") }},
		{Header: "RuntimePlatform", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RuntimePlatform") }},
		{Header: "PortMappings", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PortMappings") }},
		{Header: "Environment", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Environment") }},
	}
}

// Collect collects ECS resources.
// This method is safe for concurrent execution across multiple goroutines and regions.
func (c *ECSCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	ecsClient := ecs.NewFromConfig(*cfg, func(o *ecs.Options) {
		o.Region = region
	})
	ebClient := eventbridge.NewFromConfig(*cfg, func(o *eventbridge.Options) {
		o.Region = region
	})

	// Create a local cache for this collection run (thread-safe for this goroutine)
	taskDefCache := make(map[string]*types.TaskDefinition)

	// Collect scheduled tasks
	scheduledTasks, err := c.collectScheduledTasks(ctx, ecsClient, ebClient, region, taskDefCache)
	if err != nil {
		return nil, fmt.Errorf("failed to collect scheduled tasks: %w", err)
	}

	// Collect clusters and services
	clusterResources, err := c.collectClustersAndServices(ctx, ecsClient, region, scheduledTasks, taskDefCache)
	if err != nil {
		return nil, fmt.Errorf("failed to collect clusters and services: %w", err)
	}

	// Collect task definitions
	taskDefResources, err := c.collectTaskDefinitions(ctx, ecsClient, region, taskDefCache)
	if err != nil {
		return nil, fmt.Errorf("failed to collect task definitions: %w", err)
	}

	allResources := make([]Resource, 0, len(clusterResources)+len(taskDefResources))
	allResources = append(allResources, clusterResources...)
	allResources = append(allResources, taskDefResources...)
	return allResources, nil
}

// getTaskDef gets task definition with caching.
// This is a package-level helper function that takes cache as a parameter,
// making the collector safe for concurrent use across multiple goroutines.
func getTaskDef(ctx context.Context, ecsClient *ecs.Client, arn string, taskDefCache map[string]*types.TaskDefinition) (*types.TaskDefinition, error) {
	if td, ok := taskDefCache[arn]; ok {
		return td, nil
	}
	out, err := ecsClient.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(arn),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe task definition %s: %w", arn, err)
	}
	if out.TaskDefinition != nil {
		taskDefCache[helpers.StringValue(out.TaskDefinition.TaskDefinitionArn)] = out.TaskDefinition
		return out.TaskDefinition, nil
	}
	return nil, ErrTaskDefinitionNotFound
}

// collectScheduledTasks collects scheduled tasks from EventBridge rules
func (*ECSCollector) collectScheduledTasks(ctx context.Context, ecsClient *ecs.Client, ebClient *eventbridge.Client, region string, taskDefCache map[string]*types.TaskDefinition) (map[string][]Resource, error) {
	scheduledTasksByCluster := make(map[string][]Resource)

	var nextToken *string
	for {
		page, err := ebClient.ListRules(ctx, &eventbridge.ListRulesInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list rules: %w", err)
		}

		for i := range page.Rules {
			rule := &page.Rules[i]
			if rule.State != ebtypes.RuleStateEnabled {
				continue
			}

			// List Targets for the rule
			targetOut, listErr := ebClient.ListTargetsByRule(ctx, &eventbridge.ListTargetsByRuleInput{
				Rule: rule.Name,
			})
			if listErr != nil {
				continue
			}

			for j := range targetOut.Targets {
				target := &targetOut.Targets[j]
				if target.EcsParameters == nil {
					continue
				}

				// The target ARN is the Cluster ARN for ECS targets
				clusterArn := helpers.StringValue(target.Arn)
				taskDefArn := helpers.StringValue(target.EcsParameters.TaskDefinitionArn)
				taskLaunchType := string(target.EcsParameters.LaunchType)

				// Get RoleARN from task definition
				var taskRoleArn string
				if taskDefArn != "" {
					td, tdErr := getTaskDef(ctx, ecsClient, taskDefArn, taskDefCache)
					if tdErr == nil && td != nil {
						if td.TaskRoleArn != nil {
							taskRoleArn = *td.TaskRoleArn
						} else if td.ExecutionRoleArn != nil {
							taskRoleArn = *td.ExecutionRoleArn
						}
					}
				}

				r := NewResource(&ResourceInput{
					Category:       "ecs",
					SubCategory:    "",
					SubSubCategory: "ScheduledTask",
					Name:           rule.Name,
					Region:         region,
					ARN:            rule.Arn,
					RawData: map[string]any{
						"RoleARN":        taskRoleArn,
						"TaskDefinition": taskDefArn,
						"LaunchType":     taskLaunchType,
						"Status":         rule.State,
						"CronSchedule":   rule.ScheduleExpression,
					},
				})

				if clusterArn != "" {
					scheduledTasksByCluster[clusterArn] = append(scheduledTasksByCluster[clusterArn], r)
				}
			}
		}

		if page.NextToken == nil {
			break
		}
		nextToken = page.NextToken
	}

	return scheduledTasksByCluster, nil
}

// collectClustersAndServices collects clusters and their services
func (c *ECSCollector) collectClustersAndServices(ctx context.Context, ecsClient *ecs.Client, region string, scheduledTasks map[string][]Resource, taskDefCache map[string]*types.TaskDefinition) ([]Resource, error) {
	// Pre-allocate with estimated capacity: cluster + services + scheduled tasks
	resources := make([]Resource, 0, len(scheduledTasks)*EstimatedResourcesPerCluster)

	clusterPaginator := ecs.NewListClustersPaginator(ecsClient, &ecs.ListClustersInput{})
	for clusterPaginator.HasMorePages() {
		page, err := clusterPaginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list clusters: %w", err)
		}

		if len(page.ClusterArns) == 0 {
			continue
		}

		// Describe Clusters to get details
		descClustersOut, err := ecsClient.DescribeClusters(ctx, &ecs.DescribeClustersInput{
			Clusters: page.ClusterArns,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to describe clusters: %w", err)
		}

		for i := range descClustersOut.Clusters {
			cluster := &descClustersOut.Clusters[i]
			clusterArn := helpers.StringValue(cluster.ClusterArn)

			// Add cluster
			resources = append(resources, NewResource(&ResourceInput{
				Category:    "ecs",
				SubCategory: "Cluster",
				Name:        cluster.ClusterName,
				Region:      region,
				ARN:         cluster.ClusterArn,
				RawData: map[string]any{
					"Status": cluster.Status,
				},
			}))

			// Add services for this cluster
			serviceResources := c.collectServices(ctx, ecsClient, region, clusterArn, taskDefCache)
			resources = append(resources, serviceResources...)

			// Add scheduled tasks for this cluster
			if tasks, ok := scheduledTasks[clusterArn]; ok {
				resources = append(resources, tasks...)
			}
		}
	}

	return resources, nil
}

// collectServices collects services for the given cluster
func (*ECSCollector) collectServices(ctx context.Context, ecsClient *ecs.Client, region, clusterArn string, taskDefCache map[string]*types.TaskDefinition) []Resource {
	// Pre-allocate with estimated capacity for services per cluster
	resources := make([]Resource, 0, EstimatedServicesPerCluster)

	servicePaginator := ecs.NewListServicesPaginator(ecsClient, &ecs.ListServicesInput{
		Cluster: aws.String(clusterArn),
	})

	for servicePaginator.HasMorePages() {
		svcPage, svcErr := servicePaginator.NextPage(ctx)
		if svcErr != nil {
			continue
		}
		if len(svcPage.ServiceArns) == 0 {
			continue
		}

		// Describe Services (max MaxServicesPerDescribe)
		for chunkIndex := 0; chunkIndex < len(svcPage.ServiceArns); chunkIndex += MaxServicesPerDescribe {
			end := chunkIndex + MaxServicesPerDescribe
			if end > len(svcPage.ServiceArns) {
				end = len(svcPage.ServiceArns)
			}
			chunk := svcPage.ServiceArns[chunkIndex:end]

			descServicesOut, descErr := ecsClient.DescribeServices(ctx, &ecs.DescribeServicesInput{
				Cluster:  aws.String(clusterArn),
				Services: chunk,
			})
			if descErr != nil {
				continue
			}

			for j := range descServicesOut.Services {
				service := &descServicesOut.Services[j]
				serviceStatus := fmt.Sprintf("%s (%d/%d)", helpers.StringValue(service.Status), service.RunningCount, service.DesiredCount)

				taskDefArn := helpers.StringValue(service.TaskDefinition)
				var taskRoleArn string

				if taskDefArn != "" {
					td, taskDefErr := getTaskDef(ctx, ecsClient, taskDefArn, taskDefCache)
					if taskDefErr == nil && td != nil {
						if td.TaskRoleArn != nil {
							taskRoleArn = *td.TaskRoleArn
						} else if td.ExecutionRoleArn != nil {
							taskRoleArn = *td.ExecutionRoleArn
						}
					}
				}

				// If we couldn't get the role ARN, mark as N/A
				if taskRoleArn == "" && taskDefArn != "" {
					taskRoleArn = "N/A"
				}

				resources = append(resources, NewResource(&ResourceInput{
					Category:       "ecs",
					SubCategory:    "",
					SubSubCategory: "Service",
					Name:           service.ServiceName,
					Region:         region,
					ARN:            service.ServiceArn,
					RawData: map[string]any{
						"RoleARN":        taskRoleArn,
						"TaskDefinition": taskDefArn,
						"LaunchType":     service.LaunchType,
						"Status":         serviceStatus,
					},
				}))
			}
		}
	}

	return resources
}

// collectTaskDefinitions collects task definitions (latest revision per family only)
func (*ECSCollector) collectTaskDefinitions(ctx context.Context, ecsClient *ecs.Client, region string, taskDefCache map[string]*types.TaskDefinition) ([]Resource, error) {
	// Strategy: ListTaskDefinitions returns all revisions in ascending order by revision number.
	// We group by family and keep only the latest (last seen) revision per family
	// to reduce noise and focus on currently active task definitions.
	familyMap := make(map[string]string) // family -> latest ARN

	taskDefPaginator := ecs.NewListTaskDefinitionsPaginator(ecsClient, &ecs.ListTaskDefinitionsInput{})
	for taskDefPaginator.HasMorePages() {
		page, err := taskDefPaginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list task definitions: %w", err)
		}

		for _, taskDefArn := range page.TaskDefinitionArns {
			// Extract family from ARN: arn:aws:ecs:region:account:task-definition/family:revision
			parts := strings.Split(taskDefArn, "/")
			if len(parts) < MinARNParts {
				continue
			}
			nameParts := strings.Split(parts[1], ":")
			if len(nameParts) < MinNameParts {
				continue
			}
			family := nameParts[0]
			// ListTaskDefinitions returns in ascending order, so overwriting gives us the latest
			familyMap[family] = taskDefArn
		}
	}

	// Sort families for deterministic processing
	families := make([]string, 0, len(familyMap))
	for f := range familyMap {
		families = append(families, f)
	}
	sort.Strings(families)

	// Pre-allocate resources slice with exact capacity
	resources := make([]Resource, 0, len(families))

	// Process each family (latest revision only)
	for _, family := range families {
		taskDefArn := familyMap[family]
		td, err := getTaskDef(ctx, ecsClient, taskDefArn, taskDefCache)
		if err != nil || td == nil {
			continue
		}

		// Build Spec: CPU/Memory/NetworkMode
		spec := fmt.Sprintf("%sCPU/%sMB/%s", helpers.StringValue(td.Cpu), helpers.StringValue(td.Memory), string(td.NetworkMode))

		// Build RuntimePlatform
		var runtimePlatform string
		if td.RuntimePlatform != nil {
			osFamily := string(td.RuntimePlatform.OperatingSystemFamily)
			cpuArch := string(td.RuntimePlatform.CpuArchitecture)
			if osFamily != "" && cpuArch != "" {
				runtimePlatform = fmt.Sprintf("%s/%s", osFamily, cpuArch)
			} else {
				runtimePlatform = "LINUX/X86_64"
			}
		} else {
			runtimePlatform = "LINUX/X86_64"
		}

		// Build PortMappings and Environment from all containers
		// Pre-allocate slices with estimated capacity based on container definitions
		var portMappings, environment []string
		if len(td.ContainerDefinitions) > 0 {
			// Pre-allocate based on estimated port mappings and environment variables per container
			portMappings = make([]string, 0, len(td.ContainerDefinitions)*EstimatedPortMappingsPerContainer)
			environment = make([]string, 0, len(td.ContainerDefinitions)*EstimatedEnvVarsPerContainer)
		}
		for i := range td.ContainerDefinitions {
			container := &td.ContainerDefinitions[i]
			for j := range container.PortMappings {
				pm := &container.PortMappings[j]
				protocol := string(pm.Protocol)
				if protocol == "" {
					protocol = "tcp"
				}
				hostPort := "dynamic"
				if pm.HostPort != nil && *pm.HostPort > 0 {
					hostPort = fmt.Sprintf("%d", *pm.HostPort)
				}
				containerPort := helpers.StringValue(pm.ContainerPort)
				portMappings = append(portMappings, fmt.Sprintf("%s:%s:%s", containerPort, hostPort, protocol))
			}

			for k := range container.Environment {
				env := &container.Environment[k]
				environment = append(environment, fmt.Sprintf("%s=%s", helpers.StringValue(env.Name), helpers.StringValue(env.Value)))
			}
		}

		// Get RoleARN
		var taskRoleArn string
		if td.TaskRoleArn != nil {
			taskRoleArn = *td.TaskRoleArn
		} else if td.ExecutionRoleArn != nil {
			taskRoleArn = *td.ExecutionRoleArn
		}

		// Name format: family:revision
		name := fmt.Sprintf("%s:%d", helpers.StringValue(td.Family), td.Revision)

		resources = append(resources, NewResource(&ResourceInput{
			Category:    "ecs",
			SubCategory: "TaskDefinition",
			Name:        name,
			Region:      region,
			ARN:         td.TaskDefinitionArn,
			RawData: map[string]any{
				"RoleARN":         taskRoleArn,
				"TaskDefinition":  td.TaskDefinitionArn,
				"Status":          td.Status,
				"Spec":            spec,
				"RuntimePlatform": runtimePlatform,
				"PortMappings":    portMappings,
				"Environment":     environment,
			},
		}))
	}

	return resources, nil
}

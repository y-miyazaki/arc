package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewECSCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewECSCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.clients)
	assert.Len(t, collector.clients, len(regions))
	assert.NotNil(t, collector.ebClients)
	assert.Len(t, collector.ebClients, len(regions))
	assert.NotNil(t, collector.nameResolver)
}

func TestNewECSCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewECSCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.clients)
	assert.Len(t, collector.clients, 0)
	assert.NotNil(t, collector.ebClients)
	assert.Len(t, collector.ebClients, 0)
	assert.NotNil(t, collector.nameResolver)
}

func TestECSCollector_Basic(t *testing.T) {
	collector := &ECSCollector{
		clients:   map[string]*ecs.Client{},
		ebClients: map[string]*eventbridge.Client{},
	}
	assert.Equal(t, "ecs", collector.Name())
	assert.False(t, collector.ShouldSort())
}

func TestECSCollector_GetColumns(t *testing.T) {
	collector := &ECSCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory1", "SubCategory2", "Name", "Region", "ARN",
		"RoleARN", "TaskDefinition", "LaunchType", "Status", "CronSchedule",
		"Spec", "RuntimePlatform", "PortMappings", "Environment",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:     "ECS",
		SubCategory1: "Service",
		SubCategory2: "",
		Name:         "test-service",
		Region:       "us-east-1",
		ARN:          "arn:aws:ecs:us-east-1:123456789012:service/test-cluster/test-service",
		RawData: map[string]interface{}{
			"RoleARN":         "arn:aws:iam::123456789012:role/ecsTaskExecutionRole",
			"TaskDefinition":  "test-task-definition:1",
			"LaunchType":      "FARGATE",
			"Status":          "ACTIVE",
			"CronSchedule":    "cron(0 12 * * ? *)",
			"Spec":            "CPU: 256, Memory: 512",
			"RuntimePlatform": "LINUX/X86_64",
			"PortMappings":    "80/tcp",
			"Environment":     "KEY1=value1\nKEY2=value2",
		},
	}

	expectedValues := []string{
		"ECS", "Service", "", "test-service", "us-east-1", "arn:aws:ecs:us-east-1:123456789012:service/test-cluster/test-service",
		"arn:aws:iam::123456789012:role/ecsTaskExecutionRole", "test-task-definition:1", "FARGATE", "ACTIVE", "cron(0 12 * * ? *)",
		"CPU: 256, Memory: 512", "LINUX/X86_64", "80/tcp", "KEY1=value1\nKEY2=value2",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}

func TestECSCollector_Collect_EventBridgeClientMissing(t *testing.T) {
	collector := &ECSCollector{
		clients: map[string]*ecs.Client{},
	}

	_, err := collector.Collect(context.Background(), "us-east-1")

	require.Error(t, err)
	assert.ErrorContains(t, err, "no client found for region")
}

func TestECSCollector_Collect_ScheduledTasksError(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1", Credentials: aws.AnonymousCredentials{}}
	collector := &ECSCollector{
		clients: map[string]*ecs.Client{
			"us-east-1": ecs.NewFromConfig(cfg),
		},
		ebClients: map[string]*eventbridge.Client{
			"us-east-1": eventbridge.NewFromConfig(cfg),
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := collector.Collect(ctx, "us-east-1")

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to collect scheduled tasks")
}

func TestGetTaskDef_DescribeError(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1", Credentials: aws.AnonymousCredentials{}}
	client := ecs.NewFromConfig(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := getTaskDef(ctx, client, "arn:aws:ecs:us-east-1:123456789012:task-definition/test:1", map[string]*ecstypes.TaskDefinition{})

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to describe task definition")
}

func TestECSCollector_collectScheduledTasks_ListRulesError(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1", Credentials: aws.AnonymousCredentials{}}
	collector := &ECSCollector{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := collector.collectScheduledTasks(
		ctx,
		ecs.NewFromConfig(cfg),
		eventbridge.NewFromConfig(cfg),
		"us-east-1",
		map[string]*ecstypes.TaskDefinition{},
	)

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to list rules")
}

func TestECSCollector_collectScheduledTasks_Success(t *testing.T) {
	clusterArn := "arn:aws:ecs:us-east-1:123456789012:cluster/test"
	taskDefArn := "arn:aws:ecs:us-east-1:123456789012:task-definition/test:1"
	ruleArn := "arn:aws:events:us-east-1:123456789012:rule/ecs-schedule"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := r.Header.Get("X-Amz-Target")
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")

		switch target {
		case "AWSEvents.ListRules":
			_, _ = w.Write([]byte(`{"Rules":[{"Name":"ecs-schedule","Arn":"` + ruleArn + `","State":"ENABLED","ScheduleExpression":"rate(5 minutes)"},{"Name":"disabled-rule","Arn":"arn:aws:events:us-east-1:123456789012:rule/disabled-rule","State":"DISABLED","ScheduleExpression":"rate(1 minute)"}]}`))
		case "AWSEvents.ListTargetsByRule":
			_, _ = w.Write([]byte(`{"Targets":[{"Arn":"` + clusterArn + `","EcsParameters":{"TaskDefinitionArn":"` + taskDefArn + `","LaunchType":"FARGATE"}},{"Arn":"arn:aws:sns:us-east-1:123456789012:topic/no-ecs-params"}]}`))
		case "AmazonEC2ContainerServiceV20141113.DescribeTaskDefinition":
			_, _ = w.Write([]byte(`{"taskDefinition":{"taskDefinitionArn":"` + taskDefArn + `","family":"test","revision":1,"executionRoleArn":"arn:aws:iam::123456789012:role/execution-role","networkMode":"awsvpc"}}`))
		default:
			http.Error(w, "unexpected target: "+target, http.StatusBadRequest)
		}
	}))
	defer server.Close()

	cfg := aws.Config{
		Region:      "us-east-1",
		Credentials: aws.AnonymousCredentials{},
		HTTPClient:  server.Client(),
		EndpointResolverWithOptions: aws.EndpointResolverWithOptionsFunc(func(_, _ string, _ ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{URL: server.URL, HostnameImmutable: true}, nil
		}),
		RetryMaxAttempts: 1,
	}

	collector := &ECSCollector{}
	result, err := collector.collectScheduledTasks(
		context.Background(),
		ecs.NewFromConfig(cfg),
		eventbridge.NewFromConfig(cfg),
		"us-east-1",
		map[string]*ecstypes.TaskDefinition{},
	)

	require.NoError(t, err)
	require.Contains(t, result, clusterArn)
	require.Len(t, result[clusterArn], 1)

	resource := result[clusterArn][0]
	assert.Equal(t, "ScheduledTask", resource.SubCategory2)
	assert.Equal(t, "ecs-schedule", resource.Name)
	assert.Equal(t, ruleArn, resource.ARN)
	assert.Equal(t, taskDefArn, resource.RawData["TaskDefinition"])
	assert.Equal(t, "arn:aws:iam::123456789012:role/execution-role", resource.RawData["RoleARN"])
	assert.Equal(t, "ENABLED", resource.RawData["Status"])
	assert.Equal(t, "rate(5 minutes)", resource.RawData["CronSchedule"])
	assert.Equal(t, "FARGATE", resource.RawData["LaunchType"])
}

func TestECSCollector_collectClustersAndServices_ListClustersError(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1", Credentials: aws.AnonymousCredentials{}}
	collector := &ECSCollector{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := collector.collectClustersAndServices(
		ctx,
		ecs.NewFromConfig(cfg),
		"us-east-1",
		map[string][]Resource{},
		map[string]*ecstypes.TaskDefinition{},
	)

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to list clusters")
}

func TestECSCollector_collectClustersAndServices_DescribeClustersError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := r.Header.Get("X-Amz-Target")
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")

		switch target {
		case "AmazonEC2ContainerServiceV20141113.ListClusters":
			_, _ = w.Write([]byte(`{"clusterArns":["arn:aws:ecs:us-east-1:123456789012:cluster/test"]}`))
		case "AmazonEC2ContainerServiceV20141113.DescribeClusters":
			http.Error(w, "describe clusters failed", http.StatusInternalServerError)
		default:
			http.Error(w, "unexpected target: "+target, http.StatusBadRequest)
		}
	}))
	defer server.Close()

	cfg := aws.Config{
		Region:      "us-east-1",
		Credentials: aws.AnonymousCredentials{},
		HTTPClient:  server.Client(),
		EndpointResolverWithOptions: aws.EndpointResolverWithOptionsFunc(func(_, _ string, _ ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{URL: server.URL, HostnameImmutable: true}, nil
		}),
		RetryMaxAttempts: 1,
	}

	collector := &ECSCollector{}
	_, err := collector.collectClustersAndServices(
		context.Background(),
		ecs.NewFromConfig(cfg),
		"us-east-1",
		map[string][]Resource{},
		map[string]*ecstypes.TaskDefinition{},
	)

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to describe clusters")
}

func TestECSCollector_collectClustersAndServices_Success(t *testing.T) {
	clusterArn := "arn:aws:ecs:us-east-1:123456789012:cluster/test"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := r.Header.Get("X-Amz-Target")
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")

		switch target {
		case "AmazonEC2ContainerServiceV20141113.ListClusters":
			_, _ = w.Write([]byte(`{"clusterArns":["` + clusterArn + `"]}`))
		case "AmazonEC2ContainerServiceV20141113.DescribeClusters":
			_, _ = w.Write([]byte(`{"clusters":[{"clusterArn":"` + clusterArn + `","clusterName":"test-cluster","status":"ACTIVE"}],"failures":[]}`))
		case "AmazonEC2ContainerServiceV20141113.ListServices":
			_, _ = w.Write([]byte(`{"serviceArns":[]}`))
		default:
			http.Error(w, "unexpected target: "+target, http.StatusBadRequest)
		}
	}))
	defer server.Close()

	cfg := aws.Config{
		Region:      "us-east-1",
		Credentials: aws.AnonymousCredentials{},
		HTTPClient:  server.Client(),
		EndpointResolverWithOptions: aws.EndpointResolverWithOptionsFunc(func(_, _ string, _ ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{URL: server.URL, HostnameImmutable: true}, nil
		}),
		RetryMaxAttempts: 1,
	}

	collector := &ECSCollector{}
	scheduled := map[string][]Resource{
		clusterArn: {
			NewResource(&ResourceInput{
				Category:     "ecs",
				SubCategory2: "ScheduledTask",
				Name:         "dummy-task",
				Region:       "us-east-1",
				ARN:          "arn:aws:events:us-east-1:123456789012:rule/dummy",
				RawData:      map[string]any{"Status": "ENABLED"},
			}),
		},
	}

	resources, err := collector.collectClustersAndServices(
		context.Background(),
		ecs.NewFromConfig(cfg),
		"us-east-1",
		scheduled,
		map[string]*ecstypes.TaskDefinition{},
	)

	require.NoError(t, err)
	require.Len(t, resources, 2)

	assert.Equal(t, "Cluster", resources[0].SubCategory1)
	assert.Equal(t, "test-cluster", resources[0].Name)
	assert.Equal(t, clusterArn, resources[0].ARN)
	assert.Equal(t, "ACTIVE", resources[0].RawData["Status"])

	assert.Equal(t, "ScheduledTask", resources[1].SubCategory2)
	assert.Equal(t, "dummy-task", resources[1].Name)
}

func TestECSCollector_collectTaskDefinitions_ListError(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1", Credentials: aws.AnonymousCredentials{}}
	collector := &ECSCollector{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := collector.collectTaskDefinitions(
		ctx,
		ecs.NewFromConfig(cfg),
		"us-east-1",
		map[string]*ecstypes.TaskDefinition{},
	)

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to list task definitions")
}

func TestECSCollector_collectServices_EmptyServiceArns(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		_, _ = w.Write([]byte(`{"serviceArns":[]}`))
	}))
	defer server.Close()

	cfg := aws.Config{
		Region:      "us-east-1",
		Credentials: aws.AnonymousCredentials{},
		HTTPClient:  server.Client(),
		EndpointResolverWithOptions: aws.EndpointResolverWithOptionsFunc(func(_, _ string, _ ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{URL: server.URL, HostnameImmutable: true}, nil
		}),
		RetryMaxAttempts: 1,
	}

	collector := &ECSCollector{}
	resources := collector.collectServices(
		context.Background(),
		ecs.NewFromConfig(cfg),
		"us-east-1",
		"arn:aws:ecs:us-east-1:123456789012:cluster/test",
		map[string]*ecstypes.TaskDefinition{},
	)

	assert.Empty(t, resources)
}

func TestECSCollector_collectServices_WithTaskRole(t *testing.T) {
	serviceArn := "arn:aws:ecs:us-east-1:123456789012:service/test-cluster/test-service"
	taskDefArn := "arn:aws:ecs:us-east-1:123456789012:task-definition/test:1"
	roleArn := "arn:aws:iam::123456789012:role/ecsTaskRole"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := r.Header.Get("X-Amz-Target")
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")

		switch {
		case target == "AmazonEC2ContainerServiceV20141113.ListServices":
			_, _ = w.Write([]byte(`{"serviceArns":["` + serviceArn + `"]}`))
		case target == "AmazonEC2ContainerServiceV20141113.DescribeServices":
			_, _ = w.Write([]byte(`{"services":[{"serviceArn":"` + serviceArn + `","serviceName":"test-service","status":"ACTIVE","runningCount":1,"desiredCount":1,"taskDefinition":"` + taskDefArn + `","launchType":"FARGATE"}],"failures":[]}`))
		case target == "AmazonEC2ContainerServiceV20141113.DescribeTaskDefinition":
			payload := map[string]any{
				"taskDefinition": map[string]any{
					"taskDefinitionArn": taskDefArn,
					"family":            "test",
					"revision":          1,
					"taskRoleArn":       roleArn,
					"networkMode":       "awsvpc",
				},
			}
			_ = json.NewEncoder(w).Encode(payload)
		default:
			http.Error(w, "unexpected target: "+target, http.StatusBadRequest)
		}
	}))
	defer server.Close()

	cfg := aws.Config{
		Region:      "us-east-1",
		Credentials: aws.AnonymousCredentials{},
		HTTPClient:  server.Client(),
		EndpointResolverWithOptions: aws.EndpointResolverWithOptionsFunc(func(_, _ string, _ ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{URL: server.URL, HostnameImmutable: true}, nil
		}),
		RetryMaxAttempts: 1,
	}

	collector := &ECSCollector{}
	resources := collector.collectServices(
		context.Background(),
		ecs.NewFromConfig(cfg),
		"us-east-1",
		"arn:aws:ecs:us-east-1:123456789012:cluster/test",
		map[string]*ecstypes.TaskDefinition{},
	)

	if assert.Len(t, resources, 1) {
		assert.Equal(t, "test-service", resources[0].Name)
		assert.Equal(t, serviceArn, resources[0].ARN)
		assert.Equal(t, roleArn, resources[0].RawData["RoleARN"])
		assert.Equal(t, taskDefArn, resources[0].RawData["TaskDefinition"])
	}
}

func TestECSCollector_collectTaskDefinitions_Success(t *testing.T) {
	arnA1 := "arn:aws:ecs:us-east-1:123456789012:task-definition/family-a:1"
	arnA2 := "arn:aws:ecs:us-east-1:123456789012:task-definition/family-a:2"
	arnB1 := "arn:aws:ecs:us-east-1:123456789012:task-definition/family-b:1"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := r.Header.Get("X-Amz-Target")
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")

		switch target {
		case "AmazonEC2ContainerServiceV20141113.ListTaskDefinitions":
			_, _ = w.Write([]byte(`{"taskDefinitionArns":["invalid-arn","` + arnA1 + `","` + arnA2 + `","` + arnB1 + `"]}`))
		case "AmazonEC2ContainerServiceV20141113.DescribeTaskDefinition":
			var req map[string]string
			_ = json.NewDecoder(r.Body).Decode(&req)
			asked := req["taskDefinition"]

			switch asked {
			case arnA2:
				_, _ = w.Write([]byte(`{"taskDefinition":{"taskDefinitionArn":"` + arnA2 + `","family":"family-a","revision":2,"taskRoleArn":"arn:aws:iam::123456789012:role/task-role-a","status":"ACTIVE","cpu":"256","memory":"512","networkMode":"awsvpc","runtimePlatform":{"operatingSystemFamily":"LINUX","cpuArchitecture":"X86_64"},"containerDefinitions":[{"name":"app","portMappings":[{"containerPort":80}],"environment":[{"name":"KEY","value":"VALUE"}]}]}}`))
			case arnB1:
				_, _ = w.Write([]byte(`{"taskDefinition":{"taskDefinitionArn":"` + arnB1 + `","family":"family-b","revision":1,"executionRoleArn":"arn:aws:iam::123456789012:role/execution-role-b","status":"ACTIVE","cpu":"1024","memory":"2048","networkMode":"bridge","runtimePlatform":{"operatingSystemFamily":""},"containerDefinitions":[{"name":"worker","portMappings":[{"containerPort":8080,"hostPort":18080,"protocol":"udp"}],"environment":[{"name":"A","value":"B"}]}]}}`))
			default:
				http.Error(w, "unexpected task definition", http.StatusBadRequest)
			}
		default:
			http.Error(w, "unexpected target: "+target, http.StatusBadRequest)
		}
	}))
	defer server.Close()

	cfg := aws.Config{
		Region:      "us-east-1",
		Credentials: aws.AnonymousCredentials{},
		HTTPClient:  server.Client(),
		EndpointResolverWithOptions: aws.EndpointResolverWithOptionsFunc(func(_, _ string, _ ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{URL: server.URL, HostnameImmutable: true}, nil
		}),
		RetryMaxAttempts: 1,
	}

	collector := &ECSCollector{}
	cache := map[string]*ecstypes.TaskDefinition{}
	resources, err := collector.collectTaskDefinitions(context.Background(), ecs.NewFromConfig(cfg), "us-east-1", cache)

	require.NoError(t, err)
	require.Len(t, resources, 2)

	// family-a latest revision only
	assert.Equal(t, "family-a:2", resources[0].Name)
	assert.Equal(t, arnA2, resources[0].ARN)
	assert.Equal(t, "arn:aws:iam::123456789012:role/task-role-a", resources[0].RawData["RoleARN"])
	assert.Equal(t, "256CPU/512MB/awsvpc", resources[0].RawData["Spec"])
	assert.Equal(t, "LINUX/X86_64", resources[0].RawData["RuntimePlatform"])
	assert.Contains(t, resources[0].RawData["PortMappings"], "80:dynamic:tcp")

	// family-b role fallback + runtime fallback
	assert.Equal(t, "family-b:1", resources[1].Name)
	assert.Equal(t, arnB1, resources[1].ARN)
	assert.Equal(t, "arn:aws:iam::123456789012:role/execution-role-b", resources[1].RawData["RoleARN"])
	assert.Equal(t, "1024CPU/2048MB/bridge", resources[1].RawData["Spec"])
	assert.Equal(t, "LINUX/X86_64", resources[1].RawData["RuntimePlatform"])
	assert.Contains(t, resources[1].RawData["PortMappings"], "8080:18080:udp")
	assert.Contains(t, resources[1].RawData["Environment"], "A=B")
}

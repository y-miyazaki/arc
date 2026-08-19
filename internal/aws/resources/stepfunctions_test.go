package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewStepFunctionsCollector(t *testing.T) {
	t.Parallel()

	cfg := &aws.Config{
		Region: "us-east-1",
	}

	tests := []struct {
		name    string
		regions []string
		wantLen int
	}{
		{name: "creates clients for each region", regions: []string{"us-east-1", "eu-west-1"}, wantLen: 2},
		{name: "empty regions", regions: []string{}, wantLen: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nameResolver, err := helpers.NewNameResolver(cfg, tt.regions)
			require.NoError(t, err)

			collector, err := NewStepFunctionsCollector(cfg, tt.regions, nameResolver)
			require.NoError(t, err)
			require.NotNil(t, collector)
			assert.Len(t, collector.clients, tt.wantLen)
			for _, region := range tt.regions {
				assert.Contains(t, collector.clients, region)
			}
			assert.NotNil(t, collector.nameResolver)
		})
	}
}

func TestStepFunctionsCollector_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantName string
		wantSort bool
	}{
		{name: "reports name and sort", wantName: "stepfunctions", wantSort: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &StepFunctionsCollector{
				clients: make(map[string]*sfn.Client),
			}
			assert.Equal(t, tt.wantName, collector.Name())
			assert.Equal(t, tt.wantSort, collector.ShouldSort())
		})
	}
}

func TestStepFunctionsCollector_Collect_NoClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		region       string
		wantContains string
	}{
		{name: "missing client returns error", region: "us-west-2", wantContains: "no client found for region"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &StepFunctionsCollector{
				clients: make(map[string]*sfn.Client),
			}
			_, err := collector.Collect(context.Background(), tt.region)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantContains)
		})
	}
}

func TestStepFunctionsCollector_GetColumns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		resource    Resource
		wantHeaders []string
		wantValues  []string
	}{
		{
			name: "headers and sample values",
			resource: Resource{
				Category:     "stepfunctions",
				SubCategory1: "StateMachine",
				Name:         "order-workflow",
				Region:       "us-east-1",
				ARN:          "arn:aws:states:us-east-1:123456789012:stateMachine:order-workflow",
				RawData: map[string]any{
					"Type":                         "STANDARD",
					"Status":                       "ACTIVE",
					"RoleARN":                      "arn:aws:iam::123456789012:role/step-functions-role",
					"LoggingLevel":                 "ALL",
					"LoggingIncludeExecutionData":  "true",
					"LogDestination":               "arn:aws:logs:us-east-1:123456789012:log-group:/aws/states/order:*",
					"TracingEnabled":               "true",
					"EncryptionType":               "CUSTOMER_MANAGED_KMS_KEY",
					"KMSKeyID":                     "arn:aws:kms:us-east-1:123456789012:key/test",
					"KMSDataKeyReusePeriodSeconds": "300",
					"Definition":                   `{"Comment":"Order processing workflow"}`,
					"RevisionID":                   "revision-1",
					"CreatedDate":                  "2026-03-17T00:00:00Z",
					"Comment":                      "Order processing workflow",
				},
			},
			wantHeaders: []string{
				"Category", "SubCategory1", "Name", "Region", "ARN", "Comment", "Type", "RoleARN",
				"LoggingLevel", "LoggingIncludeExecutionData", "LogDestination", "TracingEnabled",
				"EncryptionType", "KMSKeyID", "KMSDataKeyReusePeriodSeconds", "Definition",
				"RevisionID", "Status", "CreatedDate",
			},
			wantValues: []string{
				"stepfunctions", "StateMachine", "order-workflow", "us-east-1",
				"arn:aws:states:us-east-1:123456789012:stateMachine:order-workflow", "Order processing workflow", "STANDARD",
				"arn:aws:iam::123456789012:role/step-functions-role", "ALL", "true",
				"arn:aws:logs:us-east-1:123456789012:log-group:/aws/states/order:*", "true",
				"CUSTOMER_MANAGED_KMS_KEY", "arn:aws:kms:us-east-1:123456789012:key/test", "300",
				`{"Comment":"Order processing workflow"}`, "revision-1", "ACTIVE", "2026-03-17T00:00:00Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &StepFunctionsCollector{}
			columns := collector.GetColumns()
			require.Len(t, columns, len(tt.wantHeaders))
			for i, column := range columns {
				assert.Equal(t, tt.wantHeaders[i], column.Header)
				assert.Equal(t, tt.wantValues[i], column.Value(tt.resource), "Column %d (%s) value mismatch", i, column.Header)
			}
		})
	}
}

func TestStepFunctionsCollector_Collect_ListStateMachinesError(t *testing.T) {
	cfg := aws.Config{
		Region:      "us-east-1",
		Credentials: aws.AnonymousCredentials{},
	}
	collector := &StepFunctionsCollector{
		clients: map[string]*sfn.Client{
			"us-east-1": sfn.NewFromConfig(cfg),
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := collector.Collect(ctx, "us-east-1")

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to list state machines")
}

func TestGetDefinitionComment(t *testing.T) {
	valid := `{"Comment":"workflow comment"}`
	invalid := "not-json"
	missing := `{"Name":"workflow"}`

	tests := []struct {
		name     string
		input    *string
		expected string
	}{
		{name: "nil definition", input: nil, expected: ""},
		{name: "invalid json", input: aws.String(invalid), expected: ""},
		{name: "missing comment", input: aws.String(missing), expected: ""},
		{name: "valid comment", input: aws.String(valid), expected: "workflow comment"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, getDefinitionComment(tt.input))
		})
	}
}

func TestStepFunctionsConfigHelpers(t *testing.T) {
	t.Parallel()

	period := int32(300)
	encryption := &sfntypes.EncryptionConfiguration{
		KmsKeyId:                     aws.String("key-arn"),
		KmsDataKeyReusePeriodSeconds: &period,
		Type:                         sfntypes.EncryptionTypeCustomerManagedKmsKey,
	}
	logConfig := &sfntypes.LoggingConfiguration{
		Destinations: []sfntypes.LogDestination{{
			CloudWatchLogsLogGroup: &sfntypes.CloudWatchLogsLogGroup{LogGroupArn: aws.String("log-arn")},
		}},
		IncludeExecutionData: true,
		Level:                sfntypes.LogLevelAll,
	}
	tracing := &sfntypes.TracingConfiguration{Enabled: true}

	tests := []struct {
		name string
		got  any
		want any
	}{
		{name: "encryption key id nil", got: getEncryptionKeyID(nil), want: ""},
		{name: "encryption key id set", got: getEncryptionKeyID(encryption), want: "key-arn"},
		{name: "encryption reuse period nil", got: getEncryptionKeyReusePeriod(nil), want: ""},
		{name: "encryption reuse period empty", got: getEncryptionKeyReusePeriod(&sfntypes.EncryptionConfiguration{}), want: ""},
		{name: "encryption reuse period set", got: getEncryptionKeyReusePeriod(encryption), want: int32(300)},
		{name: "encryption type nil", got: getEncryptionType(nil), want: ""},
		{name: "encryption type set", got: getEncryptionType(encryption), want: string(sfntypes.EncryptionTypeCustomerManagedKmsKey)},
		{name: "log destination nil", got: getLogDestination(nil), want: ""},
		{name: "log destination empty", got: getLogDestination(&sfntypes.LoggingConfiguration{}), want: ""},
		{name: "log destination missing group", got: getLogDestination(&sfntypes.LoggingConfiguration{Destinations: []sfntypes.LogDestination{{}}}), want: ""},
		{name: "log destination set", got: getLogDestination(logConfig), want: "log-arn"},
		{name: "logging include execution data nil", got: getLoggingIncludeExecutionData(nil), want: ""},
		{name: "logging include execution data set", got: getLoggingIncludeExecutionData(logConfig), want: true},
		{name: "logging level nil", got: getLoggingLevel(nil), want: ""},
		{name: "logging level set", got: getLoggingLevel(logConfig), want: string(sfntypes.LogLevelAll)},
		{name: "tracing enabled nil", got: getTracingEnabled(nil), want: ""},
		{name: "tracing enabled set", got: getTracingEnabled(tracing), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.got)
		})
	}
}

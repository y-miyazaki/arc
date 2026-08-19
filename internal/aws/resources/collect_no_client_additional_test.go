package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/batch"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentity"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/firehose"
	"github.com/aws/aws-sdk-go-v2/service/glue"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/quicksight"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/transfer"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectors_Collect_NoClient_FirstGuard(t *testing.T) {
	t.Parallel()

	region := "us-west-2"
	tests := []struct {
		name    string
		call    func(context.Context, string) ([]Resource, error)
		wantErr error
	}{
		{name: "acm missing client", call: (&ACMCollector{clients: map[string]*acm.Client{}}).Collect, wantErr: ErrNoClientForRegion},
		{name: "apigateway missing v1 client", call: (&APIGatewayCollector{clientsV1: map[string]*apigateway.Client{}}).Collect, wantErr: ErrNoAPIGatewayV1Client},
		{name: "batch missing client", call: (&BatchCollector{clients: map[string]*batch.Client{}}).Collect, wantErr: ErrNoBatchClient},
		{name: "cloudformation missing client", call: (&CloudFormationCollector{clients: map[string]*cloudformation.Client{}}).Collect, wantErr: ErrNoClientForRegion},
		{name: "cloudwatch alarms missing client", call: (&CloudWatchAlarmsCollector{clients: map[string]*cloudwatch.Client{}}).Collect, wantErr: ErrNoClientForRegion},
		{name: "cloudwatch logs missing client", call: (&CloudWatchLogsCollector{clients: map[string]*cloudwatchlogs.Client{}}).Collect, wantErr: ErrNoClientForRegion},
		{name: "cognito identity pool missing client", call: (&CognitoIdentityPoolCollector{clients: map[string]*cognitoidentity.Client{}}).Collect, wantErr: ErrNoClientForRegion},
		{name: "cognito user pool missing client", call: (&CognitoUserPoolCollector{clients: map[string]*cognitoidentityprovider.Client{}}).Collect, wantErr: ErrNoClientForRegion},
		{name: "dynamodb missing client", call: (&DynamoDBCollector{clients: map[string]*dynamodb.Client{}}).Collect, wantErr: ErrNoClientForRegion},
		{name: "ec2 missing client", call: (&EC2Collector{clients: map[string]*ec2.Client{}}).Collect, wantErr: ErrNoClientForRegion},
		{name: "ecr missing client", call: (&ECRCollector{clients: map[string]*ecr.Client{}}).Collect, wantErr: ErrNoClientForRegion},
		{name: "efs missing client", call: (&EFSCollector{clients: map[string]*efs.Client{}}).Collect, wantErr: ErrNoClientForRegion},
		{name: "elasticache missing client", call: (&ElastiCacheCollector{clients: map[string]*elasticache.Client{}}).Collect, wantErr: ErrNoClientForRegion},
		{name: "glue missing client", call: (&GlueCollector{clients: map[string]*glue.Client{}}).Collect, wantErr: ErrNoClientForRegion},
		{name: "kms missing client", call: (&KMSCollector{clients: map[string]*kms.Client{}}).Collect, wantErr: ErrNoClientForRegion},
		{name: "lambda missing client", call: (&LambdaCollector{clients: map[string]*lambda.Client{}}).Collect, wantErr: ErrNoClientForRegion},
		{name: "quicksight missing client", call: (&QuickSightCollector{clients: map[string]*quicksight.Client{}}).Collect, wantErr: ErrNoClientForRegion},
		{name: "rds missing client", call: (&RDSCollector{clients: map[string]*rds.Client{}}).Collect, wantErr: ErrNoClientForRegion},
		{name: "redshift missing client", call: (&RedshiftCollector{clients: map[string]*redshift.Client{}}).Collect, wantErr: ErrNoClientForRegion},
		{name: "secretsmanager missing client", call: (&SecretsManagerCollector{clients: map[string]*secretsmanager.Client{}}).Collect, wantErr: ErrNoClientForRegion},
		{name: "ses missing client", call: (&SESCollector{clients: map[string]*sesv2.Client{}}).Collect, wantErr: ErrNoClientForRegion},
		{name: "sns missing client", call: (&SNSCollector{clients: map[string]*sns.Client{}}).Collect, wantErr: ErrNoClientForRegion},
		{name: "sqs missing client", call: (&SQSCollector{clients: map[string]*sqs.Client{}}).Collect, wantErr: ErrNoSQSClient},
		{name: "stepfunctions missing client", call: (&StepFunctionsCollector{clients: map[string]*sfn.Client{}}).Collect, wantErr: ErrNoClientForRegion},
		{name: "transferfamily missing client", call: (&TransferFamilyCollector{clients: map[string]*transfer.Client{}}).Collect, wantErr: ErrNoClientForRegion},
		{name: "vpc missing client", call: (&VPCCollector{clients: map[string]*ec2.Client{}}).Collect, wantErr: ErrNoClientForRegion},
		{name: "waf missing client", call: (&WAFCollector{wafClient: map[string]*wafv2.Client{}}).Collect, wantErr: ErrNoClientForRegion},
		{name: "ecs missing client", call: (&ECSCollector{clients: map[string]*ecs.Client{}}).Collect, wantErr: ErrNoClientForRegion},
		{name: "eventbridge missing client", call: (&EventBridgeCollector{ebClients: map[string]*eventbridge.Client{}}).Collect, wantErr: ErrNoClientForRegion},
		{name: "kinesis missing client", call: (&KinesisCollector{kinesisClients: map[string]*kinesis.Client{}}).Collect, wantErr: ErrNoClientForRegion},
		{name: "elb missing client", call: (&ELBCollector{elbClients: map[string]*elasticloadbalancingv2.Client{}}).Collect, wantErr: ErrNoClientForRegion},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := tt.call(context.Background(), region)
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.ErrorContains(t, err, region)
		})
	}
}

func TestCollectors_Collect_NoClient_SecondaryGuard(t *testing.T) {
	t.Parallel()

	region := "us-east-1"
	cfg := aws.Config{Region: region, Credentials: aws.AnonymousCredentials{}}

	tests := []struct {
		name         string
		collector    Collector
		cancel       bool
		wantErr      error
		wantContains string
	}{
		{
			name: "ecs missing eventbridge client",
			collector: &ECSCollector{
				clients: map[string]*ecs.Client{region: ecs.NewFromConfig(cfg)},
			},
			wantErr:      ErrNoClientForRegion,
			wantContains: "EventBridge",
		},
		{
			name: "eventbridge missing scheduler client",
			collector: &EventBridgeCollector{
				ebClients: map[string]*eventbridge.Client{region: eventbridge.NewFromConfig(cfg)},
			},
			wantErr:      ErrNoClientForRegion,
			wantContains: "Scheduler",
		},
		{
			name: "kinesis missing firehose client",
			collector: &KinesisCollector{
				kinesisClients: map[string]*kinesis.Client{region: kinesis.NewFromConfig(cfg)},
			},
			wantErr:      ErrNoClientForRegion,
			wantContains: "Firehose",
		},
		{
			name: "elb missing waf client",
			collector: &ELBCollector{
				elbClients: map[string]*elasticloadbalancingv2.Client{region: elasticloadbalancingv2.NewFromConfig(cfg)},
			},
			wantErr:      ErrNoClientForRegion,
			wantContains: "WAF",
		},
		{
			name: "elb missing ec2 client",
			collector: &ELBCollector{
				elbClients: map[string]*elasticloadbalancingv2.Client{region: elasticloadbalancingv2.NewFromConfig(cfg)},
				wafClients: map[string]*wafv2.Client{region: wafv2.NewFromConfig(cfg)},
			},
			wantErr:      ErrNoClientForRegion,
			wantContains: "EC2",
		},
		{
			name: "eventbridge and scheduler configured",
			collector: &EventBridgeCollector{
				ebClients:  map[string]*eventbridge.Client{region: eventbridge.NewFromConfig(cfg)},
				schClients: map[string]*scheduler.Client{region: scheduler.NewFromConfig(cfg)},
			},
			cancel: true,
		},
		{
			name: "kinesis and firehose configured",
			collector: &KinesisCollector{
				kinesisClients:  map[string]*kinesis.Client{region: kinesis.NewFromConfig(cfg)},
				firehoseClients: map[string]*firehose.Client{region: firehose.NewFromConfig(cfg)},
			},
			cancel: true,
		},
		{
			name: "apigateway missing v2 client",
			collector: &APIGatewayCollector{
				clientsV1: map[string]*apigateway.Client{region: apigateway.NewFromConfig(cfg)},
				clientsV2: map[string]*apigatewayv2.Client{},
			},
			wantErr:      ErrNoAPIGatewayV2Client,
			wantContains: region,
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			if tt.cancel {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			_, err := tt.collector.Collect(ctx, region)
			require.Error(t, err)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			}
			if tt.wantContains != "" {
				assert.ErrorContains(t, err, tt.wantContains)
			}
		})
	}
}

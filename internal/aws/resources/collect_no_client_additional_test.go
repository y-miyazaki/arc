package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
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
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/transfer"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectors_Collect_NoClient_FirstGuard(t *testing.T) {
	region := "us-west-2"
	tests := []struct {
		name string
		call func(context.Context, string) ([]Resource, error)
	}{
		{name: "ec2", call: (&EC2Collector{clients: map[string]*ec2.Client{}}).Collect},
		{name: "ecr", call: (&ECRCollector{clients: map[string]*ecr.Client{}}).Collect},
		{name: "efs", call: (&EFSCollector{clients: map[string]*efs.Client{}}).Collect},
		{name: "elasticache", call: (&ElastiCacheCollector{clients: map[string]*elasticache.Client{}}).Collect},
		{name: "glue", call: (&GlueCollector{clients: map[string]*glue.Client{}}).Collect},
		{name: "kms", call: (&KMSCollector{clients: map[string]*kms.Client{}}).Collect},
		{name: "lambda", call: (&LambdaCollector{clients: map[string]*lambda.Client{}}).Collect},
		{name: "quicksight", call: (&QuickSightCollector{clients: map[string]*quicksight.Client{}}).Collect},
		{name: "rds", call: (&RDSCollector{clients: map[string]*rds.Client{}}).Collect},
		{name: "redshift", call: (&RedshiftCollector{clients: map[string]*redshift.Client{}}).Collect},
		{name: "sns", call: (&SNSCollector{clients: map[string]*sns.Client{}}).Collect},
		{name: "transferfamily", call: (&TransferFamilyCollector{clients: map[string]*transfer.Client{}}).Collect},
		{name: "vpc", call: (&VPCCollector{clients: map[string]*ec2.Client{}}).Collect},
		{name: "waf", call: (&WAFCollector{wafClient: map[string]*wafv2.Client{}}).Collect},
		{name: "ecs", call: (&ECSCollector{clients: map[string]*ecs.Client{}}).Collect},
		{name: "eventbridge", call: (&EventBridgeCollector{ebClients: map[string]*eventbridge.Client{}}).Collect},
		{name: "kinesis", call: (&KinesisCollector{kinesisClients: map[string]*kinesis.Client{}}).Collect},
		{name: "elb", call: (&ELBCollector{elbClients: map[string]*elasticloadbalancingv2.Client{}}).Collect},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.call(context.Background(), region)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrNoClientForRegion)
			assert.ErrorContains(t, err, region)
		})
	}
}

func TestCollectors_Collect_NoClient_SecondaryGuard(t *testing.T) {
	region := "us-east-1"
	cfg := aws.Config{Region: region, Credentials: aws.AnonymousCredentials{}}

	t.Run("ecs missing eventbridge client", func(t *testing.T) {
		collector := &ECSCollector{
			clients: map[string]*ecs.Client{region: ecs.NewFromConfig(cfg)},
		}
		_, err := collector.Collect(context.Background(), region)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoClientForRegion)
		assert.ErrorContains(t, err, "EventBridge")
	})

	t.Run("eventbridge missing scheduler client", func(t *testing.T) {
		collector := &EventBridgeCollector{
			ebClients: map[string]*eventbridge.Client{region: eventbridge.NewFromConfig(cfg)},
		}
		_, err := collector.Collect(context.Background(), region)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoClientForRegion)
		assert.ErrorContains(t, err, "Scheduler")
	})

	t.Run("kinesis missing firehose client", func(t *testing.T) {
		collector := &KinesisCollector{
			kinesisClients: map[string]*kinesis.Client{region: kinesis.NewFromConfig(cfg)},
		}
		_, err := collector.Collect(context.Background(), region)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoClientForRegion)
		assert.ErrorContains(t, err, "Firehose")
	})

	t.Run("elb missing waf client", func(t *testing.T) {
		collector := &ELBCollector{
			elbClients: map[string]*elasticloadbalancingv2.Client{region: elasticloadbalancingv2.NewFromConfig(cfg)},
		}
		_, err := collector.Collect(context.Background(), region)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoClientForRegion)
		assert.ErrorContains(t, err, "WAF")
	})

	t.Run("elb missing ec2 client", func(t *testing.T) {
		collector := &ELBCollector{
			elbClients: map[string]*elasticloadbalancingv2.Client{region: elasticloadbalancingv2.NewFromConfig(cfg)},
			wafClients: map[string]*wafv2.Client{region: wafv2.NewFromConfig(cfg)},
		}
		_, err := collector.Collect(context.Background(), region)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoClientForRegion)
		assert.ErrorContains(t, err, "EC2")
	})

	t.Run("eventbridge and scheduler configured", func(t *testing.T) {
		collector := &EventBridgeCollector{
			ebClients:  map[string]*eventbridge.Client{region: eventbridge.NewFromConfig(cfg)},
			schClients: map[string]*scheduler.Client{region: scheduler.NewFromConfig(cfg)},
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := collector.Collect(ctx, region)
		require.Error(t, err)
	})

	t.Run("kinesis and firehose configured", func(t *testing.T) {
		collector := &KinesisCollector{
			kinesisClients:  map[string]*kinesis.Client{region: kinesis.NewFromConfig(cfg)},
			firehoseClients: map[string]*firehose.Client{region: firehose.NewFromConfig(cfg)},
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := collector.Collect(ctx, region)
		require.Error(t, err)
	})
}

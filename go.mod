// TODO: Upgrade to Go 1.25.5 to fix vulnerabilities GO-2025-4175 and GO-2025-4155 in crypto/x509
module github.com/y-miyazaki/arc

go 1.25.4

require (
	github.com/aws/aws-sdk-go-v2 v1.41.1
	github.com/aws/aws-sdk-go-v2/config v1.32.7
	github.com/aws/aws-sdk-go-v2/service/acm v1.37.19
	github.com/aws/aws-sdk-go-v2/service/apigateway v1.38.4
	github.com/aws/aws-sdk-go-v2/service/apigatewayv2 v1.33.5
	github.com/aws/aws-sdk-go-v2/service/batch v1.58.12
	github.com/aws/aws-sdk-go-v2/service/cloudformation v1.71.5
	github.com/aws/aws-sdk-go-v2/service/cloudfront v1.59.0
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.53.1
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.63.1
	github.com/aws/aws-sdk-go-v2/service/cognitoidentity v1.33.18
	github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider v1.57.19
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.54.0
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.283.0
	github.com/aws/aws-sdk-go-v2/service/ecr v1.55.1
	github.com/aws/aws-sdk-go-v2/service/ecs v1.71.0
	github.com/aws/aws-sdk-go-v2/service/efs v1.41.10
	github.com/aws/aws-sdk-go-v2/service/elasticache v1.51.9
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.54.6
	github.com/aws/aws-sdk-go-v2/service/eventbridge v1.45.18
	github.com/aws/aws-sdk-go-v2/service/firehose v1.42.9
	github.com/aws/aws-sdk-go-v2/service/glue v1.136.1
	github.com/aws/aws-sdk-go-v2/service/iam v1.53.2
	github.com/aws/aws-sdk-go-v2/service/kinesis v1.42.10
	github.com/aws/aws-sdk-go-v2/service/kms v1.49.5
	github.com/aws/aws-sdk-go-v2/service/lambda v1.87.1
	github.com/aws/aws-sdk-go-v2/service/quicksight v1.101.0
	github.com/aws/aws-sdk-go-v2/service/rds v1.114.0
	github.com/aws/aws-sdk-go-v2/service/redshift v1.62.0
	github.com/aws/aws-sdk-go-v2/service/route53 v1.62.1
	github.com/aws/aws-sdk-go-v2/service/s3 v1.95.1
	github.com/aws/aws-sdk-go-v2/service/scheduler v1.17.18
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.41.1
	github.com/aws/aws-sdk-go-v2/service/sesv2 v1.59.1
	github.com/aws/aws-sdk-go-v2/service/sns v1.39.11
	github.com/aws/aws-sdk-go-v2/service/sqs v1.42.21
	github.com/aws/aws-sdk-go-v2/service/sts v1.41.6
	github.com/aws/aws-sdk-go-v2/service/transfer v1.68.6
	github.com/aws/aws-sdk-go-v2/service/wafv2 v1.70.7
	github.com/stretchr/testify v1.11.1
	github.com/urfave/cli/v2 v2.27.7
)

require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.4 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.19.7 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.4 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.11.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.13 // indirect
	github.com/aws/smithy-go v1.24.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.7 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/stretchr/objx v0.5.3 // indirect
	github.com/xrash/smetrics v0.0.0-20250705151800-55b8f293f342 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

package resources

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectors_Collect_SkipsNonGlobalRegion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func(context.Context, string) ([]Resource, error)
	}{
		{name: "cloudfront skips non-global region", call: (&CloudFrontCollector{}).Collect},
		{name: "iam policy skips non-global region", call: (&IAMPolicyCollector{}).Collect},
		{name: "iam role skips non-global region", call: (&IAMRoleCollector{}).Collect},
		{name: "iam user group skips non-global region", call: (&IAMUserGroupCollector{}).Collect},
		{name: "route53 skips non-global region", call: (&Route53Collector{}).Collect},
		{name: "s3 bucket skips non-global region", call: (&S3BucketCollector{}).Collect},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.call(context.Background(), "ap-northeast-1")
			require.NoError(t, err)
			assert.Nil(t, got)
		})
	}
}
